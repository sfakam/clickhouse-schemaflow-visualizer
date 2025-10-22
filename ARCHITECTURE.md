# Architecture Documentation

## Overview

ClickHouse Schema Flow Visualizer is a web-based tool for visualizing and exploring ClickHouse database schemas. It uses a client-server architecture with a Go backend and a vanilla JavaScript frontend, generating interactive Mermaid.js diagrams to visualize table relationships.

## Table of Contents

- [Architecture Overview](#architecture-overview)
- [Technology Stack](#technology-stack)
- [System Components](#system-components)
- [Data Flow](#data-flow)
- [Caching Strategy](#caching-strategy)
- [Database Schema Analysis](#database-schema-analysis)
- [API Design](#api-design)
- [Frontend Architecture](#frontend-architecture)
- [Build and Deployment](#build-and-deployment)

---

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────┐
│                         Browser                              │
│  ┌─────────────┐  ┌──────────────┐  ┌─────────────────┐    │
│  │   HTML/CSS  │  │  JavaScript  │  │  Mermaid.js     │    │
│  │   (UI)      │  │  (app.js)    │  │  (Rendering)    │    │
│  └──────┬──────┘  └──────┬───────┘  └────────┬────────┘    │
│         │                │                    │              │
└─────────┼────────────────┼────────────────────┼──────────────┘
          │                │                    │
          │         HTTP/JSON API               │
          │                │                    │
┌─────────▼────────────────▼────────────────────▼──────────────┐
│                     Go Backend (Gin)                          │
│  ┌──────────────────────────────────────────────────────┐    │
│  │              API Handlers (handlers.go)              │    │
│  │  • GetDatabases  • GetTableSchema                    │    │
│  │  • GetTableDetails  • GetDatabaseSchema              │    │
│  └────────────────────┬─────────────────────────────────┘    │
│                       │                                       │
│  ┌────────────────────▼─────────────────────────────────┐    │
│  │         Business Logic (clickhouse.go)               │    │
│  │  ┌────────────────────────────────────────────────┐  │    │
│  │  │          TableCache (Single Source)            │  │    │
│  │  │  • Cached table metadata                       │  │    │
│  │  │  • Relationship graph                          │  │    │
│  │  │  • Thread-safe with RWMutex                    │  │    │
│  │  │  • 5-minute TTL                                 │  │    │
│  │  └────────────────────────────────────────────────┘  │    │
│  │                                                        │    │
│  │  ┌────────────────────────────────────────────────┐  │    │
│  │  │      Dual Client Architecture                  │  │    │
│  │  │  • Native TCP Client (default)                 │  │    │
│  │  │  • HTTP Client (fallback/alternative)          │  │    │
│  │  └────────────────────────────────────────────────┘  │    │
│  └────────────────────┬─────────────────────────────────┘    │
└───────────────────────┼──────────────────────────────────────┘
                        │
                        │ ClickHouse Protocol
                        │ (TCP:9000 or HTTP:8123)
                        │
┌───────────────────────▼──────────────────────────────────────┐
│                    ClickHouse Server                          │
│  • system.tables (metadata)                                   │
│  • system.columns (column info)                               │
│  • User databases and tables                                  │
└───────────────────────────────────────────────────────────────┘
```

---

## Technology Stack

### Backend
- **Language**: Go 1.21+
- **Web Framework**: Gin (HTTP router and middleware)
- **ClickHouse Client**: `github.com/ClickHouse/clickhouse-go/v2`
- **Hashing**: `github.com/go-faster/city` (for node ID generation)

### Frontend
- **UI**: Vanilla HTML5/CSS3
- **JavaScript**: ES6+ (no framework)
- **Diagram Rendering**: Mermaid.js 10.6.1
- **Icons**: Font Awesome 6.5.1

### Infrastructure
- **Configuration**: Environment variables (.env)
- **Logging**: Standard Go logging
- **Containerization**: Docker & Docker Compose

---

## System Components

### 1. Main Entry Point (`main.go`)

**Responsibilities:**
- Application initialization
- Environment configuration loading
- ClickHouse client creation with dual-mode support
- Gin router setup and middleware configuration
- Server startup and binding

**Key Features:**
- Debug mode flag support
- IPv4/IPv6 binding configuration
- Graceful configuration error handling
- TLS/SSL certificate support

### 2. API Layer (`api/handlers.go`)

**Components:**

#### Handler Structure
```go
type Handler struct {
    clickhouse *models.ClickHouseClient
}
```

#### Endpoints

| Endpoint | Method | Purpose |
|----------|--------|---------|
| `/api/databases` | GET | List all databases with tables |
| `/api/schema/:database/:table` | GET | Get Mermaid schema for single table |
| `/api/database/:database/schema` | GET | Get Mermaid schema for entire database |
| `/api/table/:database/:table` | GET | Get detailed table information |

**Design Principles:**
- RESTful API design
- JSON response format
- Consistent error handling
- Parameter validation

### 3. Business Logic Layer (`models/clickhouse.go`)

#### Data Structures

**TableCache** - Single Source of Truth
```go
type TableCache struct {
    Tables       map[string]*CachedTableData  // key: database.table
    Relations    []TableRelation              // relationship graph
    DatabasesMap map[string]map[string]string // UI hierarchy
    LastRefresh  time.Time                    // cache timestamp
    mutex        sync.RWMutex                 // thread-safety
}
```

**CachedTableData** - Comprehensive Table Info
```go
type CachedTableData struct {
    Name                        string
    Database                    string
    Engine                      string
    EngineFullMeta             string
    CreateQuery                string
    TotalRows                  *uint64
    TotalBytes                 *uint64
    LoadingDependenciesDatabase []string
    LoadingDependenciesTable   []string
    Icon                       string
    LastUpdated                time.Time
}
```

**TableRelation** - Relationship Representation
```go
type TableRelation struct {
    Table          string
    DependsOnTable string
    Icon           string
}
```

#### Key Functions

**Cache Management:**
- `refreshTableCache()` - Single comprehensive query to populate cache
- `getTableFromCache()` - Thread-safe cache retrieval
- `getEngineIcon()` - Icon assignment by engine type
- `extractRelationsFromQuery()` - Parse CREATE queries for relationships

**Schema Generation:**
- `GenerateMermaidSchema()` - Single table visualization
- `GenerateDatabaseMermaidSchema()` - Full database visualization
- `getRelationsNext()` - Forward relationship traversal
- `getRelationsBack()` - Backward relationship traversal

**Data Retrieval:**
- `GetDatabases()` - Database listing
- `GetTableColumns()` - Table details with columns
- `getTableEngines()` - Engine information

### 4. Dual Client Architecture

**Native TCP Client** (Default)
```go
type NativeClient struct {
    conn clickhouse.Conn
}
```
- Port: 9000
- Protocol: ClickHouse native TCP
- Performance: Optimal for production
- Features: Full protocol support

**HTTP Client** (Alternative)
```go
type HTTPClient struct {
    baseURL  string
    username string
    password string
    client   *http.Client
}
```
- Port: 8123
- Protocol: HTTP/HTTPS
- Use case: Firewall restrictions, proxy environments
- Features: TLS/SSL support, certificate authentication

**Selection Logic:**
- Environment variable `CLICKHOUSE_USE_HTTP`
- Automatic fallback on connection failure
- Transparent to application layer

### 5. Frontend Architecture (`static/`)

#### Structure
```
static/
├── css/
│   └── styles.css       # Component-based styling
├── html/
│   └── index.html       # Single-page application
├── js/
│   └── app.js          # Application logic
└── img/
    ├── logo.png
    ├── logo_256x256.png
    └── favicon.ico
```

#### JavaScript Architecture (`app.js`)

**State Management:**
```javascript
// Global state
let currentDatabase = null;
let currentTable = null;
let mermaidInitialized = false;
```

**Core Functions:**

| Function | Purpose |
|----------|---------|
| `loadDatabases()` | Fetch and render database tree |
| `loadTableDetails()` | Fetch table metadata and columns |
| `loadSchema()` | Generate and display Mermaid diagram |
| `renderTableDetails()` | Display table information panel |
| `copyCreateQuery()` | Copy CREATE statement to clipboard |

**UI Components:**
- Collapsible sidebar with database tree
- Table details panel with metadata
- Mermaid diagram canvas
- Loading states and error handling

---

## Data Flow

### 1. Application Startup Flow

```
┌─────────────────┐
│  Start Server   │
└────────┬────────┘
         │
         ▼
┌─────────────────────────────┐
│ Load Environment Config     │
│ (.env variables)            │
└────────┬────────────────────┘
         │
         ▼
┌─────────────────────────────┐
│ Initialize ClickHouse Client│
│ (TCP or HTTP based on cfg)  │
└────────┬────────────────────┘
         │
         ▼
┌─────────────────────────────┐
│ Test Connection (Ping)      │
└────────┬────────────────────┘
         │
         ▼
┌─────────────────────────────┐
│ Initialize TableCache       │
│ (empty initially)           │
└────────┬────────────────────┘
         │
         ▼
┌─────────────────────────────┐
│ Setup Gin Routes & Start    │
│ HTTP Server (port 8080)     │
└─────────────────────────────┘
```

### 2. First Request Flow (Cache Population)

```
┌──────────────────┐
│ Browser Requests │
│ /api/databases   │
└────────┬─────────┘
         │
         ▼
┌─────────────────────────────┐
│ GetDatabases() handler      │
└────────┬────────────────────┘
         │
         ▼
┌─────────────────────────────┐
│ refreshTableCache()         │
│ Check: LastRefresh time     │
└────────┬────────────────────┘
         │
         ▼ (Cache empty or expired)
┌─────────────────────────────────────────────┐
│ Single Comprehensive Query to system.tables │
│ SELECT database, name, engine, engine_full, │
│        create_table_query, total_rows,      │
│        total_bytes, loading_dependencies... │
└────────┬────────────────────────────────────┘
         │
         ▼
┌─────────────────────────────┐
│ Process Each Row:           │
│ • Create CachedTableData    │
│ • Extract relationships     │
│ • Assign engine icons       │
│ • Build DatabasesMap        │
└────────┬────────────────────┘
         │
         ▼
┌─────────────────────────────┐
│ Store in TableCache         │
│ Set LastRefresh = now()     │
└────────┬────────────────────┘
         │
         ▼
┌─────────────────────────────┐
│ Return JSON response        │
│ with database hierarchy     │
└─────────────────────────────┘
```

### 3. Table Details Request Flow

```
┌──────────────────────────┐
│ Browser Requests         │
│ /api/table/:db/:table    │
└────────┬─────────────────┘
         │
         ▼
┌─────────────────────────────┐
│ GetTableDetails() handler   │
└────────┬────────────────────┘
         │
         ▼
┌─────────────────────────────┐
│ getTableFromCache()         │
│ (uses cached metadata)      │
└────────┬────────────────────┘
         │
         ▼
┌─────────────────────────────┐
│ Query system.columns        │
│ (only column info needed)   │
└────────┬────────────────────┘
         │
         ▼
┌─────────────────────────────┐
│ formatCreateQuery()         │
│ • Add line breaks           │
│ • Indent columns            │
│ • Format SQL keywords       │
└────────┬────────────────────┘
         │
         ▼
┌─────────────────────────────┐
│ Return TableDetails JSON    │
│ • Metadata from cache       │
│ • Columns from query        │
│ • Formatted CREATE query    │
└─────────────────────────────┘
```

### 4. Schema Visualization Flow

```
┌──────────────────────────────┐
│ Browser Requests             │
│ /api/schema/:db/:table       │
└────────┬─────────────────────┘
         │
         ▼
┌─────────────────────────────────┐
│ GenerateMermaidSchema()         │
└────────┬────────────────────────┘
         │
         ▼
┌─────────────────────────────────┐
│ Get relations from cache        │
└────────┬────────────────────────┘
         │
         ▼
┌─────────────────────────────────┐
│ Build Mermaid flowchart:        │
│ 1. Create main table node       │
│ 2. Traverse forward relations   │
│ 3. Traverse backward relations  │
│ 4. Add styles and metadata      │
└────────┬────────────────────────┘
         │
         ▼
┌─────────────────────────────────┐
│ Return Mermaid syntax string    │
└────────┬────────────────────────┘
         │
         ▼
┌─────────────────────────────────┐
│ Frontend renders with           │
│ Mermaid.js library              │
└─────────────────────────────────┘
```

---

## Caching Strategy

### Cache Architecture

**Design Goals:**
- Minimize database queries
- Provide fast response times
- Ensure data consistency
- Support concurrent access

### Cache Lifecycle

```
┌─────────────────────────────────────────────────────────┐
│                    Cache States                          │
├─────────────────────────────────────────────────────────┤
│                                                          │
│  Empty/Expired        Fresh             Stale           │
│       │                │                  │             │
│       │                │                  │             │
│       ▼                ▼                  ▼             │
│  ┌─────────┐     ┌─────────┐       ┌─────────┐        │
│  │ Refresh │────▶│  Serve  │──────▶│ Refresh │        │
│  │ Cache   │     │  from   │ 5min  │ Cache   │        │
│  └─────────┘     │  Cache  │       └─────────┘        │
│                  └─────────┘                            │
└─────────────────────────────────────────────────────────┘
```

**Refresh Triggers:**
1. Cache empty (first request)
2. TTL expired (5 minutes since last refresh)
3. Manual refresh (if implemented)

**Thread Safety:**
- `RWMutex` for concurrent read/write access
- Write lock during cache refresh
- Read lock for all cache access
- No blocking on cache hits

### Performance Metrics

**Before Optimization:**
- Database listing: ~500ms (queries system.tables)
- Table details: ~50ms (queries system.tables + columns)
- Schema generation: ~100ms (queries system.tables + relations)
- **Total: 3+ queries per page load**

**After Optimization:**
- Initial cache load: ~1.3s (single comprehensive query for 343 tables)
- Database listing: ~0.05-0.1ms (cache read)
- Table details: ~25ms (only queries columns)
- Schema generation: ~0.05-0.1ms (cache read)
- **Total: 1 query on startup, minimal queries thereafter**

**Improvement: ~1000x faster for schema operations**

---

## Database Schema Analysis

### Supported Table Engines

The system recognizes and visualizes relationships for these ClickHouse table engines:

#### MergeTree Family
- **MergeTree** - Basic table engine
- **ReplicatedMergeTree** - Distributed replication
- **SummingMergeTree** - Pre-aggregated data
- **ReplacingMergeTree** - Deduplication support
- **AggregatingMergeTree** - Pre-computed aggregations
- **CollapsingMergeTree** - State collapse logic
- **VersionedCollapsingMergeTree** - Versioned state
- **GraphiteMergeTree** - Time-series data

#### Special Engines
- **MaterializedView** - Query-based views with automatic updates
- **View** - Virtual tables
- **Dictionary** - External data sources
- **Distributed** - Sharded table access
- **Memory** - In-memory tables
- **Log Family** - Append-only logs

### Relationship Detection

**Relationship Types:**

1. **Distributed → Local Tables**
   ```sql
   ENGINE = Distributed(cluster, database, table, ...)
   ```
   Parsed from: `engine_full` metadata

2. **MaterializedView → Source + Target**
   ```sql
   CREATE MATERIALIZED VIEW mv TO target AS SELECT ... FROM source
   ```
   Parsed from: `create_table_query`

3. **Dictionary → Source Tables**
   ```sql
   CREATE DICTIONARY dict ... SOURCE(...)
   ```
   Parsed from: `loading_dependencies_database/table`

### Relationship Extraction Algorithm

```go
func extractRelationsFromQuery(database, table, engine, engineFull, 
                                createQuery string, loadingDepsDB, 
                                loadingDepsTable []string) []TableRelation {
    switch engine {
    case "Distributed":
        // Parse: Distributed('cluster', 'db', 'table', ...)
        // Extract target database and table from engine_full
        
    case "MaterializedView":
        // Parse: CREATE MV view TO target AS SELECT FROM source
        // Extract both source and target relationships
        
    case "Dictionary":
        // Use loading_dependencies metadata
        // Create dependency relationship
        
    default:
        // Create basic table node
    }
}
```

---

## API Design

### REST API Principles

**Design Philosophy:**
- Resource-oriented URLs
- HTTP methods for operations (GET)
- JSON for data exchange
- Consistent error handling
- Stateless requests

### API Endpoints Reference

#### 1. Get Databases
```
GET /api/databases
```

**Response:**
```json
{
  "database1": {
    "table1": "<i>icon</i> table1<br>Rows: 1.2M | Size: 45.3MB",
    "table2": "<i>icon</i> table2<br>Rows: 890K | Size: 32.1MB"
  },
  "database2": { ... }
}
```

#### 2. Get Table Schema
```
GET /api/schema/:database/:table
```

**Response:**
```json
{
  "schema": "flowchart TB\n    12345[\"db.table...\"]\n    ..."
}
```

#### 3. Get Database Schema
```
GET /api/database/:database/schema?engines[]=MergeTree&metadata=true
```

**Query Parameters:**
- `engines[]` - Filter by engine types (array)
- `metadata` - Include metadata (default: true)

**Response:**
```json
{
  "database": "mydb",
  "schema": "flowchart LR\n    ...",
  "filters": {
    "engines": ["MergeTree"],
    "metadata": true
  }
}
```

#### 4. Get Table Details
```
GET /api/table/:database/:table
```

**Response:**
```json
{
  "name": "table_name",
  "database": "db_name",
  "engine": "MergeTree",
  "total_rows": 1000000,
  "total_bytes": 45000000,
  "columns": [
    {
      "name": "id",
      "type": "UInt64",
      "position": 1,
      "comment": ""
    }
  ],
  "create_query": "CREATE TABLE db.table (...)"
}
```

### Error Handling

**Error Response Format:**
```json
{
  "error": "descriptive error message"
}
```

**HTTP Status Codes:**
- `200` - Success
- `400` - Bad Request (missing parameters)
- `404` - Not Found (database/table doesn't exist)
- `500` - Internal Server Error (database connection, query failure)

---

## Frontend Architecture

### Single Page Application (SPA) Design

**No Framework Philosophy:**
- Vanilla JavaScript for simplicity
- Direct DOM manipulation
- Event-driven architecture
- Modular function organization

### Component Breakdown

#### 1. Sidebar (Database Tree)
```javascript
// Collapsible database browser
// Features:
// - Lazy loading of databases
// - Expandable/collapsible sections
// - Table selection
// - Visual feedback on selection
```

#### 2. Table Details Panel
```javascript
// Right sidebar with table information
// Sections:
// - Table metadata (engine, rows, size)
// - Column list with types
// - CREATE query with syntax formatting
// - Copy-to-clipboard functionality
```

#### 3. Main Canvas (Mermaid Diagram)
```javascript
// Central visualization area
// Features:
// - Dynamic diagram rendering
// - Zoom and pan (Mermaid built-in)
// - Loading states
// - Error handling
```

### State Management

**Local Storage:**
```javascript
// Persisted UI state
localStorage.setItem('sidebarVisible', isVisible);
localStorage.setItem('tableDetailsVisible', isVisible);
```

**Session State:**
```javascript
// Runtime state (not persisted)
let currentDatabase = null;
let currentTable = null;
let mermaidInitialized = false;
```

### Event Flow

```
User clicks table
      │
      ▼
selectTable(database, table)
      │
      ├──▶ loadSchema(database, table)
      │         │
      │         └──▶ Fetch /api/schema/:db/:table
      │                   │
      │                   └──▶ renderMermaid(schema)
      │
      └──▶ loadTableDetails(database, table)
                  │
                  └──▶ Fetch /api/table/:db/:table
                            │
                            └──▶ renderTableDetails(details)
```

---

## Build and Deployment

### Development Build Process

```bash
# 1. Build binary
./build.sh
   │
   ├─ Create bin/ directory
   ├─ Compile: go build -o bin/clickhouse-schemaflow-visualizer
   └─ Report success/failure

# 2. Start server
./start.sh
   │
   ├─ Check binary exists (suggest build.sh if missing)
   ├─ Kill existing process (pkill -9)
   ├─ Start server with debug flag
   └─ Monitor process health
```

### Directory Structure

```
clickhouse-schemaflow-visualizer/
├── bin/                    # Compiled binaries (gitignored)
│   └── clickhouse-schemaflow-visualizer
├── api/                    # HTTP handlers
├── config/                 # Configuration management
├── models/                 # Business logic & data models
├── static/                 # Frontend assets
│   ├── css/
│   ├── html/
│   ├── img/
│   └── js/
├── .env.example            # Configuration template
├── build.sh                # Build script
├── start.sh                # Start script
├── main.go                 # Application entry point
└── go.mod                  # Go dependencies
```

### Configuration

**Environment Variables:**
```bash
# ClickHouse Connection
CLICKHOUSE_HOST=localhost
CLICKHOUSE_PORT=9000
CLICKHOUSE_USER=default
CLICKHOUSE_PASSWORD=

# Connection Mode
CLICKHOUSE_USE_HTTP=false    # false=TCP, true=HTTP

# TLS/SSL (for secure connections)
CLICKHOUSE_SECURE=false
CLICKHOUSE_SKIP_VERIFY=false
CLICKHOUSE_CERT_PATH=
CLICKHOUSE_KEY_PATH=
CLICKHOUSE_CA_PATH=
CLICKHOUSE_SERVER_NAME=

# Server
SERVER_ADDR=:8080
GIN_MODE=release             # release or debug
```

### Docker Deployment

**Build:**
```bash
docker build -t clickhouse-schemaflow-visualizer .
```

**Run:**
```bash
docker run -p 8080:8080 --env-file .env clickhouse-schemaflow-visualizer
```

**Docker Compose:**
```yaml
version: '3.8'
services:
  visualizer:
    build: .
    ports:
      - "8080:8080"
    environment:
      - CLICKHOUSE_HOST=clickhouse
      - CLICKHOUSE_PORT=9000
    depends_on:
      - clickhouse
```

---

## Performance Characteristics

### Query Performance

**Cache Hit (99% of requests after warm-up):**
- Database list: < 0.1ms
- Schema generation: < 0.1ms
- Table metadata: < 0.1ms

**Cache Miss (first request or after TTL):**
- Initial cache load: ~1.3s (343 tables)
- Scales linearly with table count

**Column Queries (not cached):**
- Table details with columns: ~25ms
- Depends on column count

### Memory Usage

**Cache Size:**
- ~1-2 KB per table (metadata)
- 343 tables ≈ 500 KB total
- Negligible for modern systems

**Concurrent Requests:**
- Thread-safe with RWMutex
- Multiple readers don't block
- Single writer during refresh

### Scalability Considerations

**Current Limits:**
- Tested with 343 tables across 5 databases
- Supports thousands of tables efficiently
- Linear memory growth with table count

**Potential Bottlenecks:**
- Very large databases (10,000+ tables)
- Complex relationship graphs
- Frontend rendering of massive diagrams

**Mitigation Strategies:**
- Database-level filtering
- Pagination for large schemas
- On-demand relationship loading
- Configurable cache TTL

---

## Security Considerations

### Authentication
- Supports ClickHouse username/password
- TLS/SSL certificate-based authentication
- Credentials via environment variables (not hardcoded)

### Connection Security
- Native TLS support for TCP connections
- HTTPS support for HTTP mode
- Certificate validation (configurable skip)
- Server name verification

### Best Practices
- Never commit `.env` files
- Use read-only ClickHouse users
- Deploy behind reverse proxy (nginx/Traefik)
- Enable TLS in production
- Restrict network access to ClickHouse

---

## Future Architecture Considerations

### Potential Enhancements

1. **Real-time Updates**
   - WebSocket connection for live schema changes
   - Server-sent events for cache updates

2. **Advanced Caching**
   - Redis/Memcached integration
   - Distributed cache for multi-instance deployments
   - Per-database cache granularity

3. **Query Optimization**
   - Incremental cache updates
   - Differential sync for large databases
   - Column metadata caching

4. **Visualization Enhancements**
   - Export to PNG/SVG/PDF
   - Custom diagram layouts
   - Interactive filtering
   - Search functionality

5. **Multi-tenancy**
   - Database-level access control
   - User authentication and authorization
   - Audit logging

---

## Conclusion

The ClickHouse Schema Flow Visualizer employs a clean, efficient architecture with clear separation of concerns:

- **Backend**: Go-based REST API with intelligent caching
- **Frontend**: Lightweight SPA with Mermaid.js rendering
- **Data Layer**: Dual-client support for flexible connectivity
- **Performance**: ~1000x improvement through strategic caching

The architecture prioritizes:
- **Performance** - Minimal database queries, fast response times
- **Simplicity** - No unnecessary abstractions or frameworks
- **Reliability** - Thread-safe operations, graceful error handling
- **Maintainability** - Clear code organization, comprehensive documentation

This design supports the tool's core mission: providing fast, intuitive visualization of ClickHouse database schemas for developers and database administrators.
