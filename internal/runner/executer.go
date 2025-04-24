package runner

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
)

type Executer struct {
}

func NewExecuter() *Executer {
	return &Executer{}
}

func (*Executer) ExecuteTestCase(code, testInput, testOutput string, timelimitMs, memorylimitKb int) (int, error) {
	dir, err := os.MkdirTemp("", "go-judge-runner-*")
	if err != nil {
		return -1, err
	}
	defer os.RemoveAll(dir)

	mainPath := filepath.Join(dir, "main.go")
	inputPath := filepath.Join(dir, "test_input")
	outputPath := filepath.Join(dir, "test_output")
	binPath := filepath.Join(dir, "main-bin")

	if err := os.WriteFile(mainPath, []byte(code), 0644); err != nil {
		return -1, err
	}
	if err := os.WriteFile(inputPath, []byte(testInput), 0644); err != nil {
		return -1, err
	}
	if err := os.WriteFile(outputPath, []byte(testOutput), 0644); err != nil {
		return -1, err
	}

	buildCmd := exec.Command("go", "build", "-o", binPath, mainPath)
	buildCmd.Dir = dir
	if err := buildCmd.Run(); err != nil {
		return -1, err
	}

	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return -1, err
	}
	defer cli.Close()

	memLimit := int64(memorylimitKb) * 1024 // bytes
	env := []string{
		"TIME_LIMIT=" + strconv.Itoa(timelimitMs/1000),
		"BINARY_NAME=main-bin",
	}
	mounts := []mount.Mount{
		{
			Type:   mount.TypeBind,
			Source: inputPath,
			Target: "/app/test_input",
		},
		{
			Type:   mount.TypeBind,
			Source: outputPath,
			Target: "/app/test_output",
		},
		{
			Type:   mount.TypeBind,
			Source: binPath,
			Target: "/app/main-bin",
		},
	}

	resp, err := cli.ContainerCreate(ctx, &container.Config{
		Image: imageName,
		Env:   env,
	}, &container.HostConfig{
		Mounts:     mounts,
		Resources:  container.Resources{Memory: memLimit, MemorySwap: memLimit},
		AutoRemove: true,
	}, nil, nil, "")
	if err != nil {
		return -1, err
	}

	if err := cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		return -1, err
	}

	statusCh, errCh := cli.ContainerWait(ctx, resp.ID, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil {
			return -1, err
		}
	case status := <-statusCh:
		return int(status.StatusCode), nil
	}
	return -1, nil
}
