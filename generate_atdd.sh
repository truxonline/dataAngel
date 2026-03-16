#!/bin/bash

STORIES=(
    "2-1-sidecar-litestream-backup-sqlite"
    "2-2-sidecar-rclone-sync-filesystem"
    "2-3-graceful-shutdown-with-wal-flush"
    "3-1-pre-backup-validation-sqlite-yaml"
    "3-2-post-restore-validation"
    "4-1-s3-distributed-lock-implementation"
    "4-2-lock-ttl-steal-mechanism"
    "5-1-prometheus-metrics-exporter"
    "5-2-alerting-backup-failure"
    "5-3-alerting-restore-performed"
    "6-1-cli-verify-backup-state"
    "6-2-cli-force-release-lock"
)

for story in "${STORIES[@]}"; do
    echo "Processing story: $story"
    
    story_num=$(echo $story | cut -d'-' -f1)
    mkdir -p pkg/$story_num
    mkdir -p cmd/cli
    
    touch pkg/${story_num}/${story}_test.go
    touch cmd/cli/${story}_test.go
    
    echo "Created test files for $story"
done

echo "All stories processed!"
