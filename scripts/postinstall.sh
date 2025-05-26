#!/bin/bash
# Post-installation script for ChView2

set -e

# Create systemd service file
cat > /etc/systemd/system/clickhouse-schemaflow-visualizer.service << 'EOF'
[Unit]
Description=ChView2 - ClickHouse Database Viewer
Documentation=https://github.com/FulgerX2007/clickhouse-schemaflow-visualizer
After=network.target
Wants=network.target

[Service]
Type=simple
User=clickhouse-schemaflow-visualizer
Group=clickhouse-schemaflow-visualizer
WorkingDirectory=/var/lib/clickhouse-schemaflow-visualizer
ExecStart=/usr/bin/clickhouse-schemaflow-visualizer
Restart=always
RestartSec=10
StandardOutput=journal
StandardError=journal

# Security settings
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/var/lib/clickhouse-schemaflow-visualizer /var/log/clickhouse-schemaflow-visualizer
CapabilityBoundingSet=CAP_NET_BIND_SERVICE
AmbientCapabilities=CAP_NET_BIND_SERVICE

[Install]
WantedBy=multi-user.target
EOF

# Set proper permissions for the service file
chmod 644 /etc/systemd/system/clickhouse-schemaflow-visualizer.service

# Copy example config if no config exists
if [ ! -f /etc/clickhouse-schemaflow-visualizer/config.env ]; then
    if [ -f /etc/clickhouse-schemaflow-visualizer/config.env.example ]; then
        cp /etc/clickhouse-schemaflow-visualizer/config.env.example /etc/clickhouse-schemaflow-visualizer/config.env
        chown root:clickhouse-schemaflow-visualizer /etc/clickhouse-schemaflow-visualizer/config.env
        chmod 640 /etc/clickhouse-schemaflow-visualizer/config.env
        echo "Created default configuration at /etc/clickhouse-schemaflow-visualizer/config.env"
    fi
fi

# Reload systemd and enable service
systemctl daemon-reload
systemctl enable clickhouse-schemaflow-visualizer.service

echo "ChView2 installed successfully!"
echo "To start the service: sudo systemctl start clickhouse-schemaflow-visualizer"
echo "To view logs: sudo journalctl -u clickhouse-schemaflow-visualizer -f"
echo "Configuration file: /etc/clickhouse-schemaflow-visualizer/config.env"
