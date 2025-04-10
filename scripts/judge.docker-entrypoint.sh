#!/bin/sh
set -e

echo "Running database migrations..."
./go-judge migrate

echo "Starting the application..."
exec ./go-judge serve
