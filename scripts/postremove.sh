#!/bin/bash
# Post-removal script for ChView2

set -e

# Remove systemd service file
if [ -f /etc/systemd/system/clickhouse-schemaflow-visualizer.service ]; then
    echo "Removing systemd service file..."
    rm -f /etc/systemd/system/clickhouse-schemaflow-visualizer.service
    systemctl daemon-reload
fi

# Check if this is a purge operation (DEB packages pass $1 as "purge")
# or if REMOVE_USER_DATA environment variable is set
if [ "$1" = "purge" ] || [ "$REMOVE_USER_DATA" = "true" ]; then
    echo "Performing complete removal..."
    
    # Remove user data directories
    if [ -d /var/lib/clickhouse-schemaflow-visualizer ]; then
        rm -rf /var/lib/clickhouse-schemaflow-visualizer
        echo "Removed /var/lib/clickhouse-schemaflow-visualizer"
    fi
    
    if [ -d /var/log/clickhouse-schemaflow-visualizer ]; then
        rm -rf /var/log/clickhouse-schemaflow-visualizer
        echo "Removed /var/log/clickhouse-schemaflow-visualizer"
    fi
    
    # Remove configuration directory (only on purge)
    if [ -d /etc/clickhouse-schemaflow-visualizer ]; then
        rm -rf /etc/clickhouse-schemaflow-visualizer
        echo "Removed /etc/clickhouse-schemaflow-visualizer"
    fi
    
    # Remove system user
    if id "clickhouse-schemaflow-visualizer" &>/dev/null; then
        userdel clickhouse-schemaflow-visualizer
        echo "Removed system user 'clickhouse-schemaflow-visualizer'"
    fi
    
    echo "Complete removal finished"
else
    echo "Package removed but user data preserved"
    echo "To completely remove all data, run: sudo REMOVE_USER_DATA=true dpkg --purge clickhouse-schemaflow-visualizer"
    echo "Or manually remove: /var/lib/clickhouse-schemaflow-visualizer, /var/log/clickhouse-schemaflow-visualizer, /etc/clickhouse-schemaflow-visualizer"
fi

echo "ChView2 removal completed"
