# Always mount input and output files, no checks needed
INPUT_FILE="/home/matin/code/go-judge/runner/sample-scripts/$1/test_input"
OUTPUT_FILE="/home/matin/code/go-judge/runner/sample-scripts/$1/test_output"

docker run --rm \
    --memory=$2m \
    --memory-swap=$2m \
    -e TIME_LIMIT=10 \
    -e BINARY_NAME=$1 \
    -v $INPUT_FILE:/app/test_input \
    -v $OUTPUT_FILE:/app/test_output \
    -v /home/matin/code/go-judge/runner/builds/$1:/app/$1 \
    runner-container:latest