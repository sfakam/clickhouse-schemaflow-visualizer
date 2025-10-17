# API Documentation

## Overview

The ClickHouse Schema Flow Visualizer provides a RESTful API for exploring ClickHouse databases, tables, and their relationships. The API supports both native TCP and HTTP ClickHouse clients.

## Base URL

```
http://localhost:8080/api
```

## Client Configuration

The application supports dual ClickHouse client modes:

- **Native TCP Client**: Traditional ClickHouse driver (default)
- **HTTP Client**: Custom HTTP implementation using Go's net/http

Configure via environment variable:
```bash
# Use HTTP client (port 443, mTLS)
CLICKHOUSE_USE_HTTP=true

# Use native TCP client (port 9000)
CLICKHOUSE_USE_HTTP=false
```

## Endpoints

### 1. Get Databases

**Endpoint**: `GET /api/databases`

**Description**: Returns a list of all databases and their tables with metadata.

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
    }
  ]
}
```

### 2. Get Table Schema

**Endpoint**: `GET /api/schema/:database/:table`

**Description**: Returns a Mermaid.js diagram schema for visualizing table relationships.

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
curl -X GET "http://localhost:8080/api/schema/default/users"
```

**Example Response**:
```json
{
  "schema": "graph TD\n    users[users<br/>MergeTree] --> events[events<br/>ReplicatedMergeTree]\n    events --> page_views[page_views<br/>MaterializedView]"
}
```

### 3. Get Table Details

**Endpoint**: `GET /api/table/:database/:table`

**Description**: Returns detailed column information for a specific table.

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

## Error Handling

All endpoints return standard HTTP status codes and JSON error responses.

### HTTP Status Codes

- `200 OK`: Successful request
- `400 Bad Request`: Missing or invalid parameters
- `500 Internal Server Error`: Database connection error or query failure

### Error Response Format

```json
{
  "error": "Error description message"
}
```

### Common Error Examples

**Missing Parameters (400)**:
```json
{
  "error": "database and table parameters are required"
}
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
  "error": "table 'nonexistent_table' does not exist"
}
```

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

## Authentication

Currently, the API uses the ClickHouse connection credentials configured in the environment variables. No additional API authentication is required.

## Rate Limiting

No rate limiting is currently implemented. Consider implementing rate limiting for production deployments.

## CORS

Cross-Origin Resource Sharing (CORS) headers are handled by the Gin framework. Configure as needed for cross-domain requests.

## Examples

### Complete Workflow Example

```bash
# 1. Get all databases and tables
curl -X GET "http://localhost:8080/api/databases"

# 2. Get detailed information for a specific table
curl -X GET "http://localhost:8080/api/table/default/users"

# 3. Get Mermaid schema for visualization
curl -X GET "http://localhost:8080/api/schema/default/users"
```

### JavaScript/Frontend Integration

```javascript
// Fetch databases
const databases = await fetch('/api/databases').then(r => r.json());

// Fetch table details
const tableDetails = await fetch('/api/table/default/users').then(r => r.json());

// Fetch schema for visualization
const schema = await fetch('/api/schema/default/users').then(r => r.json());

// Render Mermaid diagram
mermaid.render('diagram', schema.schema);
```

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