FROM rclone/rclone:latest

# Set working directory
WORKDIR /app

# Copy configuration
COPY rclone.conf /config/rclone/rclone.conf

# Set entrypoint
ENTRYPOINT ["rclone", "sync", "/data", "s3:backup-bucket", "--config", "/config/rclone/rclone.conf", "--transfers", "4", "--checkers", "8", "--stats", "60s"]