package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func main() {
	// Define command-line flags
	var (
		binaryName  = flag.String("binary", "app", "Name of the binary to execute")
		timeLimit   = flag.Int("timeout", 10, "Timeout in seconds")
		appDir      = flag.String("dir", "/app", "Directory containing the binary and test files")
		inputFile   = flag.String("input", "test_input", "Name of the input file")
		outputFile  = flag.String("output", "test_output", "Name of the expected output file")
		userOutFile = flag.String("user-output", "user_output", "Name of the file to write user output to")
	)

	// Parse command-line flags
	flag.Parse()

	// Construct full paths
	binaryPath := filepath.Join(*appDir, *binaryName)
	inputPath := filepath.Join(*appDir, *inputFile)
	expectedPath := filepath.Join(*appDir, *outputFile)
	userOutputPath := filepath.Join(*appDir, *userOutFile)

	// Check if binary exists
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		fmt.Printf("Error: Binary not found at %s\n", binaryPath)
		os.Exit(11)
	}

	// Read input file
	input, err := os.ReadFile(inputPath)
	if err != nil {
		fmt.Printf("Error reading input file %s: %v\n", inputPath, err)
		os.Exit(1)
	}

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(*timeLimit)*time.Millisecond)
	defer cancel()

	// Run the binary with input redirection
	cmd := exec.CommandContext(ctx, binaryPath)
	cmd.Stdin = strings.NewReader(string(input))

	// Capture both stdout and stderr
	output, err := cmd.CombinedOutput()

	// Check for timeout
	if ctx.Err() == context.DeadlineExceeded {
		fmt.Printf("Error: Process timed out after %d seconds\n", *timeLimit)
		os.Exit(12)
	}

	// Check for other errors
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode := exitErr.ExitCode()
			if exitCode == 137 {
				fmt.Println("Error: Process terminated due to memory limit violation")
				os.Exit(10)
			}
			fmt.Printf("Process failed with exit code %d\n", exitCode)
			os.Exit(exitCode)
		} else {
			fmt.Printf("Error executing binary: %v\n", err)
			os.Exit(1)
		}
	}

	// Write output to user output file
	if err := os.WriteFile(userOutputPath, output, 0644); err != nil {
		fmt.Printf("Error writing user output to %s: %v\n", userOutputPath, err)
		os.Exit(1)
	}

	// Compare outputs
	expected, err := os.ReadFile(expectedPath)
	if err != nil {
		fmt.Printf("Error reading expected output file %s: %v\n", expectedPath, err)
		os.Exit(1)
	}

	// Normalize line endings and trim whitespace for comparison
	normalizedOutput := strings.TrimSpace(string(output))
	normalizedExpected := strings.TrimSpace(string(expected))

	if normalizedOutput == normalizedExpected {
		fmt.Println("Output matches expected output.")
		os.Exit(0)
	} else {
		fmt.Println("Output does NOT match expected output.")
		fmt.Println("--- User Output ---")
		fmt.Println(normalizedOutput)
		fmt.Println("--- Expected Output ---")
		fmt.Println(normalizedExpected)
		os.Exit(2)
	}
}
