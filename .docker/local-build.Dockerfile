FROM golang:1.23 AS builder

RUN apt-get update && apt-get install -y \
    gnupg \
    ca-certificates \
    && apt-key update \
    && apt-get update && apt-get install -y \
    git \
    build-essential \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY go.mod go.sum ./

# Download dependencies
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go mod download


COPY . .


# Build the application
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o go-judge .


FROM ubuntu:22.04

RUN apt-get update && apt-get install -y \
    gnupg \
    ca-certificates \
    && apt-key update \
    && apt-get update && apt-get install -y \
    tzdata wget \
    && rm -rf /var/lib/apt/lists/*

RUN GRPC_HEALTH_PROBE_VERSION=v0.4.19 && \
    wget -qO/bin/grpc_health_probe https://github.com/grpc-ecosystem/grpc-health-probe/releases/download/${GRPC_HEALTH_PROBE_VERSION}/grpc_health_probe-linux-amd64 && \
    chmod +x /bin/grpc_health_probe

WORKDIR /app

COPY --from=builder /app/go-judge .

COPY --from=builder /app/scripts /app/scripts
RUN chmod +x /app/scripts/*.sh

ENTRYPOINT ["./judge.docker-entrypoint.sh"]
