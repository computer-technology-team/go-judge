#!/bin/sh

# Check if binary exists
if [ ! -f "/app/${BINARY_NAME}" ]; then
    echo "Error: Binary /app/${BINARY_NAME} not found"
    exit 1
fi

# Set input and output files
INPUT_FILE="/app/test_input"
EXPECTED_OUTPUT="/app/test_output"
USER_OUTPUT="/app/user_output"


timeout ${TIME_LIMIT}s /app/${BINARY_NAME} < "$INPUT_FILE" > "$USER_OUTPUT"
EXIT_STATUS=$?

# Check the exit status
if [ $EXIT_STATUS -eq 124 ]; then
    echo "Error: Process timed out after ${TIME_LIMIT} seconds"
    exit 1
elif [ $EXIT_STATUS -eq 137 ]; then
    echo "Error: Process terminated due to memory limit violation"
    exit 1
elif [ $EXIT_STATUS -ne 0 ]; then
    echo "Process failed with exit code $EXIT_STATUS"
    exit $EXIT_STATUS
fi

# Always compare outputs
if diff -q "$USER_OUTPUT" "$EXPECTED_OUTPUT" > /dev/null; then
    echo "Output matches expected output."
    exit 0
else
    echo "Output does NOT match expected output."
    echo "--- User Output ---"
    cat "$USER_OUTPUT"
    echo "--- Expected Output ---"
    cat "$EXPECTED_OUTPUT"
    exit 2
fi