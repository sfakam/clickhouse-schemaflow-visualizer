package api

import (
	"net/http"

	"github.com/fulgerX2007/clickhouse-schemaflow-visualizer/models"
	"github.com/gin-gonic/gin"
)

// Handler holds the dependencies for API handlers
type Handler struct {
	clickhouse *models.ClickHouseClient
}

// NewHandler creates a new Handler instance
func NewHandler(clickhouse *models.ClickHouseClient) *Handler {
	return &Handler{
		clickhouse: clickhouse,
	}
}

// RegisterRoutes registers all API routes
func (h *Handler) RegisterRoutes(router *gin.Engine) {
	// New clean JSON API endpoints (matching API.md specification)
	api := router.Group("/api")
	{
		api.GET("/databases", h.GetDatabasesClean)
		api.GET("/table/:database/:table", h.GetTableDetailsClean)
		api.GET("/table/:database/:table/relationships", h.GetTableRelationships)
	}
	
	// Render/visualization endpoints (Mermaid diagrams, HTML, stats, etc.)
	render := router.Group("/api/render")
	{
		render.GET("/databases", h.GetDatabases)
		render.GET("/schema/:database/:table", h.GetTableSchema)
		render.GET("/database/:database/schema", h.GetDatabaseSchema)
		render.GET("/database/:database/stats", h.GetDatabaseStats)
	}
}

// GetDatabases returns a list of all databases and their tables
func (h *Handler) GetDatabases(c *gin.Context) {
	databases, err := h.clickhouse.GetDatabases()
	if err != nil {
		c.JSON(
			http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			},
		)
		return
	}

	c.JSON(http.StatusOK, databases)
}

// GetTableSchema returns a Mermaid schema for the selected table
func (h *Handler) GetTableSchema(c *gin.Context) {
	database := c.Param("database")
	table := c.Param("table")

	if database == "" || table == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "database and table parameters are required"})

		return
	}

	schema, err := h.clickhouse.GenerateMermaidSchema(database, table)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})

		return
	}

	c.JSON(http.StatusOK, gin.H{"schema": schema})
}

// GetTableDetails returns detailed information about the selected table
func (h *Handler) GetTableDetails(c *gin.Context) {
	database := c.Param("database")
	table := c.Param("table")

	if database == "" || table == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "database and table parameters are required"})
		return
	}

	details, err := h.clickhouse.GetTableColumns(database, table)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, details)
}

// GetDatabaseSchema returns a comprehensive Mermaid schema for an entire database
func (h *Handler) GetDatabaseSchema(c *gin.Context) {
	database := c.Param("database")

	if database == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "database parameter is required"})
		return
	}

	// Get optional query parameters for filtering
	engines := c.QueryArray("engines")  // e.g., ?engines=MergeTree&engines=ReplicatedMergeTree
	includeMetadata := c.DefaultQuery("metadata", "true") == "true"

	schema, err := h.clickhouse.GenerateDatabaseMermaidSchema(database, engines, includeMetadata)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"database": database,
		"schema": schema,
		"filters": gin.H{
			"engines": engines,
			"metadata": includeMetadata,
		},
	})
}

// GetDatabaseStats returns statistics for a database
func (h *Handler) GetDatabaseStats(c *gin.Context) {
	database := c.Param("database")

	if database == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "database parameter is required"})
		return
	}

	stats, err := h.clickhouse.GetDatabaseStats(database)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// ========== NEW CLEAN JSON API HANDLERS (API.md spec) ==========

// GetDatabasesClean returns databases and tables in clean JSON format
func (h *Handler) GetDatabasesClean(c *gin.Context) {
	databases, err := h.clickhouse.GetDatabasesClean()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, databases)
}

// GetTableDetailsClean returns full table details including metadata and columns
func (h *Handler) GetTableDetailsClean(c *gin.Context) {
	database := c.Param("database")
	table := c.Param("table")

	if database == "" || table == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "database and table parameters are required"})
		return
	}

	details, err := h.clickhouse.GetTableColumns(database, table)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, details)
}

// GetTableRelationships returns table relationships in clean JSON format
func (h *Handler) GetTableRelationships(c *gin.Context) {
	database := c.Param("database")
	table := c.Param("table")

	if database == "" || table == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "database and table parameters are required"})
		return
	}

	relationships, err := h.clickhouse.GetTableRelationshipsClean(database, table)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, relationships)
}
