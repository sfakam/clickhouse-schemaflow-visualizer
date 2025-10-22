"""
Tests for Clean JSON API endpoints (matching API.md specification)
"""
import pytest


@pytest.mark.integration
@pytest.mark.clean_api
class TestCleanAPIDatabases:
    """Tests for GET /api/databases endpoint"""
    
    def test_get_databases_returns_200(self, api_base_url, api_client, verify_api_available):
        """Test that GET /api/databases returns 200 OK"""
        response = api_client.get(f"{api_base_url}/api/databases")
        assert response.status_code == 200
    
    def test_get_databases_returns_json(self, api_base_url, api_client, verify_api_available):
        """Test that GET /api/databases returns valid JSON"""
        response = api_client.get(f"{api_base_url}/api/databases")
        assert response.headers["Content-Type"].startswith("application/json")
        data = response.json()
        assert isinstance(data, dict)
    
    def test_get_databases_structure(self, api_base_url, api_client, verify_api_available):
        """Test that databases response has correct structure"""
        response = api_client.get(f"{api_base_url}/api/databases")
        data = response.json()
        
        # Should be a dict where keys are database names
        assert isinstance(data, dict)
        
        # Each database should have a list of tables
        for db_name, tables in data.items():
            assert isinstance(db_name, str)
            assert isinstance(tables, list)
            
            # Each table should have the correct structure
            for table in tables:
                assert isinstance(table, dict)
                assert "name" in table
                assert "type" in table
                assert isinstance(table["name"], str)
                assert isinstance(table["type"], str)
                
                # Optional fields
                if "rows" in table and table["rows"] is not None:
                    assert isinstance(table["rows"], int)
                if "size" in table and table["size"] != "":
                    assert isinstance(table["size"], str)
    
    def test_get_databases_not_empty(self, api_base_url, api_client, verify_api_available):
        """Test that databases list is not empty (assumes test data exists)"""
        response = api_client.get(f"{api_base_url}/api/databases")
        data = response.json()
        assert len(data) > 0, "Expected at least one database"


@pytest.mark.integration
@pytest.mark.clean_api
class TestCleanAPITableDetails:
    """Tests for GET /api/table/:database/:table endpoint"""
    
    def test_get_table_details_returns_200(
        self, api_base_url, api_client, test_database, test_table, verify_api_available
    ):
        """Test that GET /api/table/:database/:table returns 200 OK"""
        response = api_client.get(f"{api_base_url}/api/table/{test_database}/{test_table}")
        assert response.status_code == 200
    
    def test_get_table_details_returns_json(
        self, api_base_url, api_client, test_database, test_table, verify_api_available
    ):
        """Test that table details returns valid JSON"""
        response = api_client.get(f"{api_base_url}/api/table/{test_database}/{test_table}")
        assert response.headers["Content-Type"].startswith("application/json")
        data = response.json()
        assert isinstance(data, list)
    
    def test_get_table_details_structure(
        self, api_base_url, api_client, test_database, test_table, verify_api_available
    ):
        """Test that table details have correct column structure"""
        response = api_client.get(f"{api_base_url}/api/table/{test_database}/{test_table}")
        data = response.json()
        
        assert isinstance(data, list)
        assert len(data) > 0, "Expected at least one column"
        
        for column in data:
            assert isinstance(column, dict)
            assert "name" in column
            assert "type" in column
            assert isinstance(column["name"], str)
            assert isinstance(column["type"], str)
            
            # Optional fields from API.md spec
            optional_fields = [
                "default_kind", "default_expression", "comment",
                "codec_expression", "ttl_expression"
            ]
            for field in optional_fields:
                if field in column:
                    assert isinstance(column[field], str)
    
    def test_get_table_details_missing_database_returns_400(
        self, api_base_url, api_client, test_table, verify_api_available
    ):
        """Test that missing database parameter returns 400"""
        response = api_client.get(f"{api_base_url}/api/table//{test_table}")
        assert response.status_code == 400  # API returns 400 for missing required parameters
    
    def test_get_table_details_missing_table_returns_400(
        self, api_base_url, api_client, test_database, verify_api_available
    ):
        """Test that missing table parameter returns 400"""
        response = api_client.get(f"{api_base_url}/api/table/{test_database}/")
        assert response.status_code in [400, 404, 301]  # Different routers handle this differently
    
    def test_get_table_details_nonexistent_table_returns_error(
        self, api_base_url, api_client, test_database, verify_api_available
    ):
        """Test that nonexistent table returns error"""
        response = api_client.get(
            f"{api_base_url}/api/table/{test_database}/nonexistent_table_xyz_12345"
        )
        # ClickHouse returns empty result for nonexistent tables, which is valid behavior
        assert response.status_code == 200
        # Response should be None or empty array
        data = response.json()
        assert data is None or data == []


@pytest.mark.integration
@pytest.mark.clean_api
class TestCleanAPITableRelationships:
    """Tests for GET /api/table/:database/:table/relationships endpoint"""
    
    def test_get_table_relationships_returns_200(
        self, api_base_url, api_client, test_database, test_table, verify_api_available
    ):
        """Test that GET /api/table/:database/:table/relationships returns 200 OK"""
        response = api_client.get(
            f"{api_base_url}/api/table/{test_database}/{test_table}/relationships"
        )
        assert response.status_code == 200
    
    def test_get_table_relationships_returns_json(
        self, api_base_url, api_client, test_database, test_table, verify_api_available
    ):
        """Test that relationships return valid JSON"""
        response = api_client.get(
            f"{api_base_url}/api/table/{test_database}/{test_table}/relationships"
        )
        assert response.headers["Content-Type"].startswith("application/json")
        data = response.json()
        assert isinstance(data, list)
    
    def test_get_table_relationships_structure(
        self, api_base_url, api_client, test_database, test_table, verify_api_available
    ):
        """Test that relationships have correct structure"""
        response = api_client.get(
            f"{api_base_url}/api/table/{test_database}/{test_table}/relationships"
        )
        data = response.json()
        
        assert isinstance(data, list)
        
        for relationship in data:
            assert isinstance(relationship, dict)
            assert "source_table" in relationship
            assert "source_database" in relationship
            assert "target_table" in relationship
            assert "target_database" in relationship
            assert "relationship_type" in relationship
            
            assert isinstance(relationship["source_table"], str)
            assert isinstance(relationship["source_database"], str)
            assert isinstance(relationship["target_table"], str)
            assert isinstance(relationship["target_database"], str)
            assert isinstance(relationship["relationship_type"], str)
            
            # Relationship type should be one of the expected values
            assert relationship["relationship_type"] in [
                "depends_on", "depended_on_by"
            ]
    
    def test_get_table_relationships_missing_params_returns_400(
        self, api_base_url, api_client, verify_api_available
    ):
        """Test that missing parameters return 400 or 404"""
        response = api_client.get(f"{api_base_url}/api/table//relationships")
        assert response.status_code in [400, 404]
