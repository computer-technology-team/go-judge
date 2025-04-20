package submissions

import (
	"context"
	"errors"
	"fmt"
	"io"
	"iter"
	"log/slog"
	"sync"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/samber/lo"
	"google.golang.org/grpc"

	runnerPb "github.com/computer-technology-team/go-judge/api/gen/runner"
	"github.com/computer-technology-team/go-judge/internal/storage"
)

const jobsChannelBufferSize = 100

var errInternalErrorInEvaluation = errors.New("internal error happened while processing the job")

type Broker interface {
	AddSubmissionEvaluation(storage.Submission)
	StartWorkers(ctx context.Context)
	StopWorkers()
}

type submissionEvaluation struct {
	submission storage.Submission
	problem    storage.Problem
	testCases  []storage.TestCase
}

type broker struct {
	runnerClient runnerPb.RunnerClient
	jobsChan     chan submissionEvaluation
	pool         *pgxpool.Pool
	querier      storage.Querier

	workerCnt        int
	workerWg         sync.WaitGroup
	workerCancelFunc context.CancelFunc
	startOnce        sync.Once
}

func NewBroker(workerCnt int, runnerClient runnerPb.RunnerClient, querier storage.Querier, pool *pgxpool.Pool) Broker {
	return &broker{
		runnerClient: runnerClient,
		pool:         pool,
		querier:      querier,

		jobsChan:  make(chan submissionEvaluation, jobsChannelBufferSize),
		workerCnt: workerCnt,

		workerWg:         sync.WaitGroup{},
		startOnce:        sync.Once{},
		workerCancelFunc: nil,
	}
}

// AddSubmissionEvaluation implements Broker.
func (b *broker) AddSubmissionEvaluation(submission storage.Submission) {
	ctx := context.Background()

	// Get the problem details
	problem, err := b.querier.GetProblemByID(ctx, b.pool, submission.ProblemID)
	if err != nil {
		slog.Error("could not get problem for submission", "submission_id", submission.ID, "problem_id", submission.ProblemID, "error", err)
		return
	}

	// Get the test cases for the problem
	testCases, err := b.querier.GetTestCasesByProblemID(ctx, b.pool, submission.ProblemID)
	if err != nil {
		slog.Error("could not get test cases for problem", "submission_id", submission.ID, "problem_id", submission.ProblemID, "error", err)
		return
	}

	if len(testCases) == 0 {
		slog.Error("no test cases found for problem", "submission_id", submission.ID, "problem_id", submission.ProblemID)
		_, err := b.querier.UpdateSubmissionStatus(ctx, b.pool, storage.UpdateSubmissionStatusParams{
			ID:     submission.ID,
			Status: storage.SubmissionStatusINTERNALERROR,
			Message: pgtype.Text{
				Valid:  true,
				String: "No test cases found for this problem",
			},
		})
		if err != nil {
			slog.Error("could not update submission status", "error", err)
		}
		return
	}

	// Create the job and send it to the channel
	job := submissionEvaluation{
		submission: submission,
		problem:    problem,
		testCases:  testCases,
	}

	select {
	case b.jobsChan <- job:
		slog.Info("added submission to evaluation queue", "submission_id", submission.ID)
	default:
		slog.Error("evaluation queue is full, dropping submission", "submission_id", submission.ID)
		_, err := b.querier.UpdateSubmissionStatus(ctx, b.pool, storage.UpdateSubmissionStatusParams{
			ID:     submission.ID,
			Status: storage.SubmissionStatusINTERNALERROR,
			Message: pgtype.Text{
				Valid:  true,
				String: "Evaluation queue is full, please try again later",
			},
		})
		if err != nil {
			slog.Error("could not update submission status", "error", err)
		}
	}
}

// StartWorkers implements Broker.
func (b *broker) StartWorkers(ctx context.Context) {
	b.startOnce.Do(func() {
		workerCtx, workerCancelFunc := context.WithCancel(ctx)
		b.workerCancelFunc = workerCancelFunc
		for range b.workerCnt {
			b.startWorker(workerCtx)
		}
	})
}

// StopWorkers implements Broker.
func (b *broker) StopWorkers() {
	if b.workerCancelFunc == nil {
		return
	}
	close(b.jobsChan)
	b.workerCancelFunc()
	b.workerWg.Wait()
}

func (b *broker) startWorker(ctx context.Context) {
	b.workerWg.Add(1)
	go func(ctx context.Context) {
		defer b.workerWg.Done()
		for {
			select {
			case job := <-b.jobsChan:
				var err error
				job.submission, err = b.handleJob(ctx, job)
				if err != nil {
					var err error
					if !errors.Is(err, errInternalErrorInEvaluation) {
						job.submission, err = b.increaseSubmissionRetry(ctx, job.submission, b.pool)
					} else {
						job.submission, err = b.querier.UpdateSubmissionStatus(ctx, b.pool, storage.UpdateSubmissionStatusParams{
							ID:      job.submission.ID,
							Status:  storage.SubmissionStatusINQUEUE,
							Message: job.submission.Message,
						})
					}
					if err != nil {
						slog.Error("could not handle retry", "error", err)
					}

					b.jobsChan <- job
				}
			case <-ctx.Done():
				return
			}
		}
	}(ctx)
}

func (b *broker) handleJob(ctx context.Context, job submissionEvaluation) (storage.Submission, error) {
	stream, err := b.runnerClient.ExecuteSubmission(ctx, &runnerPb.SubmissionRequest{
		SubmissionId:  job.submission.ID.String(),
		Code:          job.submission.SolutionCode,
		TimeLimitMs:   job.problem.TimeLimitMs,
		MemoryLimitKb: job.problem.MemoryLimitKb,
		TestCases: lo.Map(job.testCases, func(tc storage.TestCase, _ int) *runnerPb.SubmissionRequest_TestCase {
			return tc.ToProto()
		}),
	})
	if err != nil {
		slog.Error("could not start execute submission stream", "error", err)
		return job.submission, fmt.Errorf("could not start grpc stream: %w", err)
	}

	defer func() {
		_ = stream.CloseSend()
		stream.Context().Done()
	}()

	for updateEvent := range streamToIter(stream) {
		switch updateEvent.GetStatus() {
		case runnerPb.SubmissionStatusUpdate_RUNNING:
			updatedSubmission, err := b.querier.UpdateSubmissionStatus(ctx, b.pool, storage.UpdateSubmissionStatusParams{
				ID:     job.submission.ID,
				Status: storage.SubmissionStatusRUNNING,
				Message: pgtype.Text{
					Valid:  true,
					String: fmt.Sprintf("%d/%d", updateEvent.TestsCompleted, updateEvent.TotalTests),
				},
			})
			if err != nil {
				slog.Error("could not update submission status", "status", "running", "error", err)
				return job.submission, fmt.Errorf("could not update submission status: %w", err)
			}

			job.submission = updatedSubmission
		case runnerPb.SubmissionStatusUpdate_COMPILATION_ERROR:
			updatedSubmission, err := b.querier.UpdateSubmissionStatus(ctx, b.pool, storage.UpdateSubmissionStatusParams{
				ID:     job.submission.ID,
				Status: storage.SubmissionStatusCOMPILATIONERROR,
				Message: pgtype.Text{
					Valid:  true,
					String: "compile failed",
				},
			})
			if err != nil {
				slog.Error("could not update submission status", "status", "compilation error", "error", err)
				return job.submission, fmt.Errorf("could not update submission status: %w", err)
			}

			job.submission = updatedSubmission

			return job.submission, nil
		case runnerPb.SubmissionStatusUpdate_ACCEPTED:
			updatedSubmission, err := b.querier.UpdateSubmissionStatus(ctx, b.pool, storage.UpdateSubmissionStatusParams{
				ID:     job.submission.ID,
				Status: storage.SubmissionStatusACCEPTED,
				Message: pgtype.Text{
					Valid:  true,
					String: fmt.Sprintf("Passed all %d test cases", updateEvent.TotalTests),
				},
			})
			if err != nil {
				slog.Error("could not update submission status", "status", "accepted", "error", err)
				return job.submission, fmt.Errorf("could not update submission status: %w", err)
			}

			job.submission = updatedSubmission
			return job.submission, nil
		case runnerPb.SubmissionStatusUpdate_INTERNAL_ERROR:
			tx, err := b.pool.Begin(ctx)
			if err != nil {
				slog.Error("could not begin update transaction", "error", err, "status", "internal error")
				return job.submission, fmt.Errorf("could not begin update submission transaction: %w", err)
			}

			defer func(ctx context.Context, tx pgx.Tx) {
				err := tx.Rollback(ctx)
				if err != nil && !errors.Is(err, pgx.ErrTxClosed) {
					slog.Error("could not rollback transaction")
				}
			}(ctx, tx)

			updatedSubmission, err := b.querier.UpdateSubmissionStatus(ctx, tx, storage.UpdateSubmissionStatusParams{
				ID:     job.submission.ID,
				Status: storage.SubmissionStatusINTERNALERROR,
				Message: pgtype.Text{
					Valid:  true,
					String: "Internal error occurred during evaluation",
				},
			})
			if err != nil {
				slog.Error("could not update submission status", "status", "internal error", "error", err)

				return job.submission, fmt.Errorf("could not update submission status: %w", err)
			}

			job.submission, err = b.increaseSubmissionRetry(ctx, updatedSubmission, tx)
			return job.submission, errors.Join(err)

		case runnerPb.SubmissionStatusUpdate_MEMORY_LIMIT_EXCEEDED:
			updatedSubmission, err := b.querier.UpdateSubmissionStatus(ctx, b.pool, storage.UpdateSubmissionStatusParams{
				ID:     job.submission.ID,
				Status: storage.SubmissionStatusMEMORYLIMITEXCEEDED,
				Message: pgtype.Text{
					Valid:  true,
					String: fmt.Sprintf("Memory limit exceeded (%d KB)", job.problem.MemoryLimitKb),
				},
			})
			if err != nil {
				slog.Error("could not update submission status", "status", "memory limit exceeded", "error", err)
				return job.submission, fmt.Errorf("could not update submission status: %w", err)
			}

			job.submission = updatedSubmission
			return job.submission, nil
		case runnerPb.SubmissionStatusUpdate_PENDING:
			updatedSubmission, err := b.querier.UpdateSubmissionStatus(ctx, b.pool, storage.UpdateSubmissionStatusParams{
				ID:     job.submission.ID,
				Status: storage.SubmissionStatusPENDING,
				Message: pgtype.Text{
					Valid:  true,
					String: "Waiting for evaluation",
				},
			})
			if err != nil {
				slog.Error("could not update submission status", "status", "pending", "error", err)
				return job.submission, fmt.Errorf("could not update submission status: %w", err)
			}
			job.submission = updatedSubmission
		case runnerPb.SubmissionStatusUpdate_RUNTIME_ERROR:
			updatedSubmission, err := b.querier.UpdateSubmissionStatus(ctx, b.pool, storage.UpdateSubmissionStatusParams{
				ID:     job.submission.ID,
				Status: storage.SubmissionStatusRUNTIMEERROR,
				Message: pgtype.Text{
					Valid:  true,
					String: fmt.Sprintf("Runtime error on test case %d", updateEvent.TestsCompleted+1),
				},
			})
			if err != nil {
				slog.Error("could not update submission status", "status", "runtime error", "error", err)
				return job.submission, fmt.Errorf("could not update submission status: %w", err)
			}

			job.submission = updatedSubmission
			return job.submission, nil
		case runnerPb.SubmissionStatusUpdate_TIME_LIMIT_EXCEEDED:
			updatedSubmission, err := b.querier.UpdateSubmissionStatus(ctx, b.pool, storage.UpdateSubmissionStatusParams{
				ID:     job.submission.ID,
				Status: storage.SubmissionStatusTIMELIMITEXCEEDED,
				Message: pgtype.Text{
					Valid:  true,
					String: fmt.Sprintf("Time limit exceeded (%d ms) on test case %d", job.problem.TimeLimitMs, updateEvent.TestsCompleted+1),
				},
			})
			if err != nil {
				slog.Error("could not update submission status", "status", "time limit exceeded", "error", err)
				return job.submission, fmt.Errorf("could not update submission status: %w", err)
			}

			job.submission = updatedSubmission
			return job.submission, nil
		case runnerPb.SubmissionStatusUpdate_WRONG_ANSWER:
			updatedSubmission, err := b.querier.UpdateSubmissionStatus(ctx, b.pool, storage.UpdateSubmissionStatusParams{
				ID:     job.submission.ID,
				Status: storage.SubmissionStatusWRONGANSWER,
				Message: pgtype.Text{
					Valid:  true,
					String: fmt.Sprintf("Wrong answer on test case %d", updateEvent.TestsCompleted+1),
				},
			})
			if err != nil {
				slog.Error("could not update submission status", "status", "wrong answer", "error", err)
				return job.submission, fmt.Errorf("could not update submission status: %w", err)
			}
			job.submission = updatedSubmission
			return job.submission, nil
		default:
			slog.Error("unexpected update event", "status", updateEvent.GetStatus())
			return job.submission, errors.New("unexpected update event")
		}
	}

	return job.submission, nil
}

func (b *broker) increaseSubmissionRetry(ctx context.Context, updatedSubmission storage.Submission, tx storage.DBTX) (storage.Submission, error) {
	updatedSubmission, err := b.querier.RetrySubmissionDueToInternalError(ctx, tx, updatedSubmission.ID)
	if err != nil {
		slog.Error("could not update submission retries", "status", "internal error", "error", err)
		return storage.Submission{}, fmt.Errorf("could not update submission retries: %w", err)
	}
	return updatedSubmission, nil
}

func streamToIter(
	stream grpc.ServerStreamingClient[runnerPb.SubmissionStatusUpdate],
) iter.Seq[*runnerPb.SubmissionStatusUpdate] {
	return func(yield func(*runnerPb.SubmissionStatusUpdate) bool) {
		for {
			updateEvent, err := stream.Recv()
			if err != nil {
				if errors.Is(err, io.EOF) {
					return
				}
				slog.Error("error happened in receiving update event", "error", err)
				return
			}

			if !yield(updateEvent) {
				return
			}
		}
	}
}
