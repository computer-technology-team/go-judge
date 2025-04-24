package runner

import (
	"bytes"
	"context"
	_ "embed"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
)

//go:embed utils/spy.go
var spyCode []byte

var ErrCompilationFailed = errors.New("could not compile program")

type CodeEvaluator struct {
	dockerClient   *client.Client
	utilVolumeName string
}

type BuildStatus struct {
	ExitCode int
	Message  string
}

type RunStatus struct {
	Stdout        string
	Stderr        string
	ExitCode      int
	ExecutionTime time.Duration
}

func NewCodeEvaluator(ctx context.Context, dir string) (*CodeEvaluator, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker client: %w", err)
	}

	utilVolumeName := fmt.Sprintf("spy-%d", time.Now().Unix())

	resp, err := createSpyContainer(ctx, dir, utilVolumeName, cli)
	if err != nil {
		return nil, err
	}

	evaluator := &CodeEvaluator{dockerClient: cli, utilVolumeName: utilVolumeName}

	_, err = evaluator.waitForBuildContainer(ctx, resp.ID)
	if err != nil {
		return nil, err
	}

	return evaluator, nil
}

func createSpyContainer(ctx context.Context, dir string, utilVolumeName string, cli *client.Client) (container.CreateResponse, error) {
	filePath := filepath.Join(dir, "spy.go")

	if err := os.WriteFile(filePath, []byte(spyCode), 0644); err != nil {
		return container.CreateResponse{}, err
	}

	_, err := cli.VolumeCreate(ctx, volume.CreateOptions{
		Name: utilVolumeName,
	})
	if err != nil {
		return container.CreateResponse{}, err
	}

	mounts := []mount.Mount{
		{
			Type:   mount.TypeBind,
			Source: dir,
			Target: "/source",
		},
		{
			Type:   mount.TypeVolume,
			Source: utilVolumeName,
			Target: "/out",
		},
	}

	resp, err := cli.ContainerCreate(ctx, &container.Config{
		Image:      "golang:1.23",
		Cmd:        []string{"go", "build", "-o", "/out/spy", "/source/spy.go"},
		WorkingDir: "app",
	}, &container.HostConfig{
		Mounts: mounts,
		Resources: container.Resources{
			Memory:     2_000_000,
			MemorySwap: 4_000_000,
			CPUPeriod:  100000, // 100ms (in microseconds)
			CPUQuota:   100000, // 100% of one CPU core
			CPUCount:   1,
		},
		NetworkMode: "none",
		AutoRemove:  true,
	}, nil, nil, "go-runner-build-util")
	if err != nil {
		return container.CreateResponse{}, err
	}
	return resp, nil
}

func (c *CodeEvaluator) BuildCodeBinary(ctx context.Context, submissionID, code string) (*BuildStatus, error) {
	dir, err := os.MkdirTemp("", "go-judge-runner-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)

	mainPath := filepath.Join(dir, "main.go")
	if err := os.WriteFile(mainPath, []byte(code), 0644); err != nil {
		return nil, err
	}

	volumeName := fmt.Sprintf("go-judge-volume-%s", submissionID)

	_, err = c.dockerClient.VolumeCreate(ctx, volume.CreateOptions{
		Name: volumeName,
	})
	if err != nil {
		return nil, fmt.Errorf("could not create volume: %w", err)
	}

	mounts := []mount.Mount{
		{
			Type:   mount.TypeBind,
			Source: dir,
			Target: "/judge",
		},
		{
			Type:   mount.TypeVolume,
			Source: volumeName,
			Target: "/build",
		},
	}

	resp, err := c.dockerClient.ContainerCreate(ctx, &container.Config{
		Image:      "golang:1.23",
		Cmd:        []string{"go", "build", "-o", "/build/submission", "/judge/main.go"},
		WorkingDir: "app",
	}, &container.HostConfig{
		Mounts: mounts,
		Resources: container.Resources{
			Memory:     2_000_000,
			MemorySwap: 4_000_000,
			CPUPeriod:  100000, // 100ms (in microseconds)
			CPUQuota:   100000, // 100% of one CPU core
			CPUCount:   1,
		},
		NetworkMode: "none",
		AutoRemove:  true,
	}, nil, nil, fmt.Sprintf("go-runner-build-%s", submissionID))
	if err != nil {
		return nil, err
	}

	return c.waitForBuildContainer(ctx, resp.ID)
}

func (c *CodeEvaluator) waitForBuildContainer(ctx context.Context, containerID string) (*BuildStatus, error) {
	if err := c.dockerClient.ContainerStart(ctx, containerID, container.StartOptions{}); err != nil {
		return nil, err
	}

	statusCh, errCh := c.dockerClient.ContainerWait(ctx, containerID, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil {
			slog.Error("building code failed", "error", err)
			return nil, fmt.Errorf("build error: %w", err)
		}
	case status := <-statusCh:
		if status.StatusCode != 0 {
			out, err := c.dockerClient.ContainerLogs(ctx, containerID, container.LogsOptions{
				ShowStdout: true,
				ShowStderr: true,
			})
			if err != nil {
				return &BuildStatus{ExitCode: int(status.StatusCode)}, ErrCompilationFailed
			}
			defer out.Close()

			logs, _ := io.ReadAll(out)
			return &BuildStatus{ExitCode: int(status.StatusCode), Message: string(logs)}, ErrCompilationFailed
		}
		return &BuildStatus{ExitCode: 0}, nil
	}
	return nil, nil
}

func (c *CodeEvaluator) RunTestCase(ctx context.Context, submissionID string, testInput, testOutput string, timelimitMs, memorylimitKb int64) (*RunStatus, error) {
	// Create a volume name for this submission
	volumeName := fmt.Sprintf("go-judge-volume-%s", submissionID)

	dir, err := os.MkdirTemp("", "go-judge-runner-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)

	input := filepath.Join(dir, "input")
	if err := os.WriteFile(input, []byte(testInput), 0644); err != nil {
		return nil, err
	}

	output := filepath.Join(dir, "output")
	if err := os.WriteFile(output, []byte(testOutput), 0644); err != nil {
		return nil, err
	}

	resp, err := c.dockerClient.ContainerCreate(ctx, &container.Config{
		Image:        "ubuntu:22.04",
		Cmd:          []string{"/utils/spy"},
		Tty:          false,
		AttachStdin:  true,
		AttachStdout: true,
		AttachStderr: true,
		OpenStdin:    true,
		StdinOnce:    true,
	}, &container.HostConfig{
		Mounts: []mount.Mount{
			{
				Type:   mount.TypeVolume,
				Target: "/build",
				Source: volumeName,
			},
			{
				Type:   mount.TypeBind,
				Target: "/judge",
				Source: dir,
			},
			{
				Type:   mount.TypeVolume,
				Target: "/utils",
				Source: c.utilVolumeName,
			},
		},
		Resources: container.Resources{
			Memory:    memorylimitKb * 1024, // Convert KB to bytes
			CPUCount:  1,
			CPUPeriod: 100_000,
			CPUQuota:  100_000, // Equivalent to 1 core (100% of one CPU period)
		},
		NetworkMode: "none",
		AutoRemove:  true,
	}, nil, nil, fmt.Sprintf("go-runner-execution-%s", submissionID))

	if err != nil {
		return nil, fmt.Errorf("failed to create runner container: %w", err)
	}

	runnerContainerID := resp.ID
	defer c.dockerClient.ContainerRemove(ctx, runnerContainerID, container.RemoveOptions{Force: true})

	if err := c.dockerClient.ContainerStart(ctx, runnerContainerID, container.StartOptions{}); err != nil {
		return nil, fmt.Errorf("failed to start runner container: %w", err)
	}

	var statusCh <-chan container.WaitResponse
	var errCh <-chan error

	startTime := time.Now()

	timeoutDuration := time.Duration(timelimitMs) * time.Millisecond
	if timelimitMs > 0 {
		timeoutCtx, cancel := context.WithTimeout(ctx, timeoutDuration)
		defer cancel()
		statusCh, errCh = c.dockerClient.ContainerWait(timeoutCtx, runnerContainerID, container.WaitConditionNotRunning)
	} else {
		statusCh, errCh = c.dockerClient.ContainerWait(ctx, runnerContainerID, container.WaitConditionNotRunning)
	}

	// Wait for execution to complete
	var executionError error
	var exitCode int

	select {
	case err := <-errCh:
		if err != nil {
			executionError = fmt.Errorf("execution error: %w", err)
		}
	case status := <-statusCh:
		exitCode = int(status.StatusCode)
		if status.StatusCode != 0 {
			executionError = fmt.Errorf("execution failed with status %d", status.StatusCode)
		}
	case <-ctx.Done():
		executionError = fmt.Errorf("execution timed out")
		// Kill the container if it's still running
		c.dockerClient.ContainerKill(context.Background(), runnerContainerID, "SIGKILL")
	}

	executionTime := time.Since(startTime)

	out, err := c.dockerClient.ContainerLogs(ctx, runnerContainerID, container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get logs: %w", err)
	}
	defer out.Close()

	var stdout, stderr bytes.Buffer
	_, err = stdcopy.StdCopy(&stdout, &stderr, out)
	if err != nil {
		return nil, fmt.Errorf("failed to read logs: %w", err)
	}

	status := &RunStatus{
		Stdout:        stdout.String(),
		Stderr:        stderr.String(),
		ExitCode:      exitCode,
		ExecutionTime: executionTime,
	}

	if executionError != nil {
		return status, executionError
	}

	return status, nil
}
