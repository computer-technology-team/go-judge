package runner

import (
	"log/slog"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	runnerPb "github.com/computer-technology-team/go-judge/api/gen/runner"
)

type runnerServer struct {
	runnerPb.UnimplementedRunnerServer
}

func NewRunnerServer() runnerPb.RunnerServer {
	return &runnerServer{}
}

func (rs *runnerServer) ExecuteSubmission(
	request *runnerPb.SubmissionRequest,
	stream grpc.ServerStreamingServer[runnerPb.SubmissionStatusUpdate],
) error {
	// Some random code to simulate executaion
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

	for i := range request.GetTestCases() {
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
		time.Sleep(time.Millisecond * 100)
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
