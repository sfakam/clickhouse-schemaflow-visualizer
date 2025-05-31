package models

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/go-faster/city"
)

// Config holds the ClickHouse connection configuration
type Config struct {
	Host     string
	Port     int
	User     string
	Password string
	Database string
	// TLS configuration
	Secure     bool   // Enable TLS
	SkipVerify bool   // Skip TLS certificate verification
	CertPath   string // Path to client certificate file
	KeyPath    string // Path to client key file
	CAPath     string // Path to CA certificate file
	ServerName string // Server name for certificate verification
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
	Name       string       `json:"name"`
	Database   string       `json:"database"`
	Engine     string       `json:"engine"`
	TotalRows  *uint64      `json:"total_rows"`
	TotalBytes *uint64      `json:"total_bytes"`
	Columns    []ColumnInfo `json:"columns"`
}

// ClickHouseClient represents a client for interacting with ClickHouse
type ClickHouseClient struct {
	conn clickhouse.Conn
}

// NewClickHouseClient creates a new ClickHouse client
func NewClickHouseClient(config Config) (*ClickHouseClient, error) {
	options := &clickhouse.Options{
		Addr: []string{fmt.Sprintf("%s:%d", config.Host, config.Port)},
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
		return nil, fmt.Errorf("failed to connect to ClickHouse: %v", err)
	}

	// Test connection
	if err := conn.Ping(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to ping ClickHouse: %v", err)
	}

	return &ClickHouseClient{conn: conn}, nil
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
	if DatabasesData == nil {
		_, err := c.getTablesRelations()
		if err != nil {
			return nil, fmt.Errorf("failed to get table relations: %v", err)
		}
	}

	return DatabasesData, nil
}

// GenerateMermaidSchema generates a Mermaid schema for a table and its relationships
func (c *ClickHouseClient) GenerateMermaidSchema(dbName, tableName string) (string, error) {
	// Get the table schema
	table := dbName + "." + tableName

	// Start building the Mermaid schema
	var sb strings.Builder
	sb.WriteString("flowchart TB\n")

	tablesRelations, err := c.getTablesRelations()
	if err != nil {
		return "", fmt.Errorf("failed to get table relations: %v", err)
	}

	// Generate node for the main table with additional info
	nodeContent := c.generateTableNodeContent(table)
	sb.WriteString(fmt.Sprintf("    %d[\"%s\"]\n\n", city.Hash32([]byte(table)), nodeContent))
	sb.WriteString(fmt.Sprintf("    style %d fill:#FF6D00,stroke:#AA00FF,color:#FFFFFF\n\n", city.Hash32([]byte(table))))

	seen := make(map[string]bool)
	c.getRelationsNext(&sb, tablesRelations, table, &seen)
	c.getRelationsBack(&sb, tablesRelations, table, &seen)

	return sb.String(), nil
}

func (c *ClickHouseClient) generateTableNodeContent(table string) string {
	if metadata, exists := TableMetadata[table]; exists && metadata.TotalRows != nil {
		return fmt.Sprintf(
			"%s<br><small>Rows: <b>%s</b> Size: <b>%s</b></small>",
			table,
			formatRows(metadata.TotalRows),
			formatBytes(metadata.TotalBytes),
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

// GetTableColumns returns detailed column information for a specific table
func (c *ClickHouseClient) GetTableColumns(database, table string) (*TableDetails, error) {
	ctx := context.Background()

	// First get basic table info
	tableQuery := `
		SELECT engine, total_rows, total_bytes 
		FROM system.tables 
		WHERE database = ? AND name = ?
	`

	var engine string
	var totalRows, totalBytes *uint64

	row := c.conn.QueryRow(ctx, tableQuery, database, table)
	if err := row.Scan(&engine, &totalRows, &totalBytes); err != nil {
		return nil, fmt.Errorf("failed to get table info: %v", err)
	}

	// Get column information
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
		Name:       table,
		Database:   database,
		Engine:     engine,
		TotalRows:  totalRows,
		TotalBytes: totalBytes,
		Columns:    columns,
	}, nil
}

// Close closes the ClickHouse connection
func (c *ClickHouseClient) Close() error {
	return c.conn.Close()
}
