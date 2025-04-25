package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func main() {
	// Define command-line flags
	var (
		binaryName   = flag.String("binary", "submission", "Name of the binary to execute")
		binaryFolder = flag.String("binary-dir", "/build", "Name of the dir that contains binary")
		timeLimit    = flag.Int("timeout", 10_000, "Timeout in milliseconds")
		appDir       = flag.String("dir", "/app", "Directory containing the test files")
		inputFile    = flag.String("input", "test_input", "Name of the input file")
		outputFile   = flag.String("output", "test_output", "Name of the expected output file")
		userOutFile  = flag.String("user-output", "user_output", "Name of the file to write user output to")
	)

	flag.Parse()

	binaryPath := filepath.Join(*binaryFolder, *binaryName)
	inputPath := filepath.Join(*appDir, *inputFile)
	expectedPath := filepath.Join(*appDir, *outputFile)
	userOutputPath := filepath.Join(*appDir, *userOutFile)

	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		fmt.Printf("Error: Binary not found at %s\n", binaryPath)
		os.Exit(127) // Standard exit code for "command not found"
	}

	// Read input file
	input, err := os.ReadFile(inputPath)
	if err != nil {
		fmt.Printf("Error reading input file %s: %v\n", inputPath, err)
		os.Exit(3) // Exit code 3 for internal errors (file system issues)
	}

	userOutputFile, err := os.Create(userOutputPath)
	if err != nil {
		fmt.Printf("Error creating user output file %s: %v\n", *userOutFile, err)
		os.Exit(3)
	}
	defer userOutputFile.Close()

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(*timeLimit)*time.Millisecond)
	defer cancel()

	// Run the binary with input redirection using pipes to ensure proper EOF handling
	cmd := exec.CommandContext(ctx, binaryPath)

	// Set up pipes for stdin and combined output
	stdin, err := cmd.StdinPipe()
	if err != nil {
		fmt.Printf("Error creating stdin pipe: %v\n", err)
		os.Exit(3) // Exit code 3 for internal errors
	}

	// Capture combined output
	var errorBuffer strings.Builder
	cmd.Stdout = userOutputFile
	cmd.Stderr = &errorBuffer

	// Start the command
	if err := cmd.Start(); err != nil {
		fmt.Printf("Error starting command: %v\n", err)
		os.Exit(3) // Exit code 3 for internal errors
	}

	// Write input to stdin
	_, err = stdin.Write(input)
	if err != nil {
		fmt.Printf("Error writing to stdin: %v\n", err)
		os.Exit(3) // Exit code 3 for internal errors
	}

	// Close stdin to signal EOF
	stdin.Close()

	// Wait for command to complete
	err = cmd.Wait()
	if ctx.Err() == context.DeadlineExceeded {
		fmt.Printf("Error: Process timed out after %d seconds\n", *timeLimit)
		os.Exit(124) // Standard exit code for "timed out"
	}

	// Check for other errors
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode := exitErr.ExitCode()
			// Handle OOM kill (137) and other memory-related errors
			if exitCode == 137 || exitCode == -1 {
				fmt.Println("Error: Process terminated due to memory limit violation")
				os.Exit(137) // Standardize on 137 for OOM kill
			}
			fmt.Printf("RUNTIME ERROR\n exit code %d:\n%s", exitCode, errorBuffer.String())
			os.Exit(exitCode)
		} else {
			fmt.Printf("Error executing binary: %v\n", err)
			os.Exit(3) // Exit code 3 for internal errors (execution issues)
		}
	}

	// Compare outputs
	expected, err := os.ReadFile(expectedPath)
	if err != nil {
		fmt.Printf("Error reading expected output file %s: %v\n", expectedPath, err)
		os.Exit(3) // Exit code 3 for internal errors (file system issues)
	}

	output, err := io.ReadAll(userOutputFile)
	if err != nil {
		fmt.Printf("Error reading user output file :%v\n", err)
		os.Exit(3) // Exit code 3 for internal errors (file system issues)
	}

	// Normalize line endings and trim whitespace for comparison
	normalizedOutput := strings.TrimSpace(string(output))
	normalizedExpected := strings.TrimSpace(string(expected))

	if normalizedOutput == normalizedExpected {
		fmt.Println("CORRECT")
		os.Exit(0) // Success
	} else {
		fmt.Println("INCORRECT")
		fmt.Println("--- User Output ---")
		fmt.Println(normalizedOutput)
		fmt.Println("--- Expected Output ---")
		fmt.Println(normalizedExpected)
		os.Exit(0) // Exit code 2 for wrong answer (output mismatch)
	}
}
