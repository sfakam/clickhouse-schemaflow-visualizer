#!/bin/bash
# Pre-removal script for ChView2

set -e

# Stop and disable the service if it's running
if systemctl is-active --quiet clickhouse-schemaflow-visualizer.service; then
    echo "Stopping ChView2 service..."
    systemctl stop clickhouse-schemaflow-visualizer.service
fi

if systemctl is-enabled --quiet clickhouse-schemaflow-visualizer.service; then
    echo "Disabling ChView2 service..."
    systemctl disable clickhouse-schemaflow-visualizer.service
fi

echo "ChView2 service stopped and disabled"
