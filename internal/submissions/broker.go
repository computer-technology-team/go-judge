package submissions

import (
	"context"
	"errors"
	"fmt"
	"io"
	"iter"
	"log/slog"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/samber/lo"
	"google.golang.org/grpc"

	runnerPb "github.com/computer-technology-team/go-judge/api/gen/runner"
	"github.com/computer-technology-team/go-judge/config"
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

	jobTimeout time.Duration
}

func NewBroker(brokerConfig config.BrokerConfig, runnerClient runnerPb.RunnerClient, querier storage.Querier, pool *pgxpool.Pool) Broker {
	return &broker{
		runnerClient: runnerClient,
		pool:         pool,
		querier:      querier,

		jobsChan:   make(chan submissionEvaluation, jobsChannelBufferSize),
		workerCnt:  brokerConfig.Workers,
		jobTimeout: brokerConfig.JobTimeout,

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

	b.addJob(ctx, job)
}

func (b *broker) addJob(ctx context.Context, job submissionEvaluation) {
	select {
	case b.jobsChan <- job:
		slog.Info("added submission to evaluation queue", "submission_id", job.submission.ID)
	default:
		slog.Error("evaluation queue is full, dropping submission", "submission_id", job.submission.ID)
		_, err := b.querier.UpdateSubmissionStatus(ctx, b.pool, storage.UpdateSubmissionStatusParams{
			ID:     job.submission.ID,
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
	b.workerWg.Wait()
	b.workerCancelFunc()
}

func (b *broker) startWorker(ctx context.Context) {
	b.workerWg.Add(1)
	go func(ctx context.Context) {
		defer b.workerWg.Done()
		for {
			select {
			case job, ok := <-b.jobsChan:
				if !ok {
					return
				}
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

					if job.submission.Retries >= 3 {
						continue
					}

					b.addJob(ctx, job)
				}
			case <-ctx.Done():
				return
			}
		}
	}(ctx)
}

func (b *broker) handleJob(ctx context.Context, job submissionEvaluation) (storage.Submission, error) {
	ctx, cancel := context.WithTimeout(ctx, b.jobTimeout)
	defer cancel()

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
	}()

	for updateEvent, err := range streamToIter(stream) {
		if err != nil {
			return job.submission, err
		}

		slog.Info("received update event", "status", updateEvent.GetStatus())

		// Handle the update event based on its status
		updatedSubmission, err := b.handleStatusUpdate(ctx, job, updateEvent)
		if err != nil {
			return job.submission, err
		}

		job.submission = updatedSubmission

		// For terminal states, return immediately
		if isTerminalState(updateEvent.GetStatus()) {
			return job.submission, getErrorForStatus(updateEvent.GetStatus())
		}
	}

	return job.submission, nil
}

// isTerminalState determines if a submission status is a terminal state
func isTerminalState(status runnerPb.SubmissionStatusUpdate_Status) bool {
	switch status {
	case runnerPb.SubmissionStatusUpdate_PENDING,
		runnerPb.SubmissionStatusUpdate_RUNNING:
		return false
	default:
		return true
	}
}

// getErrorForStatus returns the appropriate error for terminal states
func getErrorForStatus(status runnerPb.SubmissionStatusUpdate_Status) error {
	if status == runnerPb.SubmissionStatusUpdate_INTERNAL_ERROR {
		return errInternalErrorInEvaluation
	}
	return nil
}

// handleStatusUpdate processes a status update and returns the updated submission
func (b *broker) handleStatusUpdate(ctx context.Context, job submissionEvaluation, updateEvent *runnerPb.SubmissionStatusUpdate) (storage.Submission, error) {
	switch updateEvent.GetStatus() {
	case runnerPb.SubmissionStatusUpdate_RUNNING:
		return b.updateSubmissionStatus(ctx, b.pool, job.submission, storage.SubmissionStatusRUNNING,
			fmt.Sprintf("%d/%d", updateEvent.TestsCompleted, updateEvent.TotalTests), "running")

	case runnerPb.SubmissionStatusUpdate_COMPILATION_ERROR:
		return b.updateSubmissionStatus(ctx, b.pool, job.submission, storage.SubmissionStatusCOMPILATIONERROR,
			updateEvent.StatusMessage, "compilation error")

	case runnerPb.SubmissionStatusUpdate_ACCEPTED:
		return b.acceptSubmission(ctx, job.submission, storage.SubmissionStatusACCEPTED,
			fmt.Sprintf("Passed all %d test cases", updateEvent.TotalTests), "accepted")

	case runnerPb.SubmissionStatusUpdate_INTERNAL_ERROR:
		return b.handleInternalError(ctx, job.submission)

	case runnerPb.SubmissionStatusUpdate_MEMORY_LIMIT_EXCEEDED:
		return b.updateSubmissionStatus(ctx, b.pool, job.submission, storage.SubmissionStatusMEMORYLIMITEXCEEDED,
			fmt.Sprintf("Memory limit exceeded (%d KB)", job.problem.MemoryLimitKb), "memory limit exceeded")

	case runnerPb.SubmissionStatusUpdate_PENDING:
		return b.updateSubmissionStatus(ctx, b.pool, job.submission, storage.SubmissionStatusPENDING,
			"Waiting for evaluation", "pending")

	case runnerPb.SubmissionStatusUpdate_RUNTIME_ERROR:
		return b.updateSubmissionStatus(ctx, b.pool, job.submission, storage.SubmissionStatusRUNTIMEERROR,
			fmt.Sprintf("Runtime error on test case %d: %s", updateEvent.TestsCompleted+1, updateEvent.GetStatusMessage()), "runtime error")

	case runnerPb.SubmissionStatusUpdate_TIME_LIMIT_EXCEEDED:
		return b.updateSubmissionStatus(ctx, b.pool, job.submission, storage.SubmissionStatusTIMELIMITEXCEEDED,
			fmt.Sprintf("Time limit exceeded (%d ms) on test case %d", job.problem.TimeLimitMs, updateEvent.TestsCompleted+1), "time limit exceeded")

	case runnerPb.SubmissionStatusUpdate_WRONG_ANSWER:
		return b.updateSubmissionStatus(ctx, b.pool, job.submission, storage.SubmissionStatusWRONGANSWER,
			fmt.Sprintf("Wrong answer on test case %d", updateEvent.TestsCompleted+1), "wrong answer")

	default:
		slog.Error("unexpected update event", "status", updateEvent.GetStatus())
		return job.submission, errors.New("unexpected update event")
	}
}

// updateSubmissionStatus is a helper function to update the submission status
func (b *broker) updateSubmissionStatus(ctx context.Context, db storage.DBTX, submission storage.Submission,
	status storage.SubmissionStatus, message string, logStatus string) (storage.Submission, error) {

	updatedSubmission, err := b.querier.UpdateSubmissionStatus(ctx, db, storage.UpdateSubmissionStatusParams{
		ID:     submission.ID,
		Status: status,
		Message: pgtype.Text{
			Valid:  true,
			String: message,
		},
	})

	if err != nil {
		slog.Error("could not update submission status", "status", logStatus, "error", err)
		return submission, fmt.Errorf("could not update submission status: %w", err)
	}

	return updatedSubmission, nil
}

func (b *broker) acceptSubmission(ctx context.Context, submission storage.Submission,
	status storage.SubmissionStatus, message string, logStatus string) (storage.Submission, error) {

	tx, err := b.pool.Begin(ctx)
	if err != nil {
		slog.Error("could not begin transaction", "status", logStatus, "error", err)
		return submission, err
	}
	defer func(ctx context.Context, tx pgx.Tx) {
		err := tx.Rollback(ctx)
		if err != nil && !errors.Is(err, pgx.ErrTxClosed) {
			slog.Error("could not rollback transaction", "error", err)
		}
	}(ctx, tx)

	updatedSubmission, err := b.querier.UpdateSubmissionStatus(ctx, tx, storage.UpdateSubmissionStatusParams{
		ID:     submission.ID,
		Status: status,
		Message: pgtype.Text{
			Valid:  true,
			String: message,
		},
	})
	if err != nil {
		slog.Error("could not update submission status", "status", logStatus, "error", err)
		return submission, fmt.Errorf("could not update submission status: %w", err)
	}

	err = b.querier.IncreaseUserSolves(ctx, tx, submission.UserID)
	if err != nil {
		slog.Error("could not update user solves", "status", logStatus, "error", err)
		return submission, fmt.Errorf("could not update user solves: %w", err)
	}

	err = tx.Commit(ctx)
	if err != nil {
		slog.Error("could not commit transaction", "status", logStatus, "error", err)
		return submission, fmt.Errorf("could not commit transaction: %w", err)
	}

	return updatedSubmission, nil
}

// handleInternalError handles the internal error case which requires a transaction
func (b *broker) handleInternalError(ctx context.Context, submission storage.Submission) (storage.Submission, error) {
	tx, err := b.pool.Begin(ctx)
	if err != nil {
		slog.Error("could not begin update transaction", "error", err, "status", "internal error")
		return submission, fmt.Errorf("could not begin update submission transaction: %w", err)
	}

	defer func(ctx context.Context, tx pgx.Tx) {
		err := tx.Rollback(ctx)
		if err != nil && !errors.Is(err, pgx.ErrTxClosed) {
			slog.Error("could not rollback transaction")
		}
	}(ctx, tx)

	updatedSubmission, err := b.querier.UpdateSubmissionStatus(ctx, tx, storage.UpdateSubmissionStatusParams{
		ID:     submission.ID,
		Status: storage.SubmissionStatusINTERNALERROR,
		Message: pgtype.Text{
			Valid:  true,
			String: "Internal error occurred during evaluation",
		},
	})
	if err != nil {
		slog.Error("could not update submission status", "status", "internal error", "error", err)
		return submission, fmt.Errorf("could not update submission status: %w", err)
	}

	updatedSubmission, err = b.increaseSubmissionRetry(ctx, updatedSubmission, tx)
	if err != nil {
		return submission, fmt.Errorf("could not increase submission retry: %w", err)
	}

	err = tx.Commit(ctx)
	if err != nil {
		return submission, fmt.Errorf("could not commit update and increase retry transaction: %w", err)
	}

	return updatedSubmission, nil
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
) iter.Seq2[*runnerPb.SubmissionStatusUpdate, error] {
	return func(yield func(*runnerPb.SubmissionStatusUpdate, error) bool) {
		for {
			updateEvent, err := stream.Recv()
			if err != nil {
				if errors.Is(err, io.EOF) {
					return
				}
				slog.Error("error happened in receiving update event", "error", err)
				yield(nil, err)
				return
			}

			if !yield(updateEvent, nil) {
				return
			}
		}
	}
}
