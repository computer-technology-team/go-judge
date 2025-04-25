package runner

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"

	runnerPb "github.com/computer-technology-team/go-judge/api/gen/runner"
	"github.com/computer-technology-team/go-judge/config"
	"github.com/computer-technology-team/go-judge/internal/runner"
)

func StartServer(ctx context.Context, cfg config.Config) error {
	grpcServer := grpc.NewServer()

	evaluator, err := runner.NewCodeEvaluator(ctx)
	if err != nil {
		return fmt.Errorf("could not create code evaluator: %w", err)
	}

	runnerCnt, err := getRunnerCount(cfg.RunnerClient.Address)
	if err != nil {
		return fmt.Errorf("could not get runnner count: %w", err)
	}

	runnerServer, err := runner.NewRunnerServer(ctx, runnerCnt, evaluator)
	if err != nil {
		return fmt.Errorf("could not create runner server: %w", err)
	}

	runnerPb.RegisterRunnerServer(grpcServer, runnerServer)

	healthServer := health.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)

	healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus("gojudge.Runner", grpc_health_v1.HealthCheckResponse_SERVING)

	reflection.Register(grpcServer)

	addr := fmt.Sprintf("%s:%d", cfg.RunnerServer.Host, cfg.RunnerServer.Port)
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}

	go func() {
		slog.Info("gRPC server starting", "address", addr)
		if err := grpcServer.Serve(lis); err != nil {
			slog.Error("gRPC server failed to start", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	slog.Info("gRPC server is shutting down")

	stopped := make(chan struct{})
	go func() {
		grpcServer.GracefulStop()
		close(stopped)
	}()

	timeout := time.After(30 * time.Second)
	select {
	case <-stopped:
		slog.Info("gRPC server exited properly")
	case <-timeout:
		slog.Warn("gRPC server shutdown timed out, forcing stop")
		grpcServer.Stop()
	}

	return nil
}

func getRunnerCount(runnnerEndpoint string) (int, error) {
	parsedUrl, err := url.Parse("tcp://" + runnnerEndpoint)
	if err != nil {
		return 0, fmt.Errorf("could not parse endpoint: %w", err)
	}

	host := parsedUrl.Hostname()
	ips, err := net.LookupIP(host)
	return len(ips), err
}
