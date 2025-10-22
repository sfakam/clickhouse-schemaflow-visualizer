#!/bin/bash

# Build script for ClickHouse Schema Flow Visualizer
# This script compiles the Go application and places the binary in the bin directory

set -e  # Exit on error

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
cd "$SCRIPT_DIR"

echo "Building ClickHouse Schema Flow Visualizer..."

# Create bin directory if it doesn't exist
mkdir -p bin

# Build the application
echo "Compiling Go application..."
go build -o bin/clickhouse-schemaflow-visualizer .

if [ $? -eq 0 ]; then
    echo "✓ Build successful!"
    echo "Binary created at: bin/clickhouse-schemaflow-visualizer"
    echo ""
    echo "To start the server, run: ./start.sh"
else
    echo "✗ Build failed!"
    exit 1
fi
