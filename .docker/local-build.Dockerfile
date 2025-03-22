FROM golang:1.23-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git gcc musl-dev

# Set working directory
WORKDIR /app

# Copy go.mod and go.sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o go-judge .

# Create a minimal runtime image
FROM alpine:3.20

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

# Set working directory
WORKDIR /app

# Copy the binary from the builder stage
COPY --from=builder /app/go-judge .

# Copy entrypoint script
COPY --from=builder /app/scripts/docker-entrypoint.sh .
RUN chmod +x docker-entrypoint.sh

COPY ./configs/config.yaml /app/config.yaml

# Create a non-root user to run the application
RUN adduser -D -g '' appuser && chown -R appuser:appuser /app
USER appuser

# Expose the application port
EXPOSE 8080


# Command to run migrations and then start the application
ENTRYPOINT ["./docker-entrypoint.sh"]

