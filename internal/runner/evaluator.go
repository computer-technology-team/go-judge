package runner

import (
	"archive/tar"
	"bytes"
	"context"
	_ "embed"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/computer-technology-team/go-judge/api/gen/runner"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
)

const utilVolumeName = "go-judge_go-runner-utils"

var ErrCompilationFailed = errors.New("could not compile program")
var ErrExecutionFailed = errors.New("could not execute program")

type CodeEvaluator struct {
	dockerClient *client.Client
}

type BuildError struct {
	ExitCode int
	Logs     string
}

func (e *BuildError) Error() string {
	return fmt.Sprintf("build failed with exit code: %d", e.ExitCode)
}

type RunStatus struct {
	Stdout        string
	Stderr        string
	Status        runner.SubmissionStatusUpdate_Status
	ExecutionTime time.Duration
}

func NewCodeEvaluator(ctx context.Context) (*CodeEvaluator, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker client: %w", err)
	}
	err = pullImages(ctx, cli)
	if err != nil {
		return nil, fmt.Errorf("failed to pull images: %w", err)
	}

	return &CodeEvaluator{
		dockerClient: cli,
	}, nil

}

func pullImages(ctx context.Context, cli *client.Client) error {
	slog.Info("pulling images")
	pullResp, err := cli.ImagePull(ctx, "golang:1.23", image.PullOptions{})
	if pullResp != nil {
		pullResp.Close()
	}
	if err != nil {
		return fmt.Errorf("could not pull golang:1.23: %w", err)
	}

	pullResp, err = cli.ImagePull(ctx, "ubuntu:22.04", image.PullOptions{})
	if pullResp != nil {
		pullResp.Close()
	}
	if err != nil {
		return fmt.Errorf("could not pull golang:1.23: %w", err)
	}

	return nil
}

func (c *CodeEvaluator) GetCpuCount(ctx context.Context) (int, error) {
	sysInfo, err := c.dockerClient.Info(ctx)
	if err != nil {
		return 0, fmt.Errorf("could not get system info from docker client: %w", err)
	}

	return sysInfo.NCPU, nil
}

func (c *CodeEvaluator) BuildCodeBinary(ctx context.Context, submissionID, code string) error {
	volumeName := fmt.Sprintf("go-judge-volume-%s", submissionID)

	_, err := c.dockerClient.VolumeCreate(ctx, volume.CreateOptions{
		Name: volumeName,
	})
	if err != nil {
		return fmt.Errorf("could not create volume: %w", err)
	}

	codeBuf := byteFileToTar([]byte(code), "main.go")

	mounts := []mount.Mount{
		{
			Type:   mount.TypeVolume,
			Source: volumeName,
			Target: "/build",
		},
	}

	resp, err := c.dockerClient.ContainerCreate(ctx, &container.Config{
		Image:      "golang:1.23",
		Cmd:        []string{"go", "build", "-o", "/build/submission", "/app/main.go"},
		WorkingDir: "/app",
	}, &container.HostConfig{
		Mounts: mounts,
		Resources: container.Resources{
			Memory:     2_000_000_000,
			MemorySwap: 4_000_000_000,
			CPUPeriod:  100000, // 100ms (in microseconds)
			CPUQuota:   100000, // 100% of one CPU core
			CPUCount:   1,
		},
		NetworkMode: "none",
		AutoRemove:  true,
	}, nil, nil, fmt.Sprintf("go-runner-build-%s", submissionID))
	if err != nil {
		return err
	}

	err = c.dockerClient.CopyToContainer(ctx, resp.ID, "/app", &codeBuf, container.CopyToContainerOptions{})
	if err != nil {
		return fmt.Errorf("could not copy main.go to container: %w", err)
	}

	return c.waitForBuildContainer(ctx, resp.ID)
}

func (c *CodeEvaluator) waitForBuildContainer(ctx context.Context, containerID string) error {
	if err := c.dockerClient.ContainerStart(ctx, containerID, container.StartOptions{}); err != nil {
		return err
	}

	statusCh, errCh := c.dockerClient.ContainerWait(ctx, containerID, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil {
			slog.Error("building code failed", "error", err)
			return fmt.Errorf("build container error: %w", err)
		}
	case status := <-statusCh:
		if status.StatusCode != 0 {
			out, err := c.dockerClient.ContainerLogs(ctx, containerID, container.LogsOptions{
				ShowStdout: true,
				ShowStderr: true,
			})
			if err != nil {
				return &BuildError{ExitCode: int(status.StatusCode)}
			}
			defer out.Close()

			logs, _ := io.ReadAll(out)
			return &BuildError{ExitCode: int(status.StatusCode), Logs: sanitizeUTF8(logs)}
		}
		return nil
	}
	return nil
}

func (c *CodeEvaluator) RunTestCase(ctx context.Context, submissionID string, testInput, testOutput string, timelimitMs, memorylimitKb int64) (*RunStatus, error) {
	volumeName := fmt.Sprintf("go-judge-volume-%s", submissionID)

	inputBuf, outputBuf := byteFileToTar([]byte(testInput), "test_input"), byteFileToTar([]byte(testOutput), "test_output")

	memSize := memorylimitKb * 1024

	resp, err := c.dockerClient.ContainerCreate(ctx, &container.Config{
		Image:        "ubuntu:22.04",
		Cmd:          []string{"/utils/spy", "-timeout", strconv.Itoa(int(timelimitMs))},
		Tty:          false,
		AttachStdin:  true,
		AttachStdout: true,
		AttachStderr: true,
		OpenStdin:    true,
		StdinOnce:    true,
		WorkingDir:   "/app",
	}, &container.HostConfig{
		Mounts: []mount.Mount{
			{
				Type:     mount.TypeVolume,
				Target:   "/build",
				Source:   volumeName,
				ReadOnly: true,
			},
			{
				Type:     mount.TypeVolume,
				Target:   "/utils",
				Source:   utilVolumeName,
				ReadOnly: true,
			},
		},
		Resources: container.Resources{
			Memory:            memSize, // Convert KB to bytes
			MemoryReservation: memSize,
			MemorySwap:        memSize,
			CPUCount:          1,
			CPUPeriod:         100_000,
			CPUQuota:          100_000, // Equivalent to 1 core (100% of one CPU period)
			OomKillDisable:    &[]bool{false}[0],
		},
		NetworkMode: "none",
	}, nil, nil, fmt.Sprintf("go-runner-execution-%s", submissionID))
	if err != nil {
		return nil, fmt.Errorf("failed to create runner container: %w", err)
	}

	runnerContainerID := resp.ID

	err = c.dockerClient.CopyToContainer(ctx, runnerContainerID, "/app", &inputBuf, container.CopyToContainerOptions{})
	if err != nil {
		return nil, fmt.Errorf("could not copy input test file into container: %w", err)
	}

	err = c.dockerClient.CopyToContainer(ctx, runnerContainerID, "/app", &outputBuf, container.CopyToContainerOptions{})
	if err != nil {
		return nil, fmt.Errorf("could not copy output test file into container: %w", err)
	}

	defer c.dockerClient.ContainerRemove(ctx, runnerContainerID, container.RemoveOptions{Force: true})

	if err := c.dockerClient.ContainerStart(ctx, runnerContainerID, container.StartOptions{}); err != nil {
		return nil, fmt.Errorf("failed to start runner container: %w", err)
	}

	var statusCh <-chan container.WaitResponse
	var errCh <-chan error

	startTime := time.Now()

	statusCh, errCh = c.dockerClient.ContainerWait(ctx, runnerContainerID, container.WaitConditionNotRunning)

	// Wait for execution to complete
	var executionError error
	var exitCode int

	select {
	case err := <-errCh:
		if err != nil {
			executionError = err
		}
	case status := <-statusCh:
		exitCode = int(status.StatusCode)
		if status.StatusCode != 0 {
			executionError = ErrExecutionFailed
		}
	case <-ctx.Done():
		executionError = errors.New("execution timed out")
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
		Status:        c.getStatusCode(stdout.String(), exitCode),
		ExecutionTime: executionTime,
	}

	if executionError != nil {
		return status, executionError
	}

	return status, nil
}

// sanitizeUTF8 removes null bytes and ensures the string is valid UTF-8
func sanitizeUTF8(input []byte) string {
	// Remove null bytes
	input = bytes.ReplaceAll(input, []byte{0}, []byte{})

	// Convert to string, replacing invalid UTF-8 sequences
	return string(bytes.ToValidUTF8(input, []byte{}))
}

func byteFileToTar(content []byte, name string) bytes.Buffer {
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)

	hdr := &tar.Header{
		Name: name,
		Mode: 0644,
		Size: int64(len(content)),
	}
	tw.WriteHeader(hdr)
	tw.Write([]byte(content))
	tw.Close()

	return buf
}

func (*CodeEvaluator) getStatusCode(stdout string, exitCode int) runner.SubmissionStatusUpdate_Status {
	if exitCode == 0 {
		if strings.HasPrefix(stdout, "CORRECT") {
			return runner.SubmissionStatusUpdate_RUNNING
		} else {
			return runner.SubmissionStatusUpdate_WRONG_ANSWER
		}
	}

	if strings.HasPrefix(stdout, "RUNTIME ERROR") {
		return runner.SubmissionStatusUpdate_RUNTIME_ERROR
	}

	st, ok := exitCodeToStatus[exitCode]
	if ok {
		return st
	}

	return runner.SubmissionStatusUpdate_INTERNAL_ERROR
}
