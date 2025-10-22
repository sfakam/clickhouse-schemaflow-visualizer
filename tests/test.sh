#!/bin/bash

# Test runner script for ClickHouse Schema Flow Visualizer API tests
# Usage: ./test.sh [options]
# Options:
#   -v, --verbose     Show verbose output
#   -k EXPRESSION     Only run tests matching EXPRESSION
#   -m MARKER         Only run tests with the given marker
#   --cov             Run with coverage report
#   --html            Generate HTML coverage report

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Get the directory where this script is located
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
cd "$SCRIPT_DIR"

echo -e "${BLUE}================================${NC}"
echo -e "${BLUE}ClickHouse Schema Flow Visualizer${NC}"
echo -e "${BLUE}API Test Suite${NC}"
echo -e "${BLUE}================================${NC}\n"

# Check if virtual environment exists
if [ ! -d ".venv" ]; then
    echo -e "${YELLOW}Virtual environment not found. Creating...${NC}"
    python3 -m venv .venv
    echo -e "${GREEN}✓ Virtual environment created${NC}\n"
fi

# Activate virtual environment
echo -e "${BLUE}Activating virtual environment...${NC}"
source .venv/bin/activate
echo -e "${GREEN}✓ Virtual environment activated${NC}\n"

# Install/upgrade dependencies
echo -e "${BLUE}Installing/upgrading dependencies...${NC}"
pip install -q --upgrade pip
pip install -q -r requirements.txt
echo -e "${GREEN}✓ Dependencies installed${NC}\n"

# Check if server is running
echo -e "${BLUE}Checking if API server is running...${NC}"
if ! curl -s http://localhost:8080/api/databases > /dev/null 2>&1; then
    echo -e "${RED}✗ API server is not running at http://localhost:8080${NC}"
    echo -e "${YELLOW}Please start the server before running tests:${NC}"
    echo -e "${YELLOW}  cd .. && ./build.sh && ./start.sh${NC}"
    exit 1
fi
echo -e "${GREEN}✓ API server is running${NC}\n"

# Build pytest command with arguments
PYTEST_ARGS=()

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -v|--verbose)
            PYTEST_ARGS+=("-v")
            shift
            ;;
        -k)
            PYTEST_ARGS+=("-k" "$2")
            shift 2
            ;;
        -m)
            PYTEST_ARGS+=("-m" "$2")
            shift 2
            ;;
        --cov)
            PYTEST_ARGS+=("--cov=." "--cov-report=term-missing")
            shift
            ;;
        --html)
            PYTEST_ARGS+=("--cov=." "--cov-report=html")
            shift
            ;;
        *)
            PYTEST_ARGS+=("$1")
            shift
            ;;
    esac
done

# Run pytest
echo -e "${BLUE}Running tests...${NC}\n"
echo -e "${BLUE}================================${NC}\n"

if pytest "${PYTEST_ARGS[@]}" --tb=short --color=yes; then
    echo -e "\n${BLUE}================================${NC}"
    echo -e "${GREEN}✓ All tests passed!${NC}"
    echo -e "${BLUE}================================${NC}\n"
    exit 0
else
    echo -e "\n${BLUE}================================${NC}"
    echo -e "${RED}✗ Some tests failed${NC}"
    echo -e "${BLUE}================================${NC}\n"
    exit 1
fi
