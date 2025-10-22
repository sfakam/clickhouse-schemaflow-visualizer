#!/bin/bash

# ClickHouse Schema Flow Visualizer Start Script
# This script kills any existing instance and starts the server in debug mode

echo "Starting ClickHouse Schema Flow Visualizer..."

# Change to the correct directory
cd /home/sfathall/git/sfathall/clickhouse-schemaflow-visualizer

# Kill any existing instances
echo "Checking for existing processes..."
if pgrep -f "./clickhouse-schemaflow-visualizer" > /dev/null; then
    echo "Found running process. Killing existing instance..."
    pkill -f "./clickhouse-schemaflow-visualizer"
    sleep 2
    echo "Process killed."
else
    echo "No existing process found."
fi

# Start the server in debug mode with logging
echo "Starting server in debug mode..."
./clickhouse-schemaflow-visualizer --debug > server.log 2>&1 &

# Get the process ID
SERVER_PID=$!

# Wait a moment for the server to start
sleep 3

# Check if the server started successfully
if ps -p $SERVER_PID > /dev/null; then
    echo "Server started successfully with PID: $SERVER_PID"
    echo "Server is running on http://localhost:8080"
    echo "Logs are being written to server.log"
    echo ""
    echo "To stop the server, run: pkill -f './clickhouse-schemaflow-visualizer'"
    echo "To view logs, run: tail -f server.log"
else
    echo "Failed to start server. Check server.log for errors."
    exit 1
fi