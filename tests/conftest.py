"""
Pytest configuration and fixtures
"""
import pytest
import requests
import os


# Base URL for the API
BASE_URL = os.getenv("API_BASE_URL", "http://localhost:8080")


@pytest.fixture(scope="session")
def api_base_url():
    """Return the base URL for the API"""
    return BASE_URL


@pytest.fixture(scope="session")
def api_client():
    """Create a requests session for API testing"""
    session = requests.Session()
    session.headers.update({"Content-Type": "application/json"})
    return session


@pytest.fixture(scope="session")
def test_database():
    """Return a test database name that should exist in ClickHouse"""
    # This should be configured based on your ClickHouse setup
    return os.getenv("TEST_DATABASE", "owl")


@pytest.fixture(scope="session")
def test_table():
    """Return a test table name that should exist in the test database"""
    return os.getenv("TEST_TABLE", "sflows")


@pytest.fixture
def verify_api_available(api_base_url, api_client):
    """Verify that the API is available before running tests"""
    try:
        response = api_client.get(f"{api_base_url}/api/render/databases", timeout=5)
        if response.status_code != 200:
            pytest.skip("API server is not available or not responding correctly")
    except requests.exceptions.RequestException:
        pytest.skip("API server is not available")


def pytest_configure(config):
    """Configure pytest with custom markers"""
    config.addinivalue_line(
        "markers", "integration: mark test as integration test (requires running server)"
    )
    config.addinivalue_line(
        "markers", "clean_api: mark test as clean JSON API test"
    )
    config.addinivalue_line(
        "markers", "mermaid_api: mark test as Mermaid API test"
    )
