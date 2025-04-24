package runner

import (
	"os"
	"os/exec"
	"testing"

	runnerPb "github.com/computer-technology-team/go-judge/api/gen/runner"
)

func TestMain(m *testing.M) {
	// Build the Docker image
	cmd := exec.Command("docker", "build", "-t", "code-executer", "-f", "./test.Dockerfile", ".")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		panic("Failed to build Docker image: " + err.Error())
	}

	// Run the tests
	os.Exit(m.Run())
}

const helloWorld = `package main
import "fmt"
func main() {
fmt.Println("Hello, World!")
}`

func Test_Executer_Simple(t *testing.T) {
	testInput := ""
	testOutput := "Hello, World!\n"
	timeLimitMs := 20_000
	memoryLimitKb := 10_000

	exitCode, err := NewExecuter().ExecuteTestCase(helloWorld, testInput, testOutput, timeLimitMs, memoryLimitKb)
	if err != nil {
		t.Fatalf("RunGoCodeInContainer returned error: %v", err)
	}
	status, ok := exitCodeToStatus[int32(exitCode)]
	if !ok {
		t.Errorf("Couldn't find the status of this exit code %d.", exitCode)
	}
	if status != runnerPb.SubmissionStatusUpdate_ACCEPTED {
		t.Errorf("Expected exit code 0, got %d", exitCode)
	}
}

const memoryHello = `package main
import (
	"fmt"
	"time"
)
func main() {
	fmt.Println("Starting memory hog...")
	const mb = 1024 * 1024 // 1MB blocks
	const mbPerBlock = 10
	var data [][]byte
	for {
		block := make([]byte, mb*mbPerBlock)
		// Fill the block with data to ensure memory is actually used
		for i := 0; i < len(block); i += 1024 {
			block[i] = 1
		}
		data = append(data, block)
		fmt.Printf("Allocated %d MB, total blocks: %d\n", mbPerBlock, len(data))
		time.Sleep(50 * time.Millisecond)
	}
}`

func Test_Executer_Memory_Limit(t *testing.T) {
	testInput := ""
	testOutput := ""
	timeLimitMs := 3000
	memoryLimitKb := 10_000

	exitCode, err := NewExecuter().ExecuteTestCase(memoryHello, testInput, testOutput, timeLimitMs, memoryLimitKb)
	if err != nil {
		t.Fatalf("RunGoCodeInContainer returned error: %v", err)
	}
	status, ok := exitCodeToStatus[int32(exitCode)]
	if !ok {
		t.Errorf("Couldn't find the status of this exit code %d.", exitCode)
	}
	if status != runnerPb.SubmissionStatusUpdate_MEMORY_LIMIT_EXCEEDED {
		t.Errorf("Expected exit code 0, got %d", exitCode)
	}
}

const billionHellos = `package main
import "fmt"
func main() {
	for i := 0; i < 1000000000; i++ {
		fmt.Printf("Hello %d \n", i)
	}
}`

func Test_Executer_Time_Limit(t *testing.T) {
	testInput := ""
	testOutput := ""
	timeLimitMs := 3000
	memoryLimitKb := 100_000

	exitCode, err := NewExecuter().ExecuteTestCase(billionHellos, testInput, testOutput, timeLimitMs, memoryLimitKb)
	if err != nil {
		t.Fatalf("RunGoCodeInContainer returned error: %v", err)
	}
	status, ok := exitCodeToStatus[int32(exitCode)]
	if !ok {
		t.Errorf("Couldn't find the status of this exit code %d.", exitCode)
	}
	if status != runnerPb.SubmissionStatusUpdate_TIME_LIMIT_EXCEEDED {
		t.Errorf("Expected exit code 0, got %d", exitCode)
	}
}

const sumInts = `package main
import (
	"fmt"
)
func main() {
	var a, b int
	_, err := fmt.Scan(&a, &b)
	if err != nil {
		fmt.Println("Invalid input.")
		return
	}
	fmt.Printf("%d", a+b)
}`

func Test_Executers_Input_Output(t *testing.T) {
	testInput := "2 3"
	testOutput := "5"
	timeLimitMs := 20_000
	memoryLimitKb := 100_000

	exitCode, err := NewExecuter().ExecuteTestCase(sumInts, testInput, testOutput, timeLimitMs, memoryLimitKb)
	if err != nil {
		t.Fatalf("RunGoCodeInContainer returned error: %v", err)
	}
	status, ok := exitCodeToStatus[int32(exitCode)]
	if !ok {
		t.Errorf("Couldn't find the status of this exit code %d.", exitCode)
	}
	if status != runnerPb.SubmissionStatusUpdate_ACCEPTED {
		t.Errorf("Expected exit code 0, got %d", exitCode)
	}
}
