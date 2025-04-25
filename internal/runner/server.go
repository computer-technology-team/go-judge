package runner

import (
	"context"
	"errors"
	"log/slog"

	"golang.org/x/sync/semaphore"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	runnerPb "github.com/computer-technology-team/go-judge/api/gen/runner"
	"github.com/samber/lo"
)

type runnerServer struct {
	runnerPb.UnimplementedRunnerServer

	codeEvaluator   *CodeEvaluator
	resourceLimiter *semaphore.Weighted
}

func NewRunnerServer(ctx context.Context, runnerCnt int, evaluator *CodeEvaluator) (runnerPb.RunnerServer, error) {

	cpuCnt, err := evaluator.GetCpuCount(ctx)
	if err != nil {
		return nil, err
	}

	cpuAllowance := int64(max(1, (cpuCnt-2)/runnerCnt))

	slog.Info("creating runner server", "cpu_allowance", cpuAllowance, "runner_cnt", runnerCnt)

	return &runnerServer{
		codeEvaluator:   evaluator,
		resourceLimiter: semaphore.NewWeighted(cpuAllowance),
	}, nil
}

func (rs *runnerServer) ExecuteSubmission(
	request *runnerPb.SubmissionRequest,
	stream grpc.ServerStreamingServer[runnerPb.SubmissionStatusUpdate],
) error {
	logger := slog.With("memory_limit", request.GetMemoryLimitKb(), "timelimit", request.GetTimeLimitMs(),
		"code", request.GetCode(), "submission_id", request.GetSubmissionId())
	logger.Info("recieved request")

	err := rs.resourceLimiter.Acquire(stream.Context(), 1)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			logger.Warn("failed to acquire resource in time")
			return nil
		}
		logger.Error("failed to acquire resource", "error", err)
		return status.Error(codes.Internal, "could not acquire resource")
	}
	defer rs.resourceLimiter.Release(1)

	err = stream.Send(&runnerPb.SubmissionStatusUpdate{
		SubmissionId:   request.GetSubmissionId(),
		Status:         runnerPb.SubmissionStatusUpdate_RUNNING,
		TestsCompleted: 0,
		TotalTests:     int32(len(request.GetTestCases())),
		MaxTimeSpentMs: 0,
	})
	if err != nil {
		logger.Error("could not send update in stream", "error", err)
		return status.Error(codes.Internal, "could not send first message in stream")
	}

	err = rs.codeEvaluator.BuildCodeBinary(stream.Context(), request.GetSubmissionId(), request.GetCode())
	if err != nil {
		if buildErr, ok := lo.ErrorsAs[*BuildError](err); ok {
			logger.Warn("compilation failed", "error", err)
			stream.Send(&runnerPb.SubmissionStatusUpdate{
				SubmissionId:   request.SubmissionId,
				Status:         runnerPb.SubmissionStatusUpdate_COMPILATION_ERROR,
				StatusMessage:  buildErr.Logs,
				TestsCompleted: 0,
				TotalTests:     int32(len(request.TestCases)),
				MaxTimeSpentMs: 0,
			})
			return nil
		} else {
			logger.Error("unexpected error in building code volume", "error", err)
			stream.Send(&runnerPb.SubmissionStatusUpdate{
				SubmissionId:   request.SubmissionId,
				Status:         runnerPb.SubmissionStatusUpdate_INTERNAL_ERROR,
				TestsCompleted: 0,
				TotalTests:     int32(len(request.TestCases)),
				MaxTimeSpentMs: 0,
			})
			return nil
		}
	}

	var maxTimeSpendMs int64
	for i, tc := range request.GetTestCases() {
		logger.Info("running test case", "i", i)
		err := stream.Send(&runnerPb.SubmissionStatusUpdate{
			SubmissionId:   request.GetSubmissionId(),
			Status:         runnerPb.SubmissionStatusUpdate_RUNNING,
			TestsCompleted: int32(i),
			TotalTests:     int32(len(request.GetTestCases())),
			MaxTimeSpentMs: int64(maxTimeSpendMs),
		})
		if err != nil {
			logger.Error("could not send update in stream", "error", err)
			return status.Error(codes.Internal, "could not send subsequent messages in stream")
		}

		runStatus, err := rs.codeEvaluator.RunTestCase(stream.Context(), request.GetSubmissionId(), tc.GetInput(), tc.GetOutput(), request.GetTimeLimitMs(), request.GetMemoryLimitKb())
		if err != nil {
			if errors.Is(err, ErrExecutionFailed) {
				logger.Info("exection failed", "error", err, "exit_code", runStatus.ExitCode, "stdout", runStatus.Stdout,
					"stderr", runStatus.Stderr)

				st, ok := exitCodeToStatus[runStatus.ExitCode]
				if !ok {
					st = runnerPb.SubmissionStatusUpdate_INTERNAL_ERROR
				}

				stream.Send(&runnerPb.SubmissionStatusUpdate{
					SubmissionId:   request.GetSubmissionId(),
					Status:         st,
					StatusMessage:  runStatus.Stdout,
					TestsCompleted: int32(i),
					TotalTests:     int32(len(request.GetTestCases())),
					MaxTimeSpentMs: maxTimeSpendMs,
				})

			} else {
				logger.Error("run test case failed", "error", err, "exit_code", runStatus.ExitCode, "stdout", runStatus.Stdout,
					"stderr", runStatus.Stderr)
				stream.Send(&runnerPb.SubmissionStatusUpdate{
					SubmissionId:   request.GetSubmissionId(),
					Status:         runnerPb.SubmissionStatusUpdate_INTERNAL_ERROR,
					TestsCompleted: int32(i),
					TotalTests:     int32(len(request.GetTestCases())),
					MaxTimeSpentMs: maxTimeSpendMs,
				})
			}
			return nil
		}

		maxTimeSpendMs = max(runStatus.ExecutionTime.Milliseconds(), maxTimeSpendMs)
	}

	logger.Info("submission accepted")

	err = stream.Send(&runnerPb.SubmissionStatusUpdate{
		SubmissionId:   request.GetSubmissionId(),
		Status:         runnerPb.SubmissionStatusUpdate_ACCEPTED,
		TestsCompleted: int32(len(request.GetTestCases())),
		TotalTests:     int32(len(request.GetTestCases())),
		MaxTimeSpentMs: 100,
	})
	if err != nil {
		logger.Error("could not send update in stream", "error", err)
		return status.Error(codes.Internal, "could not send last message in stream")
	}

	return nil
}
