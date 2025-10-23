# API Documentation

## Overview

The ClickHouse Schema Flow Visualizer provides two sets of RESTful APIs:

1. **Clean JSON API** (`/api/*`) - Structured JSON responses for programmatic access
2. **Render/Visualization API** (`/api/render/*`) - Diagram and HTML-enhanced responses for UI rendering

The application supports both native TCP and HTTP ClickHouse clients.

## Base URLs

```
http://localhost:8080/api         # Clean JSON API
http://localhost:8080/api/render  # Render/Visualization API
```

## Client Configuration

The application supports dual ClickHouse client modes:

- **Native TCP Client**: Traditional ClickHouse driver (default, port 9000)
- **HTTP Client**: Custom HTTP implementation using Go's net/http (port 443, mTLS)

Configure via environment variable:
```bash
# Use HTTP client (port 443, mTLS)
CLICKHOUSE_USE_HTTP=true

# Use native TCP client (port 9000) - Default
CLICKHOUSE_USE_HTTP=false
```

---

## Clean JSON API Endpoints

These endpoints return clean, structured JSON suitable for programmatic access and integration.

### 1. Get Databases

**Endpoint**: `GET /api/databases`

**Description**: Returns all databases and their tables with metadata in structured JSON format.

**Parameters**: None

**Response Format**:
```json
{
  "database_name": [
    {
      "name": "table_name",
      "type": "table_engine_type",
      "rows": 1000000,
      "size": "50.2 MB"
    }
  ]
}
```

**Notes**:
- Distributed tables may not include `rows` or `size` fields
- Rows and size are only available for local tables

**Example Request**:
```bash
curl -X GET "http://localhost:8080/api/databases"
```

**Example Response**:
```json
{
  "default": [
    {
      "name": "users",
      "type": "MergeTree",
      "rows": 2500000,
      "size": "125.8 MB"
    },
    {
      "name": "events",
      "type": "ReplicatedMergeTree",
      "rows": 50000000,
      "size": "2.3 GB"
    }
  ],
  "analytics": [
    {
      "name": "page_views",
      "type": "MaterializedView",
      "rows": 15000000,
      "size": "800.5 MB"
    },
    {
      "name": "distributed_events",
      "type": "Distributed"
    }
  ]
}
```

---

### 2. Get Table Details

**Endpoint**: `GET /api/table/:database/:table`

**Description**: Returns detailed column information for a specific table in structured JSON format.

**Parameters**:
- `database` (path parameter, required): Database name
- `table` (path parameter, required): Table name

**Response Format**:
```json
[
  {
    "name": "column_name",
    "type": "column_type",
    "default_kind": "default_type",
    "default_expression": "default_value",
    "comment": "column_comment",
    "codec_expression": "compression_codec",
    "ttl_expression": "ttl_rule"
  }
]
```

**Notes**:
- Returns `null` or empty array if table doesn't exist
- Returns `400` if database or table parameters are missing
- All string fields may be empty (`""`) if not set in ClickHouse

**Example Request**:
```bash
curl -X GET "http://localhost:8080/api/table/default/users"
```

**Example Response**:
```json
[
  {
    "name": "id",
    "type": "UInt64",
    "default_kind": "",
    "default_expression": "",
    "comment": "User ID primary key",
    "codec_expression": "",
    "ttl_expression": ""
  },
  {
    "name": "username",
    "type": "String",
    "default_kind": "",
    "default_expression": "",
    "comment": "Username for login",
    "codec_expression": "LZ4",
    "ttl_expression": ""
  },
  {
    "name": "created_at",
    "type": "DateTime",
    "default_kind": "DEFAULT",
    "default_expression": "now()",
    "comment": "Account creation timestamp",
    "codec_expression": "",
    "ttl_expression": ""
  },
  {
    "name": "last_login",
    "type": "Nullable(DateTime)",
    "default_kind": "",
    "default_expression": "",
    "comment": "Last login timestamp",
    "codec_expression": "",
    "ttl_expression": ""
  }
]
```

---

### 3. Get Table Relationships

**Endpoint**: `GET /api/table/:database/:table/relationships`

**Description**: Returns relationships between tables based on dependencies (e.g., MaterializedViews, Distributed tables).

**Parameters**:
- `database` (path parameter, required): Database name
- `table` (path parameter, required): Table name

**Response Format**:
```json
[
  {
    "source_table": "table_name",
    "source_database": "database_name",
    "target_table": "related_table_name",
    "target_database": "related_database_name",
    "relationship_type": "depends_on"
  }
]
```

**Relationship Types**:
- `depends_on`: Source table depends on target table
- `depended_on_by`: Source table is depended on by target table

**Notes**:
- Returns empty array if no relationships exist
- Returns `400` if database or table parameters are missing

**Example Request**:
```bash
curl -X GET "http://localhost:8080/api/table/default/mv_user_stats/relationships"
```

**Example Response**:
```json
[
  {
    "source_table": "mv_user_stats",
    "source_database": "default",
    "target_table": "users",
    "target_database": "default",
    "relationship_type": "depends_on"
  },
  {
    "source_table": "mv_user_stats",
    "source_database": "default",
    "target_table": "events",
    "target_database": "default",
    "relationship_type": "depends_on"
  }
]
```

---

## Render/Visualization API Endpoints

These endpoints return HTML-enhanced or Mermaid diagram strings optimized for UI rendering.

### 1. Get Databases (Mermaid Format)

**Endpoint**: `GET /api/render/databases`

**Description**: Returns databases and tables with HTML-formatted strings including icons and formatted metadata.

**Parameters**: None

**Response Format**:
```json
{
  "database_name": {
    "table_name": "HTML_string_with_icons_and_metadata"
  }
}
```

**Example Request**:
```bash
curl -X GET "http://localhost:8080/api/render/databases"
```

**Example Response**:
```json
{
  "default": {
    "users": "<i class=\"fa-solid fa-circle-nodes\"></i> users<br><small style=\"color: #000; font-size: 0.8em;\">Rows: <b>2.5M</b> | Size: <b>125.8 MB</b></small>",
    "events": "<i class=\"fa-solid fa-circle-nodes\"></i> events<br><small style=\"color: #000; font-size: 0.8em;\">Rows: <b>50.0M</b> | Size: <b>2.3 GB</b></small>",
    "distributed_events": "<i class=\"fa-solid fa-diagram-project\"></i> distributed_events"
  }
}
```

**Icons Used**:
- `fa-circle-nodes`: Local tables (MergeTree, Replicated, etc.)
- `fa-diagram-project`: Distributed tables

---

### 2. Get Table Schema (Mermaid Diagram)

**Endpoint**: `GET /api/render/schema/:database/:table`

**Description**: Returns a Mermaid.js flowchart diagram showing table relationships.

**Parameters**:
- `database` (path parameter, required): Database name
- `table` (path parameter, required): Table name

**Response Format**:
```json
{
  "schema": "mermaid_diagram_string"
}
```

**Example Request**:
```bash
curl -X GET "http://localhost:8080/api/render/schema/default/users"
```

**Example Response**:
```json
{
  "schema": "flowchart TD\n    users[users<br/>MergeTree] --> events[events<br/>ReplicatedMergeTree]\n    events --> page_views[page_views<br/>MaterializedView]"
}
```

---

### 3. Get Database Schema (Mermaid Diagram)

**Endpoint**: `GET /api/render/database/:database/schema`

**Description**: Returns a comprehensive Mermaid.js flowchart for an entire database with optional filtering.

**Parameters**:
- `database` (path parameter, required): Database name
- `engines` (query parameter, optional): Filter by engine types (can be repeated)
- `metadata` (query parameter, optional): Include metadata tables (default: `true`)

**Response Format**:
```json
{
  "database": "database_name",
  "schema": "mermaid_diagram_string",
  "filters": {
    "engines": ["engine1", "engine2"],
    "metadata": true
  }
}
```

**Example Request**:
```bash
# Get all tables
curl -X GET "http://localhost:8080/api/render/database/default/schema"

# Filter by specific engines
curl -X GET "http://localhost:8080/api/render/database/default/schema?engines=MergeTree&engines=Distributed"

# Exclude metadata tables
curl -X GET "http://localhost:8080/api/render/database/default/schema?metadata=false"
```

**Example Response**:
```json
{
  "database": "default",
  "schema": "flowchart TD\n    users[users<br/>MergeTree]\n    events[events<br/>ReplicatedMergeTree]\n    users --> mv_stats[mv_stats<br/>MaterializedView]",
  "filters": {
    "engines": ["MergeTree", "Distributed"],
    "metadata": false
  }
}
```

---

### 4. Get Database Statistics

**Endpoint**: `GET /api/render/database/:database/stats`

**Description**: Returns comprehensive statistics for a database including table counts, row counts, and size by engine type.

**Parameters**:
- `database` (path parameter, required): Database name

**Response Format**:
```json
{
  "database": "database_name",
  "total_tables": 100,
  "total_rows": 1000000000,
  "total_bytes": 50000000000,
  "engine_counts": {
    "MergeTree": {
      "count": 50,
      "total_rows": 500000000,
      "total_bytes": 25000000000
    }
  }
}
```

**Example Request**:
```bash
curl -X GET "http://localhost:8080/api/render/database/default/stats"
```

**Example Response**:
```json
{
  "database": "default",
  "total_tables": 150,
  "total_rows": 750000000,
  "total_bytes": 40000000000,
  "engine_counts": {
    "MergeTree": {
      "count": 60,
      "total_rows": 400000000,
      "total_bytes": 20000000000
    },
    "ReplicatedMergeTree": {
      "count": 50,
      "total_rows": 300000000,
      "total_bytes": 15000000000
    },
    "MaterializedView": {
      "count": 20,
      "total_rows": 50000000,
      "total_bytes": 5000000000
    },
    "Distributed": {
      "count": 20,
      "total_rows": 0,
      "total_bytes": 0
    }
  }
}
```

---

## Error Handling

All endpoints return standard HTTP status codes and JSON error responses.

### HTTP Status Codes

- `200 OK`: Successful request
- `400 Bad Request`: Missing or invalid required parameters
- `404 Not Found`: Endpoint not found
- `500 Internal Server Error`: Database connection error or query failure

### Error Response Format

```json
{
  "error": "Error description message"
}
```

### Common Error Examples

**Missing Required Parameters (400)**:
```bash
# Missing database parameter
curl "http://localhost:8080/api/table//users"
```
```json
{
  "error": "database and table parameters are required"
}
```

**Nonexistent Table (200 with null)**:
```bash
# Table doesn't exist - returns empty result, not error
curl "http://localhost:8080/api/table/default/nonexistent_table"
```
```json
null
```

**Database Connection Error (500)**:
```json
{
  "error": "failed to connect to ClickHouse: connection refused"
}
```

**Query Execution Error (500)**:
```json
{
  "error": "failed to execute query: syntax error"
}
```

---

## API Comparison

| Feature | Clean JSON API (`/api/*`) | Render API (`/api/render/*`) |
|---------|---------------------------|--------------------------------|
| **Purpose** | Programmatic access | UI visualization |
| **Response Format** | Clean JSON structures | HTML/Mermaid strings |
| **Databases Endpoint** | Structured arrays | HTML with icons |
| **Table Details** | ✅ Available | ❌ Not available |
| **Relationships** | ✅ Available | ❌ Not available |
| **Table Schema** | ❌ Not available | ✅ Mermaid diagram |
| **Database Schema** | ❌ Not available | ✅ Mermaid diagram |
| **Database Stats** | ❌ Not available | ✅ Statistics |
| **Best For** | APIs, integrations, data processing | Frontend rendering, visualizations |

---

## Data Types

The API handles various ClickHouse data types including:

- **Numeric**: `UInt8`, `UInt16`, `UInt32`, `UInt64`, `Int8`, `Int16`, `Int32`, `Int64`, `Float32`, `Float64`
- **String**: `String`, `FixedString(N)`
- **Date/Time**: `Date`, `DateTime`, `DateTime64`
- **Nullable**: `Nullable(T)` for any type T
- **Arrays**: `Array(T)` for any type T
- **Complex**: `Tuple`, `Map`, `Nested`

## Table Engine Types

The API recognizes and displays various ClickHouse table engines:

- **MergeTree Family**: `MergeTree`, `ReplacingMergeTree`, `SummingMergeTree`, `AggregatingMergeTree`
- **Replicated**: `ReplicatedMergeTree`, `ReplicatedReplacingMergeTree`, etc.
- **Views**: `MaterializedView`, `View`
- **Special**: `Dictionary`, `Distributed`, `Memory`, `Log`

---

## Complete Workflow Examples

### Example 1: Clean JSON API Workflow

```bash
# 1. Get all databases and tables (clean JSON)
curl -X GET "http://localhost:8080/api/databases" | jq '.'

# 2. Get detailed column information for a specific table
curl -X GET "http://localhost:8080/api/table/default/users" | jq '.'

# 3. Get table relationships
curl -X GET "http://localhost:8080/api/table/default/users/relationships" | jq '.'
```

### Example 2: Render/Visualization API Workflow

```bash
# 1. Get databases with HTML formatting for UI
curl -X GET "http://localhost:8080/api/render/databases" | jq '.'

# 2. Get Mermaid diagram for a specific table
curl -X GET "http://localhost:8080/api/render/schema/default/users" | jq '.schema'

# 3. Get comprehensive database schema diagram
curl -X GET "http://localhost:8080/api/render/database/default/schema" | jq '.schema'

# 4. Get database statistics
curl -X GET "http://localhost:8080/api/render/database/default/stats" | jq '.'
```

### Example 3: JavaScript/Frontend Integration

```javascript
// Using Clean JSON API
async function getDatabaseInfo() {
  // Fetch all databases
  const databases = await fetch('/api/databases').then(r => r.json());
  console.log('Databases:', databases);
  
  // Get specific table columns
  const columns = await fetch('/api/table/default/users').then(r => r.json());
  console.log('User table columns:', columns);
  
  // Get relationships
  const relationships = await fetch('/api/table/default/users/relationships')
    .then(r => r.json());
  console.log('Relationships:', relationships);
}

// Using Render API for visualization
async function renderDatabaseSchema() {
  // Get Mermaid schema
  const response = await fetch('/api/render/database/default/schema')
    .then(r => r.json());
  
  // Render with Mermaid.js
  const element = document.querySelector('#diagram');
  const { svg } = await mermaid.render('diagram', response.schema);
  element.innerHTML = svg;
  
  // Get and display statistics
  const stats = await fetch('/api/render/database/default/stats')
    .then(r => r.json());
  console.log('Database statistics:', stats);
}
```

### Example 4: Python Integration

```python
import requests
import json

BASE_URL = "http://localhost:8080/api"

# Get all databases (clean JSON)
response = requests.get(f"{BASE_URL}/databases")
databases = response.json()
print(f"Found {len(databases)} databases")

# Get table details
response = requests.get(f"{BASE_URL}/table/default/users")
columns = response.json()
print(f"Users table has {len(columns)} columns")

# Get relationships
response = requests.get(f"{BASE_URL}/table/default/users/relationships")
relationships = response.json()
print(f"Found {len(relationships)} relationships")

# Get database statistics (Render API)
response = requests.get(f"{BASE_URL}/mermaid/database/default/stats")
stats = response.json()
print(f"Total tables: {stats['total_tables']}")
print(f"Total rows: {stats['total_rows']:,}")
```

---

## Authentication

Currently, the API uses the ClickHouse connection credentials configured in the environment variables. No additional API authentication is required.

## Rate Limiting

No rate limiting is currently implemented. Consider implementing rate limiting for production deployments.

## CORS

Cross-Origin Resource Sharing (CORS) headers are handled by the Gin framework. Configure as needed for cross-domain requests.

---

## Client Implementation Notes

### HTTP Client Mode

When `CLICKHOUSE_USE_HTTP=true`:
- Uses port 443 with mTLS authentication
- Implements custom HTTP client with Go's `net/http`
- Handles tab-separated value parsing
- Supports parameterized queries with `?` and `$N` placeholders
- Includes proper data type scanning for nullable types

### Native TCP Client Mode

When `CLICKHOUSE_USE_HTTP=false`:
- Uses port 9000 with native ClickHouse protocol
- Leverages official ClickHouse Go driver
- Provides better performance for large result sets
- Supports all ClickHouse features natively

## Troubleshooting

### Common Issues

1. **Connection Refused**: Verify ClickHouse server is running and accessible
2. **Authentication Failed**: Check username/password in environment variables
3. **TLS Errors**: Verify certificate paths and server name for HTTPS mode
4. **Empty Response**: Check database and table names for typos
5. **500 Errors**: Review server logs for detailed error messages

### Debug Mode

Enable debug logging:
```bash
GIN_MODE=debug
```

This will provide detailed request/response logging and client type information.