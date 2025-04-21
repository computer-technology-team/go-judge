# Use a minimal base image
FROM alpine:latest

# Install timeout utility
RUN apk add --no-cache coreutils

# Create a non-root user
RUN adduser -D appuser

# Set the working directory and ensure proper permissions
WORKDIR /app
RUN chown -R appuser:appuser /app

# Copy and set permissions for entrypoint script
COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh && chown appuser:appuser /entrypoint.sh

# Switch to non-root user
USER appuser

# Entry point script to handle timeout and exit status
ENTRYPOINT ["/bin/sh", "/entrypoint.sh"]