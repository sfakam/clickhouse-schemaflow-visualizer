#!/bin/bash
# Pre-installation script for ChView2

set -e

# Create system user for clickhouse-schemaflow-visualizer if it doesn't exist
if ! id "clickhouse-schemaflow-visualizer" &>/dev/null; then
    useradd --system --home-dir /var/lib/clickhouse-schemaflow-visualizer --shell /bin/false clickhouse-schemaflow-visualizer
    echo "Created system user 'clickhouse-schemaflow-visualizer'"
fi

# Create necessary directories
mkdir -p /var/lib/clickhouse-schemaflow-visualizer
mkdir -p /var/log/clickhouse-schemaflow-visualizer
mkdir -p /etc/clickhouse-schemaflow-visualizer

# Set proper ownership
chown clickhouse-schemaflow-visualizer:clickhouse-schemaflow-visualizer /var/lib/clickhouse-schemaflow-visualizer
chown clickhouse-schemaflow-visualizer:clickhouse-schemaflow-visualizer /var/log/clickhouse-schemaflow-visualizer
chown root:clickhouse-schemaflow-visualizer /etc/clickhouse-schemaflow-visualizer

# Set proper permissions
chmod 755 /var/lib/clickhouse-schemaflow-visualizer
chmod 755 /var/log/clickhouse-schemaflow-visualizer
chmod 750 /etc/clickhouse-schemaflow-visualizer

echo "Pre-installation completed successfully"
