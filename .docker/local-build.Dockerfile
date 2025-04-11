FROM golang:1.23 AS builder

# Update apt keys and install build dependencies
RUN apt-get update && apt-get install -y \
    gnupg \
    ca-certificates \
    && apt-key update \
    && apt-get update && apt-get install -y \
    git \
    build-essential \
    && rm -rf /var/lib/apt/lists/*

# Set working directory
WORKDIR /app

# Copy go.mod and go.sum files
COPY go.mod go.sum ./

# Download dependencies
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go mod download

# Copy the source code
COPY . .

# Build the application
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o go-judge .

# Create a runtime image
FROM ubuntu:22.04

# Install runtime dependencies
RUN apt-get update && apt-get install -y \
    gnupg \
    ca-certificates \
    && apt-key update \
    && apt-get update && apt-get install -y \
    tzdata \
    && rm -rf /var/lib/apt/lists/*

# Set working directory
WORKDIR /app

# Copy the binary from the builder stage
COPY --from=builder /app/go-judge .

# Copy entrypoint script
COPY --from=builder /app/scripts/docker-entrypoint.sh .
RUN chmod +x docker-entrypoint.sh

COPY ./configs/config.yaml /app/config.yaml

# Create a non-root user to run the application
RUN useradd -m appuser && chown -R appuser:appuser /app
USER appuser

# Expose the application port
EXPOSE 8080


# Command to run migrations and then start the application
ENTRYPOINT ["./docker-entrypoint.sh"]
