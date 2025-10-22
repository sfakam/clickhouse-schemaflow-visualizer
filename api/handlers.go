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
	api := router.Group("/api")
	{
		api.GET("/databases", h.GetDatabases)
		api.GET("/schema/:database/:table", h.GetTableSchema)
		api.GET("/database/:database/schema", h.GetDatabaseSchema)
		api.GET("/table/:database/:table", h.GetTableDetails)
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
