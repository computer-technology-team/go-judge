# This is a copy of the first stage of the Dockerfile in .docker

FROM alpine:latest AS code-executer

RUN apk add --no-cache coreutils

RUN adduser -D appuser

WORKDIR /app

RUN chown -R appuser:appuser /app

COPY entrypoint-symlink.sh /entrypoint.sh

RUN chmod +x /entrypoint.sh && chown appuser:appuser /entrypoint.sh

USER appuser

ENTRYPOINT ["/bin/sh", "/entrypoint.sh"]