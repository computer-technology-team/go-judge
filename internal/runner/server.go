package runner

import (
	"log/slog"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	runnerPb "github.com/computer-technology-team/go-judge/api/gen/runner"
)

type runnerServer struct {
	runnerPb.UnimplementedRunnerServer
	executer *Executer
}

func NewRunnerServer() runnerPb.RunnerServer {
	return &runnerServer{executer: NewExecuter()}
}

func (rs *runnerServer) ExecuteSubmission(
	request *runnerPb.SubmissionRequest,
	stream grpc.ServerStreamingServer[runnerPb.SubmissionStatusUpdate],
) error {
	err := stream.Send(&runnerPb.SubmissionStatusUpdate{
		SubmissionId:   request.GetSubmissionId(),
		Status:         runnerPb.SubmissionStatusUpdate_RUNNING,
		TestsCompleted: 0,
		TotalTests:     int32(len(request.GetTestCases())),
		MaxTimeSpentMs: 0,
	})
	if err != nil {
		slog.Error("could not send update in stream", "error", err)
		return status.Error(codes.Internal, "could not send first message in stream")
	}

	for i, tc := range request.GetTestCases() {
		err := stream.Send(&runnerPb.SubmissionStatusUpdate{
			SubmissionId:   request.GetSubmissionId(),
			Status:         runnerPb.SubmissionStatusUpdate_RUNNING,
			TestsCompleted: int32(i + 1),
			TotalTests:     int32(len(request.GetTestCases())),
			MaxTimeSpentMs: 100,
		})
		if err != nil {
			slog.Error("could not send update in stream", "error", err)
			return status.Error(codes.Internal, "could not send subsequent messages in stream")
		}

		input := tc.Input
		output := tc.Output
		code := request.Code
		timeLimit := request.TimeLimitMs
		memoryLimit := request.MemoryLimitKb

		exitCode, err := rs.executer.ExecuteTestCase(code, input, output, int(timeLimit), int(memoryLimit))
		if err != nil {
			slog.Error("could not send update in stream", "error", err)
			return status.Error(codes.Internal, "could not send subsequent messages in stream")
		}

		if exitCode == 0 {
			// The testcase is correct, continue to next
			continue
		}

		stat, ok := exitCodeToStatus[int32(exitCode)]
		if !ok {
			stat = runnerPb.SubmissionStatusUpdate_INTERNAL_ERROR
		}

		err = stream.Send(&runnerPb.SubmissionStatusUpdate{
			SubmissionId:   request.GetSubmissionId(),
			Status:         stat,
			TestsCompleted: int32(i + 1),
			TotalTests:     int32(len(request.GetTestCases())),
			MaxTimeSpentMs: 100,
		})
		if err != nil {
			slog.Error("could not send update in stream", "error", err)
			return status.Error(codes.Internal, "could not send subsequent messages in stream")
		}
		return nil
	}

	err = stream.Send(&runnerPb.SubmissionStatusUpdate{
		SubmissionId:   request.GetSubmissionId(),
		Status:         runnerPb.SubmissionStatusUpdate_ACCEPTED,
		TestsCompleted: int32(len(request.GetTestCases())),
		TotalTests:     int32(len(request.GetTestCases())),
		MaxTimeSpentMs: 100,
	})
	if err != nil {
		slog.Error("could not send update in stream", "error", err)
		return status.Error(codes.Internal, "could not send last message in stream")
	}

	return nil
}
