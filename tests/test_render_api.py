"""
Tests for Render/Visualization API endpoints
"""
import pytest


@pytest.mark.integration
@pytest.mark.render_api
class TestRenderAPIDatabases:
    """Tests for GET /api/render/databases endpoint"""
    
    def test_get_render_databases_returns_200(
        self, api_base_url, api_client, verify_api_available
    ):
        """Test that GET /api/render/databases returns 200 OK"""
        response = api_client.get(f"{api_base_url}/api/render/databases")
        assert response.status_code == 200
    
    def test_get_render_databases_returns_json(
        self, api_base_url, api_client, verify_api_available
    ):
        """Test that Render databases return valid JSON"""
        response = api_client.get(f"{api_base_url}/api/render/databases")
        assert response.headers["Content-Type"].startswith("application/json")
        data = response.json()
        assert isinstance(data, dict)
    
    def test_get_render_databases_html_format(
        self, api_base_url, api_client, verify_api_available
    ):
        """Test that Render databases return HTML strings"""
        response = api_client.get(f"{api_base_url}/api/render/databases")
        data = response.json()
        
        # Should be a dict where keys are database names
        assert isinstance(data, dict)
        
        # Each database should have a dict of tables with HTML strings
        for db_name, tables in data.items():
            assert isinstance(db_name, str)
            assert isinstance(tables, dict)
            
            for table_name, html_content in tables.items():
                assert isinstance(table_name, str)
                assert isinstance(html_content, str)
                # HTML should contain icon tags
                assert "<i class=" in html_content or table_name in html_content


@pytest.mark.integration
@pytest.mark.render_api
class TestRenderAPITableSchema:
    """Tests for GET /api/render/schema/:database/:table endpoint"""
    
    def test_get_render_schema_returns_200(
        self, api_base_url, api_client, test_database, test_table, verify_api_available
    ):
        """Test that GET /api/render/schema returns 200 OK"""
        response = api_client.get(
            f"{api_base_url}/api/render/schema/{test_database}/{test_table}"
        )
        assert response.status_code == 200
    
    def test_get_render_schema_returns_json(
        self, api_base_url, api_client, test_database, test_table, verify_api_available
    ):
        """Test that Render schema returns valid JSON"""
        response = api_client.get(
            f"{api_base_url}/api/render/schema/{test_database}/{test_table}"
        )
        assert response.headers["Content-Type"].startswith("application/json")
        data = response.json()
        assert isinstance(data, dict)
    
    def test_get_render_schema_structure(
        self, api_base_url, api_client, test_database, test_table, verify_api_available
    ):
        """Test that Render schema has correct structure"""
        response = api_client.get(
            f"{api_base_url}/api/render/schema/{test_database}/{test_table}"
        )
        data = response.json()
        
        assert "schema" in data
        assert isinstance(data["schema"], str)
        
        # Schema should be a Render diagram string
        schema = data["schema"]
        assert len(schema) > 0
        # Should contain erDiagram or flowchart keywords
        assert "erDiagram" in schema or "flowchart" in schema or "graph" in schema
    
    def test_get_render_schema_missing_params_returns_400(
        self, api_base_url, api_client, verify_api_available
    ):
        """Test that missing parameters return 400 or 404"""
        response = api_client.get(f"{api_base_url}/api/render/schema//")
        assert response.status_code in [400, 404]


@pytest.mark.integration
@pytest.mark.render_api
class TestRenderAPIDatabaseSchema:
    """Tests for GET /api/render/database/:database/schema endpoint"""
    
    def test_get_database_schema_returns_200(
        self, api_base_url, api_client, test_database, verify_api_available
    ):
        """Test that GET /api/render/database/:database/schema returns 200 OK"""
        response = api_client.get(
            f"{api_base_url}/api/render/database/{test_database}/schema"
        )
        assert response.status_code == 200
    
    def test_get_database_schema_returns_json(
        self, api_base_url, api_client, test_database, verify_api_available
    ):
        """Test that database schema returns valid JSON"""
        response = api_client.get(
            f"{api_base_url}/api/render/database/{test_database}/schema"
        )
        assert response.headers["Content-Type"].startswith("application/json")
        data = response.json()
        assert isinstance(data, dict)
    
    def test_get_database_schema_structure(
        self, api_base_url, api_client, test_database, verify_api_available
    ):
        """Test that database schema has correct structure"""
        response = api_client.get(
            f"{api_base_url}/api/render/database/{test_database}/schema"
        )
        data = response.json()
        
        assert "database" in data
        assert "schema" in data
        assert "filters" in data
        
        assert data["database"] == test_database
        assert isinstance(data["schema"], str)
        assert isinstance(data["filters"], dict)
        
        # Schema should contain Render diagram
        assert len(data["schema"]) > 0
        assert "flowchart" in data["schema"] or "erDiagram" in data["schema"]
    
    def test_get_database_schema_with_filters(
        self, api_base_url, api_client, test_database, verify_api_available
    ):
        """Test database schema with query filters"""
        response = api_client.get(
            f"{api_base_url}/api/render/database/{test_database}/schema",
            params={"engines": ["MergeTree", "Distributed"], "metadata": "false"}
        )
        assert response.status_code == 200
        data = response.json()
        
        assert "filters" in data
        assert "metadata" in data["filters"]
        assert data["filters"]["metadata"] is False
    
    def test_get_database_schema_missing_database_returns_400(
        self, api_base_url, api_client, verify_api_available
    ):
        """Test that missing database returns 400 or 404"""
        response = api_client.get(f"{api_base_url}/api/render/database//schema")
        assert response.status_code in [400, 404]


@pytest.mark.integration
@pytest.mark.render_api
class TestRenderAPIDatabaseStats:
    """Tests for GET /api/render/database/:database/stats endpoint"""
    
    def test_get_database_stats_returns_200(
        self, api_base_url, api_client, test_database, verify_api_available
    ):
        """Test that GET /api/render/database/:database/stats returns 200 OK"""
        response = api_client.get(
            f"{api_base_url}/api/render/database/{test_database}/stats"
        )
        assert response.status_code == 200
    
    def test_get_database_stats_returns_json(
        self, api_base_url, api_client, test_database, verify_api_available
    ):
        """Test that database stats return valid JSON"""
        response = api_client.get(
            f"{api_base_url}/api/render/database/{test_database}/stats"
        )
        assert response.headers["Content-Type"].startswith("application/json")
        data = response.json()
        assert isinstance(data, dict)
    
    def test_get_database_stats_structure(
        self, api_base_url, api_client, test_database, verify_api_available
    ):
        """Test that database stats have correct structure"""
        response = api_client.get(
            f"{api_base_url}/api/render/database/{test_database}/stats"
        )
        data = response.json()
        
        assert "database" in data
        assert "total_tables" in data
        assert "total_rows" in data
        assert "total_bytes" in data
        assert "engine_counts" in data
        
        assert data["database"] == test_database
        assert isinstance(data["total_tables"], int)
        assert isinstance(data["total_rows"], int)
        assert isinstance(data["total_bytes"], int)
        assert isinstance(data["engine_counts"], dict)
        
        # Each engine should have stats
        for engine, stats in data["engine_counts"].items():
            assert isinstance(engine, str)
            assert isinstance(stats, dict)
            assert "count" in stats
            assert "total_rows" in stats
            assert "total_bytes" in stats
    
    def test_get_database_stats_missing_database_returns_400(
        self, api_base_url, api_client, verify_api_available
    ):
        """Test that missing database returns 400 or 404"""
        response = api_client.get(f"{api_base_url}/api/render/database//stats")
        assert response.status_code in [400, 404]
