# API Tests

This directory contains pytest-based tests for the ClickHouse Schema Flow Visualizer API endpoints.

## Quick Start

The easiest way to run tests is using the provided test script:

```bash
# Run all tests
./test.sh

# Run with verbose output
./test.sh -v

# Run only specific tests
./test.sh -k test_get_databases

# Run tests with a specific marker
./test.sh -m clean_api

# Run with coverage report
./test.sh --cov

# Run with HTML coverage report
./test.sh --html
```

## Manual Setup

If you prefer to run tests manually:

## Test Structure

```
tests/
├── __init__.py              # Package initialization
├── conftest.py              # Pytest configuration and fixtures
├── test_clean_api.py        # Tests for clean JSON API endpoints
└── test_mermaid_api.py      # Tests for Mermaid/visualization API endpoints
```

## Prerequisites

1. **Python 3.7+** installed
2. **ClickHouse Schema Flow Visualizer server** running on `http://localhost:8080`
3. **Test data** in ClickHouse (at least one database with tables)

## Installation

Install test dependencies:

```bash
pip install -r requirements-test.txt
```

Or install individually:

```bash
pip install pytest requests pytest-html pytest-cov
```

## Running Tests

### Run All Tests

```bash
pytest
```

### Run Specific Test Categories

```bash
## Test Organization

Tests are organized into two main categories:

### Clean JSON API Tests (`test_api_clean.py`)
Tests for the new clean JSON API endpoints that match the API.md specification:
- `GET /api/databases` - Returns databases and tables with metadata
- `GET /api/table/:database/:table` - Returns table columns
- `GET /api/table/:database/:table/relationships` - Returns table relationships

**Markers:** `@pytest.mark.clean_api`

### Mermaid API Tests (`test_api_mermaid.py`)
Tests for the Mermaid visualization API endpoints:
- `GET /api/mermaid/databases` - Returns HTML-formatted database/table list
- `GET /api/mermaid/schema/:database/:table` - Returns Mermaid diagram for single table
- `GET /api/mermaid/database/:database/schema` - Returns Mermaid diagram for entire database
- `GET /api/mermaid/database/:database/stats` - Returns database statistics

**Markers:** `@pytest.mark.mermaid_api`

## Running Specific Tests

### Run only Clean API tests
```bash
./test.sh -m clean_api
```

### Run only Mermaid API tests
```bash
./test.sh -m mermaid_api
```

### Run a specific test file
```bash
./test.sh test_api_clean.py
```

### Run tests matching a pattern
```bash
./test.sh -k "database"  # Runs all tests with "database" in the name
```

## Configuration

Test configuration is managed in `conftest.py`:
- `BASE_URL`: API server URL (default: `http://localhost:8080`)
- `TEST_DATABASE`: Database to use for tests (auto-detected from available databases)
- `TEST_TABLE`: Table to use for tests (auto-detected from available tables)

## Prerequisites

- Python 3.7+
- Running instance of ClickHouse Schema Flow Visualizer server
- Access to ClickHouse database with test data

## Continuous Integration

To integrate with CI/CD pipelines:

```bash
# Example for GitHub Actions or GitLab CI
./tests/test.sh --cov --html
```

## Troubleshooting

### Server not running
If you see `API server is not running`, start the server:
```bash
cd .. && ./build.sh && ./start.sh
```

### Tests failing due to missing data
Ensure your ClickHouse instance has at least one database with tables.

### Import errors
Make sure dependencies are installed:
```bash
cd tests
source .venv/bin/activate
pip install -r requirements.txt
```
```

### Run Specific Test Files

```bash
# Test clean JSON API endpoints
pytest tests/test_clean_api.py

# Test Mermaid API endpoints
pytest tests/test_mermaid_api.py
```

### Run Specific Test Classes

```bash
# Test clean databases endpoint
pytest tests/test_clean_api.py::TestCleanAPIDatabases

# Test Mermaid schema endpoint
pytest tests/test_mermaid_api.py::TestMermaidAPITableSchema
```

### Run Specific Test Functions

```bash
pytest tests/test_clean_api.py::TestCleanAPIDatabases::test_get_databases_returns_200
```

### Run with Coverage

```bash
# Generate coverage report
pytest --cov=. --cov-report=html

# View coverage report
open htmlcov/index.html
```

### Generate HTML Report

```bash
pytest --html=report.html --self-contained-html
```

## Configuration

### Environment Variables

Configure the test environment using environment variables:

```bash
# API base URL (default: http://localhost:8080)
export API_BASE_URL="http://localhost:8080"

# Test database name (default: owl)
export TEST_DATABASE="your_test_database"

# Test table name (default: sflows)
export TEST_TABLE="your_test_table"
```

### Custom Configuration

Edit `pytest.ini` to customize pytest behavior:

```ini
[pytest]
minversion = 6.0
testpaths = tests
addopts = -v --strict-markers --tb=short
```

## Test Markers

Tests are organized with the following markers:

- **`@pytest.mark.integration`** - Integration tests (require running server)
- **`@pytest.mark.clean_api`** - Clean JSON API tests
- **`@pytest.mark.mermaid_api`** - Mermaid/visualization API tests

## API Endpoints Tested

### Clean JSON API Endpoints (`/api/*`)

| Endpoint | Method | Tests |
|----------|--------|-------|
| `/api/databases` | GET | ✅ Status, JSON format, structure, non-empty |
| `/api/table/:database/:table` | GET | ✅ Status, JSON format, structure, error handling |
| `/api/table/:database/:table/relationships` | GET | ✅ Status, JSON format, structure, relationship types |

### Mermaid API Endpoints (`/api/mermaid/*`)

| Endpoint | Method | Tests |
|----------|--------|-------|
| `/api/mermaid/databases` | GET | ✅ Status, JSON format, HTML structure |
| `/api/mermaid/schema/:database/:table` | GET | ✅ Status, JSON format, Mermaid diagram |
| `/api/mermaid/database/:database/schema` | GET | ✅ Status, JSON format, filters, structure |
| `/api/mermaid/database/:database/stats` | GET | ✅ Status, JSON format, stats structure |

## Test Coverage

The test suite validates:

- ✅ **HTTP status codes** (200, 400, 404, 500)
- ✅ **Response content types** (application/json)
- ✅ **JSON structure** (correct fields, types, nesting)
- ✅ **Data validation** (non-empty, valid values)
- ✅ **Error handling** (missing params, nonexistent resources)
- ✅ **API specification compliance** (matches API.md)

## Common Issues

### Server Not Running

If you see "API server is not available", ensure the server is running:

```bash
./start.sh
```

### Test Database Not Found

If tests fail with database errors, configure the test database:

```bash
export TEST_DATABASE="your_database"
export TEST_TABLE="your_table"
pytest
```

### Permission Issues

Ensure ClickHouse allows connections from the test client.

## Continuous Integration

Example GitHub Actions workflow:

```yaml
name: API Tests

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2
      - uses: actions/setup-python@v2
        with:
          python-version: '3.9'
      - run: pip install -r requirements-test.txt
      - run: ./build.sh
      - run: ./start.sh &
      - run: sleep 5  # Wait for server to start
      - run: pytest --cov --html=report.html
```

## Contributing

When adding new API endpoints:

1. Add corresponding test class in `test_clean_api.py` or `test_mermaid_api.py`
2. Test all CRUD operations (if applicable)
3. Test error cases (400, 404, 500)
4. Validate response structure against API.md
5. Run full test suite: `pytest`

## Documentation

- **API Specification**: See [API.md](../API.md)
- **Project README**: See [README.md](../README.md)
