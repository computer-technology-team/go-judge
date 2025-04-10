package runner

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/backoff"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/keepalive"

	runnerPb "github.com/computer-technology-team/go-judge/api/gen/runner"
	"github.com/computer-technology-team/go-judge/config"
)

type RunnerClient struct {
	runnerPb.RunnerClient
	conn *grpc.ClientConn
}

func (c *RunnerClient) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

func NewClient(ctx context.Context, cfg config.ClientConfig) (*RunnerClient, error) {
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultServiceConfig(`{"loadBalancingPolicy":"round_robin"}`),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:    10 * time.Second,
			Timeout: 3 * time.Second,
		}),
		grpc.WithConnectParams(grpc.ConnectParams{
			Backoff: backoff.Config{
				BaseDelay:  100 * time.Millisecond,
				Multiplier: 1.6,
				Jitter:     0.2,
				MaxDelay:   10 * time.Second,
			},
			MinConnectTimeout: time.Second * 5,
		}),
	}

	conn, err := grpc.NewClient(
		cfg.Address,
		opts...,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to runner service: %w", err)
	}

	client := runnerPb.NewRunnerClient(conn)

	healthClient := grpc_health_v1.NewHealthClient(conn)
	resp, err := healthClient.Check(ctx, &grpc_health_v1.HealthCheckRequest{
		Service: "gojudge.Runner",
	})
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("health check failed: %w", err)
	}
	if resp.Status != grpc_health_v1.HealthCheckResponse_SERVING {
		conn.Close()
		return nil, fmt.Errorf("service is not serving: %v", resp.Status)
	}

	return &RunnerClient{
		RunnerClient: client,
		conn:         conn,
	}, nil
}
