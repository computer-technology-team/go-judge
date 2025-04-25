package runner

import (
	"errors"
	"log/slog"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	runnerPb "github.com/computer-technology-team/go-judge/api/gen/runner"
)

type runnerServer struct {
	runnerPb.UnimplementedRunnerServer

	codeEvaluator *CodeEvaluator
}

func NewRunnerServer(evaluator *CodeEvaluator) (runnerPb.RunnerServer, error) {

	return &runnerServer{
		codeEvaluator: evaluator,
	}, nil
}

func (rs *runnerServer) ExecuteSubmission(
	request *runnerPb.SubmissionRequest,
	stream grpc.ServerStreamingServer[runnerPb.SubmissionStatusUpdate],
) error {
	logger := slog.With("memory_limit", request.GetMemoryLimitKb(), "timelimit", request.GetTimeLimitMs(),
		"code", request.GetCode(), "submission_id", request.GetSubmissionId())
	logger.Info("recieved request")
	err := stream.Send(&runnerPb.SubmissionStatusUpdate{
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

	_, err = rs.codeEvaluator.BuildCodeBinary(stream.Context(), request.GetSubmissionId(), request.GetCode())
	if err != nil {
		if errors.Is(err, ErrCompilationFailed) {
			logger.Warn("compilation failed", "error", err)
			stream.Send(&runnerPb.SubmissionStatusUpdate{
				SubmissionId:   request.SubmissionId,
				Status:         runnerPb.SubmissionStatusUpdate_COMPILATION_ERROR,
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
	for i, tc := range request.GetTestCases() {
		logger.Info("running test case", "i", i)
		err := stream.Send(&runnerPb.SubmissionStatusUpdate{
			SubmissionId:   request.GetSubmissionId(),
			Status:         runnerPb.SubmissionStatusUpdate_RUNNING,
			TestsCompleted: int32(i),
			TotalTests:     int32(len(request.GetTestCases())),
			MaxTimeSpentMs: 100,
		})
		if err != nil {
			logger.Error("could not send update in stream", "error", err)
			return status.Error(codes.Internal, "could not send subsequent messages in stream")
		}

		runStatus, err := rs.codeEvaluator.RunTestCase(stream.Context(), request.GetSubmissionId(), tc.GetInput(), tc.GetOutput(), request.GetTimeLimitMs(), request.GetMemoryLimitKb())
		if err != nil {
			if errors.Is(err, ErrExecutionFailed) {
				logger.Info("exection failed", "error", err, "exit_code", runStatus.ExitCode, "stdout", runStatus.Stdout, "stderr", runStatus.Stderr)
				st, ok := exitCodeToStatus[runStatus.ExitCode]
				if !ok {
					st = runnerPb.SubmissionStatusUpdate_INTERNAL_ERROR
				}

				stream.Send(&runnerPb.SubmissionStatusUpdate{
					SubmissionId:   request.GetSubmissionId(),
					Status:         st,
					TestsCompleted: int32(i),
					TotalTests:     int32(len(request.GetTestCases())),
					MaxTimeSpentMs: 100,
				})

			} else {
				logger.Error("run test case failed", "error", err)
				stream.Send(&runnerPb.SubmissionStatusUpdate{
					SubmissionId:   request.GetSubmissionId(),
					Status:         runnerPb.SubmissionStatusUpdate_INTERNAL_ERROR,
					TestsCompleted: int32(i),
					TotalTests:     int32(len(request.GetTestCases())),
					MaxTimeSpentMs: 100,
				})
			}
			return nil
		}
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
