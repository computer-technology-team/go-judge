#!/bin/bash

# Usage check
if [ -z "$1" ]; then
  echo "Usage: $0 <output-binary-name>"
  exit 1
fi

# Set the output binary name from the first argument
OUTPUT_BINARY="$1"

# Define directories
BUILD_DIR="/home/matin/code/go-judge/runner/builds"
SRC_DIR="/home/matin/code/go-judge/runner/sample-scripts/${OUTPUT_BINARY}"

# Create the builds directory if needed
mkdir -p "${BUILD_DIR}"

# Build the Go program
go build -o "${BUILD_DIR}/${OUTPUT_BINARY}" "${SRC_DIR}/${OUTPUT_BINARY}.go"

# Check if the build was successful
if [ $? -eq 0 ]; then
  echo "Build successful. Binary generated: ${BUILD_DIR}/${OUTPUT_BINARY}"
else
  echo "Build failed."
  exit 1
fi
