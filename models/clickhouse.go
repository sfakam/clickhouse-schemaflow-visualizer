package models

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/go-faster/city"
)

// Config holds the ClickHouse connection configuration
type Config struct {
	Host       string
	Port       int
	User       string
	Password   string
	Database   string
	Secure     bool
	SkipVerify bool
	ServerName string
	CertPath   string
	KeyPath    string
	CAPath     string
	UseHTTP    bool  // Use HTTP client instead of native client
}

var DatabasesData map[string]map[string]string
var TableRelations []TableRelation
var TableMetadata map[string]TableInfo

type TableRelation struct {
	DependsOnTable string
	Table          string
	Icon           string
}

type TableInfo struct {
	Name       string
	Database   string
	TotalRows  *uint64
	TotalBytes *uint64
	Engine     string
	Icon       string
}

type ColumnInfo struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Position uint64 `json:"position"`
	Comment  string `json:"comment"`
}

type TableDetails struct {
	Name        string       `json:"name"`
	Database    string       `json:"database"`
	Engine      string       `json:"engine"`
	TotalRows   *uint64      `json:"total_rows"`
	TotalBytes  *uint64      `json:"total_bytes"`
	Columns     []ColumnInfo `json:"columns"`
	CreateQuery string       `json:"create_query"`
}

// DatabaseStats represents statistics for a database
type DatabaseStats struct {
	Database     string                 `json:"database"`
	TotalTables  int                    `json:"total_tables"`
	TotalRows    uint64                 `json:"total_rows"`
	TotalBytes   uint64                 `json:"total_bytes"`
	EngineCounts map[string]EngineStats `json:"engine_counts"`
}

// EngineStats represents statistics for a specific engine type
type EngineStats struct {
	Count      int    `json:"count"`
	TotalRows  uint64 `json:"total_rows"`
	TotalBytes uint64 `json:"total_bytes"`
}

// CachedTableData represents comprehensive table information from a single query
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

// TableCache manages all table data with a single source of truth
type TableCache struct {
	Tables          map[string]*CachedTableData // key: database.table
	Relations       []TableRelation
	DatabasesMap    map[string]map[string]string
	LastRefresh     time.Time
	mutex           sync.RWMutex
}

// ClickHouseDBClient interface defines the database operations
type ClickHouseDBClient interface {
	Query(ctx context.Context, query string, args ...interface{}) (driver.Rows, error)
	QueryRow(ctx context.Context, query string, args ...interface{}) driver.Row
	Ping(ctx context.Context) error
	Close() error
}

// NativeClient wraps the native ClickHouse connection
type NativeClient struct {
	conn driver.Conn
}

func (n *NativeClient) Query(ctx context.Context, query string, args ...interface{}) (driver.Rows, error) {
	return n.conn.Query(ctx, query, args...)
}

func (n *NativeClient) QueryRow(ctx context.Context, query string, args ...interface{}) driver.Row {
	return n.conn.QueryRow(ctx, query, args...)
}

func (n *NativeClient) Ping(ctx context.Context) error {
	return n.conn.Ping(ctx)
}

func (n *NativeClient) Close() error {
	return n.conn.Close()
}

// HTTPRows implements driver.Rows for HTTP responses
type HTTPRows struct {
	lines    []string
	current  int
	columns  []string
	response *http.Response
}

func (r *HTTPRows) Next() bool {
	r.current++
	return r.current < len(r.lines)
}

func (r *HTTPRows) Scan(dest ...interface{}) error {
	if r.current >= len(r.lines) || r.current < 0 {
		return fmt.Errorf("no more rows")
	}
	
	line := r.lines[r.current]
	fields := strings.Split(line, "\t")
	
	for i, field := range fields {
		if i >= len(dest) {
			break
		}
		
		switch v := dest[i].(type) {
		case *string:
			*v = field
		case *int:
			if val, err := strconv.Atoi(field); err == nil {
				*v = val
			}
		case *uint64:
			if val, err := strconv.ParseUint(field, 10, 64); err == nil {
				*v = val
			}
		case **uint64:
			// Handle pointer to pointer to uint64 (nullable uint64)
			if field == "\\N" || field == "" || field == "0" {
				*v = nil
			} else {
				if val, err := strconv.ParseUint(field, 10, 64); err == nil {
					*v = &val
				}
			}
		case *[]string:
			// Handle array fields (enclosed in [])
			if strings.HasPrefix(field, "[") && strings.HasSuffix(field, "]") {
				field = strings.Trim(field, "[]")
				if field == "" {
					*v = []string{}
				} else {
					parts := strings.Split(field, ",")
					for j, part := range parts {
						parts[j] = strings.Trim(strings.Trim(part, "'"), "\"")
					}
					*v = parts
				}
			} else {
				*v = []string{field}
			}
		default:
			// Try to handle as string pointer
			if strPtr, ok := dest[i].(**string); ok {
				if field == "\\N" || field == "" {
					*strPtr = nil
				} else {
					str := field
					*strPtr = &str
				}
			} else {
				return fmt.Errorf("unsupported scan type: %T", dest[i])
			}
		}
	}
	
	return nil
}

func (r *HTTPRows) Close() error {
	if r.response != nil && r.response.Body != nil {
		return r.response.Body.Close()
	}
	return nil
}

func (r *HTTPRows) Err() error {
	return nil
}

func (r *HTTPRows) ColumnTypes() []driver.ColumnType {
	return []driver.ColumnType{}
}

func (r *HTTPRows) ScanStruct(dest interface{}) error {
	return fmt.Errorf("ScanStruct not implemented for HTTP client")
}

func (r *HTTPRows) Totals(dest ...interface{}) error {
	return fmt.Errorf("Totals not implemented for HTTP client")
}

func (r *HTTPRows) Columns() []string {
	return r.columns
}

// HTTPRow implements driver.Row for HTTP single row responses
type HTTPRow struct {
	data []string
	err  error
}

func (r *HTTPRow) Scan(dest ...interface{}) error {
	for i, field := range r.data {
		if i >= len(dest) {
			break
		}
		
		switch v := dest[i].(type) {
		case *string:
			*v = field
		case *int:
			if val, err := strconv.Atoi(field); err == nil {
				*v = val
			}
		case *uint64:
			if val, err := strconv.ParseUint(field, 10, 64); err == nil {
				*v = val
			}
		case **uint64:
			// Handle pointer to pointer to uint64 (nullable uint64)
			if field == "\\N" || field == "" || field == "0" {
				*v = nil
			} else {
				if val, err := strconv.ParseUint(field, 10, 64); err == nil {
					*v = &val
				}
			}
		case *[]string:
			// Handle array fields (enclosed in [])
			if strings.HasPrefix(field, "[") && strings.HasSuffix(field, "]") {
				field = strings.Trim(field, "[]")
				if field == "" {
					*v = []string{}
				} else {
					parts := strings.Split(field, ",")
					for j, part := range parts {
						parts[j] = strings.Trim(strings.Trim(part, "'"), "\"")
					}
					*v = parts
				}
			} else {
				*v = []string{field}
			}
		default:
			// Try to handle as string pointer
			if strPtr, ok := dest[i].(**string); ok {
				if field == "\\N" || field == "" {
					*strPtr = nil
				} else {
					str := field
					*strPtr = &str
				}
			} else {
				return fmt.Errorf("unsupported scan type: %T", dest[i])
			}
		}
	}
	return nil
}

func (r *HTTPRow) Err() error {
	return r.err
}

func (r *HTTPRow) ScanStruct(dest interface{}) error {
	return fmt.Errorf("ScanStruct not implemented for HTTP client")
}

// HTTPClient wraps HTTP-based ClickHouse connection using standard net/http
type HTTPClient struct {
	config Config
	client *http.Client
	baseURL string
}

func (h *HTTPClient) Query(ctx context.Context, query string, args ...interface{}) (driver.Rows, error) {
	// Replace any placeholders in query with args
	finalQuery := query
	for i, arg := range args {
		// Handle both ? and $N placeholders
		placeholder1 := "?"
		placeholder2 := fmt.Sprintf("$%d", i+1)
		
		// Escape string arguments
		var argStr string
		if str, ok := arg.(string); ok {
			argStr = fmt.Sprintf("'%s'", strings.ReplaceAll(str, "'", "''"))
		} else {
			argStr = fmt.Sprintf("%v", arg)
		}
		
		// Replace first occurrence of ? or $N
		if strings.Contains(finalQuery, placeholder1) {
			finalQuery = strings.Replace(finalQuery, placeholder1, argStr, 1)
		} else if strings.Contains(finalQuery, placeholder2) {
			finalQuery = strings.ReplaceAll(finalQuery, placeholder2, argStr)
		}
	}
	
	resp, err := h.executeQuery(ctx, finalQuery)
	if err != nil {
		return nil, err
	}
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		resp.Body.Close()
		return nil, fmt.Errorf("failed to read response: %v", err)
	}
	
	lines := strings.Split(strings.TrimSpace(string(body)), "\n")
	if len(lines) == 1 && lines[0] == "" {
		lines = []string{}
	}
	
	return &HTTPRows{
		lines:    lines,
		current:  -1,
		response: resp,
	}, nil
}

func (h *HTTPClient) QueryRow(ctx context.Context, query string, args ...interface{}) driver.Row {
	rows, err := h.Query(ctx, query, args...)
	if err != nil {
		return &HTTPRow{data: []string{}, err: err}
	}
	defer rows.Close()
	
	if rows.Next() {
		var data []string
		// Get the first row data - determine number of columns dynamically
		var col1, col2, col3, col4, col5, col6, col7, col8 string
		scanArgs := []interface{}{&col1, &col2, &col3, &col4, &col5, &col6, &col7, &col8}
		
		if err := rows.Scan(scanArgs...); err == nil {
			// Only include non-empty columns or up to the actual result
			data = []string{col1, col2, col3, col4, col5, col6, col7, col8}
			// Trim trailing empty strings
			for len(data) > 0 && data[len(data)-1] == "" {
				data = data[:len(data)-1]
			}
		} else {
			return &HTTPRow{data: []string{}, err: err}
		}
		return &HTTPRow{data: data}
	}
	
	return &HTTPRow{data: []string{}}
}

func (h *HTTPClient) Ping(ctx context.Context) error {
	_, err := h.executeQuery(ctx, "SELECT 1")
	return err
}

func (h *HTTPClient) Close() error {
	// HTTP client doesn't need explicit closing
	return nil
}

func (h *HTTPClient) executeQuery(ctx context.Context, query string) (*http.Response, error) {
	reqBody := strings.NewReader(query)
	
	req, err := http.NewRequestWithContext(ctx, "POST", h.baseURL, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}
	
	// Set basic auth
	auth := base64.StdEncoding.EncodeToString([]byte(h.config.User + ":" + h.config.Password))
	req.Header.Set("Authorization", "Basic "+auth)
	req.Header.Set("Content-Type", "text/plain")
	
	// Set database parameter
	if h.config.Database != "" {
		q := req.URL.Query()
		q.Set("database", h.config.Database)
		req.URL.RawQuery = q.Encode()
	}
	
	resp, err := h.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %v", err)
	}
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("query failed with status %d: %s", resp.StatusCode, string(body))
	}
	
	return resp, nil
}

// ClickHouseClient represents a client for interacting with ClickHouse
type ClickHouseClient struct {
	conn  ClickHouseDBClient
	cache *TableCache
}

// NewClickHouseClient creates a new ClickHouse client
func NewClickHouseClient(config Config) (*ClickHouseClient, error) {
	var client ClickHouseDBClient
	var err error

	if config.UseHTTP {
		client, err = createHTTPClient(config)
	} else {
		client, err = createNativeClient(config)
	}

	if err != nil {
		return nil, err
	}

	// Test connection
	if err := client.Ping(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to ping ClickHouse: %v", err)
	}

	return &ClickHouseClient{
		conn: client,
		cache: &TableCache{
			Tables:       make(map[string]*CachedTableData),
			Relations:    make([]TableRelation, 0),
			DatabasesMap: make(map[string]map[string]string),
		},
	}, nil
}

// createNativeClient creates a native TCP-based ClickHouse client
func createNativeClient(config Config) (*NativeClient, error) {
	options := &clickhouse.Options{
		Protocol: clickhouse.Native,
		Addr:     []string{fmt.Sprintf("%s:%d", config.Host, config.Port)},
		Auth: clickhouse.Auth{
			Database: config.Database,
			Username: config.User,
			Password: config.Password,
		},
	}

	// Configure TLS if enabled
	if config.Secure {
		tlsConfig := &tls.Config{
			InsecureSkipVerify: config.SkipVerify,
		}

		// Set server name if provided
		if config.ServerName != "" {
			tlsConfig.ServerName = config.ServerName
		}

		// Load client certificate if provided
		if config.CertPath != "" && config.KeyPath != "" {
			cert, err := tls.LoadX509KeyPair(config.CertPath, config.KeyPath)
			if err != nil {
				return nil, fmt.Errorf("failed to load client certificate: %v", err)
			}
			tlsConfig.Certificates = []tls.Certificate{cert}
		}

		// Load CA certificate if provided
		if config.CAPath != "" {
			caCert, err := os.ReadFile(config.CAPath)
			if err != nil {
				return nil, fmt.Errorf("failed to read CA certificate: %v", err)
			}
			caCertPool := x509.NewCertPool()
			if !caCertPool.AppendCertsFromPEM(caCert) {
				return nil, fmt.Errorf("failed to append CA certificate")
			}
			tlsConfig.RootCAs = caCertPool
		}

		options.TLS = tlsConfig
	}

	conn, err := clickhouse.Open(options)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to ClickHouse (native): %v", err)
	}

	return &NativeClient{conn: conn}, nil
}

// createHTTPClient creates an HTTP-based ClickHouse client using standard net/http
func createHTTPClient(config Config) (*HTTPClient, error) {
	// Create HTTP client with TLS configuration
	httpClient := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Configure TLS if enabled
	if config.Secure {
		tlsConfig := &tls.Config{
			InsecureSkipVerify: config.SkipVerify,
		}

		// Set server name if provided
		if config.ServerName != "" {
			tlsConfig.ServerName = config.ServerName
		}

		// Load client certificate if provided
		if config.CertPath != "" && config.KeyPath != "" {
			cert, err := tls.LoadX509KeyPair(config.CertPath, config.KeyPath)
			if err != nil {
				return nil, fmt.Errorf("failed to load client certificate: %v", err)
			}
			tlsConfig.Certificates = []tls.Certificate{cert}
		}

		// Load CA certificate if provided
		if config.CAPath != "" {
			caCert, err := os.ReadFile(config.CAPath)
			if err != nil {
				return nil, fmt.Errorf("failed to read CA certificate: %v", err)
			}
			caCertPool := x509.NewCertPool()
			if !caCertPool.AppendCertsFromPEM(caCert) {
				return nil, fmt.Errorf("failed to append CA certificate")
			}
			tlsConfig.RootCAs = caCertPool
		}

		httpClient.Transport = &http.Transport{
			TLSClientConfig: tlsConfig,
		}
	}

	// Build base URL
	protocol := "http"
	if config.Secure {
		protocol = "https"
	}
	baseURL := fmt.Sprintf("%s://%s:%d/", protocol, config.Host, config.Port)

	return &HTTPClient{
		config:  config,
		client:  httpClient,
		baseURL: baseURL,
	}, nil
}

type result struct {
	createQuery string
	engineFull  string
	engine      string
	totalRows   *uint64
	totalBytes  *uint64
}

func formatBytes(bytes *uint64) string {
	if bytes == nil {
		return "N/A"
	}
	const unit = 1024
	b := float64(*bytes)
	if b < unit {
		return fmt.Sprintf("%d B", *bytes)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", b/float64(div), "KMGTPE"[exp])
}

func formatRows(rows *uint64) string {
	if rows == nil {
		return "N/A"
	}
	if *rows < 1000 {
		return fmt.Sprintf("%d", *rows)
	}
	if *rows < 1000000 {
		return fmt.Sprintf("%.1fK", float64(*rows)/1000)
	}
	if *rows < 1000000000 {
		return fmt.Sprintf("%.1fM", float64(*rows)/1000000)
	}
	return fmt.Sprintf("%.1fB", float64(*rows)/1000000000)
}

// generateTableListContent creates the content for table display in the left sidebar
func generateTableListContent(icon, tableName string, totalRows *uint64, totalBytes *uint64) string {
	if totalRows == nil {
		return fmt.Sprintf(`%s %s`, icon, tableName)
	}

	return fmt.Sprintf(
		`%s %s<br><small style="color: #000; font-size: 0.8em;">Rows: <b>%s</b> | Size: <b>%s</b></small>`,
		icon, tableName, formatRows(totalRows), formatBytes(totalBytes),
	)
}

// refreshTableCache performs a single comprehensive query to populate all table data
func (c *ClickHouseClient) refreshTableCache() error {
	c.cache.mutex.Lock()
	defer c.cache.mutex.Unlock()

	// Check if cache is still fresh (refresh every 5 minutes)
	if time.Since(c.cache.LastRefresh) < 5*time.Minute && len(c.cache.Tables) > 0 {
		return nil
	}

	log.Println("Refreshing table cache with comprehensive query")
	ctx := context.Background()
	
	// Single query to get ALL table information
	query := `
		SELECT 
			database, 
			name, 
			engine, 
			engine_full, 
			create_table_query, 
			total_rows, 
			total_bytes, 
			loading_dependencies_database, 
			loading_dependencies_table
		FROM system.tables 
		WHERE database NOT IN ('system', 'information_schema', 'performance_schema', 'mysql')
		ORDER BY database, name
	`

	rows, err := c.conn.Query(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to query system.tables: %v", err)
	}
	defer rows.Close()

	// Clear existing cache
	c.cache.Tables = make(map[string]*CachedTableData)
	c.cache.Relations = make([]TableRelation, 0)
	c.cache.DatabasesMap = make(map[string]map[string]string)

	for rows.Next() {
		var database, name, engine, engineFull, createQuery string
		var totalRows, totalBytes *uint64
		var loadingDepsDB, loadingDepsTable []string

		if err := rows.Scan(&database, &name, &engine, &engineFull, &createQuery, 
			&totalRows, &totalBytes, &loadingDepsDB, &loadingDepsTable); err != nil {
			return fmt.Errorf("failed to scan table data: %v", err)
		}

		// Skip unwanted databases
		if !allowedDatabase(database) {
			continue
		}

		fullTableName := database + "." + name
		icon := c.getEngineIcon(engine)

		// Store in cache
		c.cache.Tables[fullTableName] = &CachedTableData{
			Name:                        name,
			Database:                    database,
			Engine:                      engine,
			EngineFullMeta:             engineFull,
			CreateQuery:                createQuery,
			TotalRows:                  totalRows,
			TotalBytes:                 totalBytes,
			LoadingDependenciesDatabase: loadingDepsDB,
			LoadingDependenciesTable:   loadingDepsTable,
			Icon:                       icon,
			LastUpdated:                time.Now(),
		}

		// Build databases map for UI
		if c.cache.DatabasesMap[database] == nil {
			c.cache.DatabasesMap[database] = make(map[string]string)
		}
		c.cache.DatabasesMap[database][name] = generateTableListContent(icon, name, totalRows, totalBytes)

		// Generate relations from the create query
		relations := c.extractRelationsFromQuery(database, name, engine, engineFull, createQuery, loadingDepsDB, loadingDepsTable)
		c.cache.Relations = append(c.cache.Relations, relations...)
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("error iterating table rows: %v", err)
	}

	c.cache.LastRefresh = time.Now()
	log.Printf("Cache refreshed with %d tables from %d databases", len(c.cache.Tables), len(c.cache.DatabasesMap))
	return nil
}

// getEngineIcon returns the appropriate icon for a table engine
func (c *ClickHouseClient) getEngineIcon(engine string) string {
	switch {
	case engine == "MergeTree":
		return `<i class="fa-solid fa-database"></i>`
	case strings.HasPrefix(engine, "Replicated"):
		return `<i class="fa-solid fa-circle-nodes"></i>`
	case strings.HasPrefix(engine, "Dictionary"):
		return `<i class="fa-solid fa-book"></i>`
	case engine == "Distributed":
		return `<i class="fa-solid fa-diagram-project"></i>`
	case engine == "MaterializedView":
		return `<i class="fa-solid fa-eye"></i>`
	default:
		return `<i class="fa-solid fa-table"></i>`
	}
}

// extractRelationsFromQuery extracts table relationships from create queries
func (c *ClickHouseClient) extractRelationsFromQuery(database, table, engine, engineFull, createQuery string, loadingDepsDB, loadingDepsTable []string) []TableRelation {
	var relations []TableRelation
	fullTableName := database + "." + table
	icon := c.getEngineIcon(engine)

	switch engine {
	case "MergeTree":
		relations = append(relations, TableRelation{Table: fullTableName, Icon: icon})

	case "Distributed":
		// Extract target table from engine_full
		if parts := strings.Split(engineFull, "'"); len(parts) >= 6 {
			targetDB := parts[3]
			targetTable := parts[5]
			targetFullName := targetDB + "." + targetTable
			relations = append(relations, TableRelation{
				DependsOnTable: fullTableName,
				Table:          targetFullName,
				Icon:           icon,
			})
		} else {
			relations = append(relations, TableRelation{Table: fullTableName, Icon: icon})
		}

	case "MaterializedView":
		// Extract source and target from CREATE query
		queryParts1 := strings.Split(createQuery, " ")
		queryParts2 := strings.Split(createQuery, "FROM ")
		if len(queryParts1) > 5 && len(queryParts2) > 1 {
			mvTable := queryParts1[3]
			targetTable := queryParts1[5]
			queryParts3 := strings.Split(queryParts2[1], " ")
			sourceTable := queryParts3[0]

			relations = append(relations, TableRelation{
				DependsOnTable: sourceTable,
				Table:          mvTable,
				Icon:           icon,
			})
			relations = append(relations, TableRelation{
				DependsOnTable: mvTable,
				Table:          targetTable,
				Icon:           icon,
			})
		}

	default:
		// Handle dictionary dependencies
		if len(loadingDepsDB) > 0 && len(loadingDepsTable) > 0 {
			depTable := loadingDepsDB[0] + "." + loadingDepsTable[0]
			relations = append(relations, TableRelation{
				DependsOnTable: depTable,
				Table:          fullTableName,
				Icon:           icon,
			})
		} else {
			relations = append(relations, TableRelation{Table: fullTableName, Icon: icon})
		}
	}

	return relations
}

// getTableFromCache retrieves table data from cache, refreshing if necessary
func (c *ClickHouseClient) getTableFromCache(database, table string) (*CachedTableData, error) {
	if err := c.refreshTableCache(); err != nil {
		return nil, err
	}

	c.cache.mutex.RLock()
	defer c.cache.mutex.RUnlock()

	fullName := database + "." + table
	tableData, exists := c.cache.Tables[fullName]
	if !exists {
		return nil, fmt.Errorf("table %s.%s not found", database, table)
	}

	return tableData, nil
}

func (c *ClickHouseClient) getTablesRelations() ([]TableRelation, error) {
	if TableRelations != nil && DatabasesData != nil && TableMetadata != nil {
		log.Println("Using cached tables relations")
		return TableRelations, nil
	}

	log.Println("Querying tables relations")
	ctx := context.Background()
	query := fmt.Sprintf("SELECT create_table_query, engine_full, engine, database, name, loading_dependencies_database, loading_dependencies_table, total_rows, total_bytes FROM system.tables ORDER BY name")
	rows, err := c.conn.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query tables: %v", err)
	}
	defer rows.Close()

	var tables []TableRelation
	if TableMetadata == nil {
		TableMetadata = make(map[string]TableInfo)
	}

	for rows.Next() {
		res := result{}
		database, table := "", ""
		var loadingDependenciesTable []string
		var loadingDependenciesDatabase []string
		if err := rows.Scan(&res.createQuery, &res.engineFull, &res.engine, &database, &table, &loadingDependenciesDatabase, &loadingDependenciesTable, &res.totalRows, &res.totalBytes); err != nil {
			return nil, fmt.Errorf("failed to scan table data: %v", err)
		}

		if !allowedDatabase(database) {
			continue
		}

		if DatabasesData == nil {
			DatabasesData = make(map[string]map[string]string)
		}

		if DatabasesData[database] == nil {
			DatabasesData[database] = make(map[string]string)
		}

		fullTableName := database + "." + table
		var icon string

		// Extract the relation from the creation query
		if res.engine == "MergeTree" { // Local Table
			queryParts := strings.Split(res.createQuery, " ")
			icon = `<i class="fa-solid fa-database"></i>`
			if len(queryParts) > 2 {
				tableName := queryParts[2]
				DatabasesData[database][table] = generateTableListContent(icon, table, res.totalRows, res.totalBytes)

				tables = append(tables, TableRelation{Table: tableName, Icon: icon})
			}
		} else if strings.HasPrefix(res.engine, "Replicated") { // Replicated Table
			queryParts := strings.Split(res.createQuery, " ")
			icon = `<i class="fa-solid fa-circle-nodes"></i>`
			DatabasesData[database][table] = generateTableListContent(icon, table, res.totalRows, res.totalBytes)
			if len(queryParts) > 2 {
				tableName := queryParts[2]

				tables = append(tables, TableRelation{Table: tableName, Icon: icon})
			}
		} else if strings.HasPrefix(res.engine, "Dictionary") { // Dictionary Table
			queryParts := strings.Split(res.createQuery, " ")
			icon = `<i class="fa-solid fa-book"></i>`
			DatabasesData[database][table] = generateTableListContent(icon, table, res.totalRows, res.totalBytes)
			if len(queryParts) > 2 {
				tableName := queryParts[2]

				if len(loadingDependenciesDatabase) > 0 && len(loadingDependenciesTable) > 0 {
					tables = append(tables, TableRelation{DependsOnTable: loadingDependenciesDatabase[0] + "." + loadingDependenciesTable[0], Table: tableName, Icon: icon})
				} else {
					tables = append(tables, TableRelation{Table: tableName, Icon: icon})
				}
			}
		} else if res.engine == "Distributed" { // Distributed Table
			queryParts := strings.Split(res.createQuery, " ")
			queryParts2 := strings.Split(res.engineFull, "'")
			icon = `<i class="fa-solid fa-diagram-project"></i>`
			DatabasesData[database][table] = generateTableListContent(icon, table, res.totalRows, res.totalBytes)
			if len(queryParts) > 2 {
				tableName := queryParts[2]
				if len(queryParts2) >= 6 {
					dstTable := queryParts2[3] + "." + queryParts2[5]
					tables = append(tables, TableRelation{DependsOnTable: tableName, Table: dstTable, Icon: icon})
				} else {
					tables = append(tables, TableRelation{Table: tableName, Icon: icon})
				}
			}
		} else if res.engine == "MaterializedView" { // Materialized View
			queryParts1 := strings.Split(res.createQuery, " ")
			queryParts2 := strings.Split(res.createQuery, "FROM ")
			icon = `<i class="fa-solid fa-eye"></i>`
			DatabasesData[database][table] = generateTableListContent(icon, table, res.totalRows, res.totalBytes)
			if len(queryParts1) > 3 && len(queryParts2) > 1 {
				mvTable := queryParts1[3]
				dstTable := queryParts1[5]
				queryParts3 := strings.Split(queryParts2[1], " ")
				srcTable := queryParts3[0]

				tables = append(tables, TableRelation{DependsOnTable: srcTable, Table: mvTable, Icon: icon})
				tables = append(tables, TableRelation{DependsOnTable: mvTable, Table: dstTable, Icon: icon})
			}
		} else {
			// Default case for other engines
			icon = `<i class="fa-solid fa-table"></i>`
			DatabasesData[database][table] = generateTableListContent(icon, table, res.totalRows, res.totalBytes)
			tables = append(tables, TableRelation{Table: table, Icon: icon})
		}

		// Store table metadata
		TableMetadata[fullTableName] = TableInfo{
			Name:       table,
			Database:   database,
			TotalRows:  res.totalRows,
			TotalBytes: res.totalBytes,
			Engine:     res.engine,
			Icon:       icon,
		}
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating table rows: %v", err)
	}

	TableRelations = tables

	return tables, nil
}

func allowedDatabase(database string) bool {
	switch {
	case database == "":
		return false
	case database == "system":
		return false
	case strings.ToLower(database) == "information_schema":
		return false
	case database == "performance_schema":
		return false
	case database == "mysql":
		return false
	default:
		return true
	}
}

// GetDatabases returns a list of all databases
func (c *ClickHouseClient) GetDatabases() (map[string]map[string]string, error) {
	if err := c.refreshTableCache(); err != nil {
		return nil, fmt.Errorf("failed to refresh table cache: %v", err)
	}

	c.cache.mutex.RLock()
	defer c.cache.mutex.RUnlock()

	return c.cache.DatabasesMap, nil
}

// GetDatabaseStats returns statistics for a specific database
func (c *ClickHouseClient) GetDatabaseStats(dbName string) (*DatabaseStats, error) {
	if err := c.refreshTableCache(); err != nil {
		return nil, fmt.Errorf("failed to refresh table cache: %v", err)
	}

	c.cache.mutex.RLock()
	defer c.cache.mutex.RUnlock()

	stats := &DatabaseStats{
		Database:     dbName,
		TotalTables:  0,
		TotalRows:    0,
		TotalBytes:   0,
		EngineCounts: make(map[string]EngineStats),
	}

	// Iterate through all tables in the cache for this database
	for _, tableData := range c.cache.Tables {
		if tableData.Database != dbName {
			continue
		}

		stats.TotalTables++

		// Add rows and bytes to totals (if available)
		if tableData.TotalRows != nil {
			stats.TotalRows += *tableData.TotalRows
		}
		if tableData.TotalBytes != nil {
			stats.TotalBytes += *tableData.TotalBytes
		}

		// Track engine statistics
		engineStats := stats.EngineCounts[tableData.Engine]
		engineStats.Count++
		if tableData.TotalRows != nil {
			engineStats.TotalRows += *tableData.TotalRows
		}
		if tableData.TotalBytes != nil {
			engineStats.TotalBytes += *tableData.TotalBytes
		}
		stats.EngineCounts[tableData.Engine] = engineStats
	}

	return stats, nil
}

// GenerateMermaidSchema generates a Mermaid schema for a table and its relationships
func (c *ClickHouseClient) GenerateMermaidSchema(dbName, tableName string) (string, error) {
	if err := c.refreshTableCache(); err != nil {
		return "", fmt.Errorf("failed to refresh table cache: %v", err)
	}

	table := dbName + "." + tableName

	// Start building the Mermaid schema
	var sb strings.Builder
	sb.WriteString("flowchart TB\n")

	c.cache.mutex.RLock()
	relations := c.cache.Relations
	c.cache.mutex.RUnlock()

	// Generate node for the main table with additional info
	nodeContent := c.generateTableNodeContent(table)
	sb.WriteString(fmt.Sprintf("    %d[\"%s\"]\n\n", city.Hash32([]byte(table)), nodeContent))
	sb.WriteString(fmt.Sprintf("    style %d fill:#FF6D00,stroke:#AA00FF,color:#FFFFFF\n\n", city.Hash32([]byte(table))))

	seen := make(map[string]bool)
	c.getRelationsNext(&sb, relations, table, &seen)
	c.getRelationsBack(&sb, relations, table, &seen)

	return sb.String(), nil
}

func (c *ClickHouseClient) generateTableNodeContent(table string) string {
	c.cache.mutex.RLock()
	defer c.cache.mutex.RUnlock()
	
	if tableData, exists := c.cache.Tables[table]; exists && tableData.TotalRows != nil {
		return fmt.Sprintf(
			"%s<br><small>Rows: <b>%s</b> Size: <b>%s</b></small>",
			table,
			formatRows(tableData.TotalRows),
			formatBytes(tableData.TotalBytes),
		)
	}
	return table
}

func (c *ClickHouseClient) getRelationsNext(sb *strings.Builder, tablesRelations []TableRelation, table string, seen *map[string]bool) {
	for _, rel := range tablesRelations {
		if rel.DependsOnTable == table && table != "" {
			depContent := c.generateTableNodeContent(rel.DependsOnTable)
			relContent := c.generateTableNodeContent(rel.Table)

			mermaidRow := fmt.Sprintf(
				"    %d[\"%s\"] --> %d[\"%s\"]\n",
				city.Hash32([]byte(rel.DependsOnTable)), depContent,
				city.Hash32([]byte(rel.Table)), relContent,
			)

			if !(*seen)[mermaidRow] {
				(*seen)[mermaidRow] = true
				sb.WriteString(mermaidRow)
			}
			c.getRelationsNext(sb, tablesRelations, rel.Table, seen)
		}
	}
}

func (c *ClickHouseClient) getRelationsBack(sb *strings.Builder, tablesRelations []TableRelation, table string, seen *map[string]bool) {
	for _, rel := range tablesRelations {
		if rel.Table == table && rel.DependsOnTable != "" {
			depContent := c.generateTableNodeContent(rel.DependsOnTable)
			relContent := c.generateTableNodeContent(rel.Table)

			mermaidRow := fmt.Sprintf(
				"    %d[\"%s\"] --> %d[\"%s\"]\n",
				city.Hash32([]byte(rel.DependsOnTable)), depContent,
				city.Hash32([]byte(rel.Table)), relContent,
			)

			if !(*seen)[mermaidRow] {
				(*seen)[mermaidRow] = true
				sb.WriteString(mermaidRow)
			}
			c.getRelationsBack(sb, tablesRelations, rel.DependsOnTable, seen)
		}
	}
}

// GenerateDatabaseMermaidSchema generates a comprehensive Mermaid schema for an entire database
func (c *ClickHouseClient) GenerateDatabaseMermaidSchema(dbName string, engineFilters []string, includeMetadata bool) (string, error) {
	// Get all tables in the database
	databases, err := c.GetDatabases()
	if err != nil {
		return "", fmt.Errorf("failed to get databases: %v", err)
	}

	// Check if database exists
	tablesMap, exists := databases[dbName]
	if !exists {
		return "", fmt.Errorf("database '%s' not found", dbName)
	}

	// Get table relations
	tablesRelations, err := c.getTablesRelations()
	if err != nil {
		return "", fmt.Errorf("failed to get table relations: %v", err)
	}

	// Start building the Mermaid schema
	var sb strings.Builder
	sb.WriteString("flowchart LR\n")
	sb.WriteString("    %% Database: " + dbName + "\n\n")

	// Create a map to store engine types and their styles
	engineStyles := map[string]string{
		"MergeTree":                "#1f77b4", // Blue
		"ReplicatedMergeTree":      "#ff7f0e", // Orange  
		"SummingMergeTree":         "#2ca02c", // Green
		"ReplicatedSummingMergeTree": "#2ca02c", // Green (same as SummingMergeTree)
		"ReplacingMergeTree":       "#d62728", // Red
		"ReplicatedReplacingMergeTree": "#d62728", // Red (same as ReplacingMergeTree)
		"AggregatingMergeTree":     "#9467bd", // Purple
		"ReplicatedAggregatingMergeTree": "#9467bd", // Purple (same as AggregatingMergeTree)
		"CollapsingMergeTree":      "#8c564b", // Brown
		"ReplicatedCollapsingMergeTree": "#8c564b", // Brown (same as CollapsingMergeTree)
		"VersionedCollapsingMergeTree": "#e377c2", // Pink
		"ReplicatedVersionedCollapsingMergeTree": "#e377c2", // Pink (same as VersionedCollapsingMergeTree)
		"GraphiteMergeTree":        "#7f7f7f", // Gray
		"ReplicatedGraphiteMergeTree": "#7f7f7f", // Gray (same as GraphiteMergeTree)
		"MaterializedView":         "#bcbd22", // Olive
		"View":                     "#17becf", // Cyan
		"Dictionary":               "#ffbb78", // Light Orange
		"Distributed":              "#ff9896", // Light Red
		"Memory":                   "#c5b0d5", // Light Purple
		"Log":                      "#c7c7c7", // Light Gray
		"TinyLog":                  "#dbdb8d", // Light Olive
		"StripeLog":                "#9edae5", // Light Cyan
	}

	engineIcons := map[string]string{
		"MergeTree":                "fa-solid fa-database",
		"ReplicatedMergeTree":      "fa-solid fa-copy",
		"SummingMergeTree":         "fa-solid fa-calculator",
		"ReplicatedSummingMergeTree": "fa-solid fa-calculator",
		"ReplacingMergeTree":       "fa-solid fa-sync-alt",
		"ReplicatedReplacingMergeTree": "fa-solid fa-sync-alt",
		"AggregatingMergeTree":     "fa-solid fa-chart-bar",
		"ReplicatedAggregatingMergeTree": "fa-solid fa-chart-bar",
		"CollapsingMergeTree":      "fa-solid fa-compress",
		"ReplicatedCollapsingMergeTree": "fa-solid fa-compress",
		"VersionedCollapsingMergeTree": "fa-solid fa-code-branch",
		"ReplicatedVersionedCollapsingMergeTree": "fa-solid fa-code-branch",
		"GraphiteMergeTree":        "fa-solid fa-chart-line",
		"ReplicatedGraphiteMergeTree": "fa-solid fa-chart-line",
		"MaterializedView":         "fa-solid fa-eye",
		"View":                     "fa-solid fa-search",
		"Dictionary":               "fa-solid fa-book",
		"Distributed":              "fa-solid fa-share-alt",
		"Memory":                   "fa-solid fa-memory",
		"Log":                      "fa-solid fa-file-text",
		"TinyLog":                  "fa-solid fa-file",
		"StripeLog":                "fa-solid fa-stream",
	}

	// Track processed tables and their engine types
	processedTables := make(map[string]bool)
	engineCounts := make(map[string]int)

	// Get engine information for all tables in the database
	tableEngines, err := c.getTableEngines(dbName)
	if err != nil {
		return "", fmt.Errorf("failed to get table engines: %v", err)
	}

	// Collect all unique table names from relations data for this database
	// Maps clean table name to preferred relation table name (prefer escaped format for consistency with relations)
	uniqueTables := make(map[string]string)
	
	for _, relation := range tablesRelations {
		// Check both Table and DependsOnTable fields
		for _, relationTableName := range []string{relation.Table, relation.DependsOnTable} {
			if relationTableName != "" {
				// Check if this table belongs to our target database
				if strings.HasPrefix(relationTableName, dbName+".") || strings.HasPrefix(relationTableName, dbName+"\\.") {
					// Extract clean table name for engine lookup
					cleanTableName := relationTableName
					
					// Remove database prefix
					if strings.HasPrefix(cleanTableName, dbName+"\\.") {
						// Handle escaped format: "owl\.table_name\"
						cleanTableName = strings.TrimPrefix(cleanTableName, dbName+"\\.")
						cleanTableName = strings.ReplaceAll(cleanTableName, "\\", "")
					} else {
						// Handle normal format: "owl.table_name"
						cleanTableName = strings.TrimPrefix(cleanTableName, dbName+".")
					}
					
					// Remove any remaining backticks
					cleanTableName = strings.ReplaceAll(cleanTableName, "`", "")
					
					// Only include if the clean table exists in system.tables
					if _, exists := tablesMap[cleanTableName]; exists {
						// Prefer escaped format for consistency with relations data
						if existingRelationName, exists := uniqueTables[cleanTableName]; !exists {
							uniqueTables[cleanTableName] = relationTableName
						} else {
							// If escaped format is available, use it over clean format
							if strings.Contains(relationTableName, "\\.") && !strings.Contains(existingRelationName, "\\.") {
								uniqueTables[cleanTableName] = relationTableName
							}
						}
					}
				}
			}
		}
	}

	// Process each unique table from relations data that also exists in system.tables  
	for cleanTableName, relationTableName := range uniqueTables {
		// Get engine type from the database query
		engineType, exists := tableEngines[cleanTableName]
		if !exists {
			engineType = "Unknown"
		}
		
		// Apply engine filter if specified
		if len(engineFilters) > 0 && !c.containsEngine(engineFilters, engineType) {
			continue
		}

		engineCounts[engineType]++

		// Generate node content
		var nodeContent string
		if includeMetadata {
			nodeContent = c.generateTableNodeContent(relationTableName)
		} else {
			nodeContent = cleanTableName
		}

		// Use the full table name without line breaks - let CSS handle the box sizing
		displayName := cleanTableName

		// Simplified node content for better Mermaid rendering
		nodeContent = fmt.Sprintf("%s (%s)", displayName, engineType)

		// Create node using the relation table name for consistent hashing
		nodeId := city.Hash32([]byte(relationTableName))
		sb.WriteString(fmt.Sprintf("    %d[\"%s\"]\n", nodeId, nodeContent))

		// Apply styling based on engine type
		if color, exists := engineStyles[engineType]; exists {
			sb.WriteString(fmt.Sprintf("    style %d fill:%s,stroke:#333,stroke-width:2px,color:#fff\n", nodeId, color))
		}

		processedTables[relationTableName] = true
	}

	sb.WriteString("\n    %% Relationships\n")

	// Track seen relationships to avoid duplicates
	seenRelationships := make(map[string]bool)

	// Add relationships between tables in this database - explore all relationships recursively
	for relationTableName := range processedTables {
		// Get all relationships for this table (both forward and backward)
		c.getDatabaseRelationsNext(&sb, tablesRelations, relationTableName, &seenRelationships, processedTables)
		c.getDatabaseRelationsBack(&sb, tablesRelations, relationTableName, &seenRelationships, processedTables)
	}



	// Add legend for engine types
	if len(engineCounts) > 0 {
		sb.WriteString("\n    %% Legend\n")
		legendId := 999999
		for engineType, count := range engineCounts {
			if icon, exists := engineIcons[engineType]; exists {
				legendContent := fmt.Sprintf("<i class=\"%s\"></i> %s (%d)", icon, engineType, count)
				sb.WriteString(fmt.Sprintf("    %d[\"%s\"]\n", legendId, legendContent))
				if color, exists := engineStyles[engineType]; exists {
					sb.WriteString(fmt.Sprintf("    style %d fill:%s,stroke:#333,stroke-width:1px,color:#fff\n", legendId, color))
				}
				legendId++
			}
		}
	}

	// Log the complete Mermaid diagram for external validation
	mermaidSchema := sb.String()
	fmt.Printf("\n=== COMPLETE MERMAID DIAGRAM FOR EXTERNAL VALIDATION ===\n")
	fmt.Printf("%s\n", mermaidSchema)
	fmt.Printf("=== END MERMAID DIAGRAM ===\n\n")

	return mermaidSchema, nil
}

// getDatabaseRelationsNext recursively finds forward relationships for database view
func (c *ClickHouseClient) getDatabaseRelationsNext(sb *strings.Builder, tablesRelations []TableRelation, table string, seen *map[string]bool, processedTables map[string]bool) {
	for _, rel := range tablesRelations {
		if rel.DependsOnTable == table && table != "" {
			// Only include if both tables are in processed list (pass engine filters)
			if processedTables[rel.DependsOnTable] && processedTables[rel.Table] {
				sourceId := city.Hash32([]byte(rel.DependsOnTable))
				targetId := city.Hash32([]byte(rel.Table))
				
				relationshipKey := fmt.Sprintf("%d-->%d", sourceId, targetId)
				if !(*seen)[relationshipKey] {
					(*seen)[relationshipKey] = true
					relationshipType := c.getRelationshipType(rel)
					sb.WriteString(fmt.Sprintf("    %d %s %d\n", sourceId, relationshipType, targetId))
				}
				// Recursively explore further
				c.getDatabaseRelationsNext(sb, tablesRelations, rel.Table, seen, processedTables)
			}
		}
	}
}

// getDatabaseRelationsBack recursively finds backward relationships for database view
func (c *ClickHouseClient) getDatabaseRelationsBack(sb *strings.Builder, tablesRelations []TableRelation, table string, seen *map[string]bool, processedTables map[string]bool) {
	for _, rel := range tablesRelations {
		if rel.Table == table && rel.DependsOnTable != "" {
			// Only include if both tables are in processed list (pass engine filters)
			if processedTables[rel.DependsOnTable] && processedTables[rel.Table] {
				sourceId := city.Hash32([]byte(rel.DependsOnTable))
				targetId := city.Hash32([]byte(rel.Table))
				
				relationshipKey := fmt.Sprintf("%d-->%d", sourceId, targetId)
				if !(*seen)[relationshipKey] {
					(*seen)[relationshipKey] = true
					relationshipType := c.getRelationshipType(rel)
					sb.WriteString(fmt.Sprintf("    %d %s %d\n", sourceId, relationshipType, targetId))
				}
				// Recursively explore further
				c.getDatabaseRelationsBack(sb, tablesRelations, rel.DependsOnTable, seen, processedTables)
			}
		}
	}
}

// Helper function to extract engine type from HTML string
func (c *ClickHouseClient) extractEngineFromHTML(htmlString string) string {
	// This is a simplified extraction - you might need to adjust based on your HTML format
	// Looking for patterns like "ReplicatedMergeTree", "MergeTree", etc.
	engines := []string{
		"ReplicatedMergeTree", "MergeTree", "SummingMergeTree", "ReplacingMergeTree",
		"AggregatingMergeTree", "CollapsingMergeTree", "VersionedCollapsingMergeTree",
		"GraphiteMergeTree", "MaterializedView", "View", "Dictionary", "Distributed",
		"Memory", "TinyLog", "StripeLog", "Log",
	}
	
	htmlLower := strings.ToLower(htmlString)
	for _, engine := range engines {
		if strings.Contains(htmlLower, strings.ToLower(engine)) {
			return engine
		}
	}
	return "Unknown"
}

// Helper function to check if engine is in filter list
func (c *ClickHouseClient) containsEngine(filters []string, engineType string) bool {
	if len(filters) == 0 {
		return true
	}
	for _, filter := range filters {
		if strings.EqualFold(filter, engineType) {
			return true
		}
	}
	return false
}

// getTableEngines returns a map of table names to their engine types for a specific database
func (c *ClickHouseClient) getTableEngines(dbName string) (map[string]string, error) {
	if err := c.refreshTableCache(); err != nil {
		return nil, fmt.Errorf("failed to refresh cache: %v", err)
	}

	c.cache.mutex.RLock()
	defer c.cache.mutex.RUnlock()

	tableEngines := make(map[string]string)
	for _, tableData := range c.cache.Tables {
		if tableData.Database == dbName {
			tableEngines[tableData.Name] = tableData.Engine
		}
	}

	return tableEngines, nil
}

// formatLongTableName breaks long table names into multiple lines for better display
func (c *ClickHouseClient) formatLongTableName(tableName string) string {
	if len(tableName) <= 25 {
		return tableName
	}

	// Try to break at natural separators like underscores, dots, or camelCase
	words := strings.FieldsFunc(tableName, func(r rune) bool {
		return r == '_' || r == '.' || r == '-'
	})

	if len(words) <= 1 {
		// No natural separators, break every 25 characters
		var result strings.Builder
		for i, char := range tableName {
			if i > 0 && i%25 == 0 {
				result.WriteString("<br>")
			}
			result.WriteRune(char)
		}
		return result.String()
	}

	// Reconstruct with line breaks, trying to keep lines under 25 chars
	var result strings.Builder
	currentLine := ""
	separator := "_" // Use the most common separator found

	for _, word := range words {
		testLine := word
		if currentLine != "" {
			testLine = currentLine + separator + word
		}

		if len(testLine) > 25 && currentLine != "" {
			// Start new line
			if result.Len() > 0 {
				result.WriteString("<br>")
			}
			result.WriteString(currentLine)
			currentLine = word
		} else {
			currentLine = testLine
		}
	}

	// Add the last line
	if currentLine != "" {
		if result.Len() > 0 {
			result.WriteString("<br>")
		}
		result.WriteString(currentLine)
	}

	return result.String()
}

// Helper function to determine relationship type for Mermaid arrows
func (c *ClickHouseClient) getRelationshipType(relation TableRelation) string {
	// Different arrow types based on relationship
	switch {
	case strings.Contains(strings.ToLower(relation.Icon), "materialized"):
		return "-.->|materialized|"
	case strings.Contains(strings.ToLower(relation.Icon), "distributed"):
		return "==>|distributed|"
	case strings.Contains(strings.ToLower(relation.Icon), "replicated"):
		return "-->|replicated|"
	case strings.Contains(strings.ToLower(relation.Icon), "dictionary"):
		return "..->|dictionary|"
	default:
		return "-->|depends|"
	}
}

// GetTableColumns returns detailed column information for a specific table
func (c *ClickHouseClient) GetTableColumns(database, table string) (*TableDetails, error) {
	// Get basic table info from cache (much faster than querying system.tables)
	tableData, err := c.getTableFromCache(database, table)
	if err != nil {
		return nil, fmt.Errorf("failed to get table from cache: %v", err)
	}

	// Only query columns separately (this is the minimal query needed)
	ctx := context.Background()
	columnsQuery := `
		SELECT name, type, position, comment
		FROM system.columns 
		WHERE database = ? AND table = ? 
		ORDER BY position
	`

	rows, err := c.conn.Query(ctx, columnsQuery, database, table)
	if err != nil {
		return nil, fmt.Errorf("failed to query columns: %v", err)
	}
	defer rows.Close()

	var columns []ColumnInfo
	for rows.Next() {
		var col ColumnInfo
		if err := rows.Scan(&col.Name, &col.Type, &col.Position, &col.Comment); err != nil {
			return nil, fmt.Errorf("failed to scan column: %v", err)
		}
		columns = append(columns, col)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating columns: %v", err)
	}

	return &TableDetails{
		Name:        tableData.Name,
		Database:    tableData.Database,
		Engine:      tableData.Engine,
		TotalRows:   tableData.TotalRows,
		TotalBytes:  tableData.TotalBytes,
		Columns:     columns,
		CreateQuery: c.formatCreateQuery(tableData.CreateQuery),
	}, nil
}

// formatCreateQuery formats a CREATE TABLE query for better readability
func (c *ClickHouseClient) formatCreateQuery(query string) string {
	if query == "" {
		return ""
	}

	// Basic formatting - add line breaks at key SQL keywords
	formatted := query
	
	// Add line breaks after key keywords
	keywords := []string{
		"CREATE TABLE", "CREATE MATERIALIZED VIEW", "CREATE VIEW", "CREATE DICTIONARY",
		") ENGINE =", ") AS SELECT", "ORDER BY", "PARTITION BY", "SAMPLE BY", 
		"SETTINGS", "TTL", "PRIMARY KEY", "REFRESH EVERY",
	}
	
	for _, keyword := range keywords {
		formatted = strings.ReplaceAll(formatted, keyword, "\n"+keyword)
	}
	
	// Add line breaks after column definitions (look for comma followed by backtick)
	formatted = regexp.MustCompile(`,(\s*\x60)`).ReplaceAllString(formatted, ",\n    $1")
	
	// Add proper indentation for column definitions
	lines := strings.Split(formatted, "\n")
	var result []string
	inColumnDefinition := false
	
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		
		// Detect if we're in column definition section
		if strings.Contains(trimmed, "CREATE") && (strings.Contains(trimmed, "TABLE") || strings.Contains(trimmed, "VIEW")) {
			result = append(result, trimmed)
			inColumnDefinition = false
		} else if trimmed == "(" || (inColumnDefinition && strings.HasPrefix(trimmed, "(")) {
			result = append(result, trimmed)
			inColumnDefinition = true
		} else if strings.HasPrefix(trimmed, ") ENGINE") || strings.HasPrefix(trimmed, ") AS") {
			result = append(result, trimmed)
			inColumnDefinition = false
		} else if inColumnDefinition && (strings.HasPrefix(trimmed, "`") || strings.Contains(trimmed, "`")) {
			// Column definition - add indentation
			result = append(result, "    "+trimmed)
		} else if strings.HasPrefix(trimmed, "ORDER BY") || strings.HasPrefix(trimmed, "PARTITION BY") || 
				 strings.HasPrefix(trimmed, "SETTINGS") || strings.HasPrefix(trimmed, "TTL") ||
				 strings.HasPrefix(trimmed, "PRIMARY KEY") || strings.HasPrefix(trimmed, "REFRESH EVERY") {
			result = append(result, trimmed)
		} else if trimmed != "" {
			// Check if previous line needs continuation
			if i > 0 && (strings.HasSuffix(strings.TrimSpace(lines[i-1]), ",") || 
						strings.Contains(trimmed, "SELECT") || strings.Contains(trimmed, "FROM") ||
						strings.Contains(trimmed, "WHERE") || strings.Contains(trimmed, "GROUP BY")) {
				result = append(result, "    "+trimmed)
			} else {
				result = append(result, trimmed)
			}
		}
	}
	
	return strings.Join(result, "\n")
}

// Close closes the ClickHouse connection
func (c *ClickHouseClient) Close() error {
	return c.conn.Close()
}
