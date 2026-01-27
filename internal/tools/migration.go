// Package tools provides MCP tool registration and management.
package tools

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/dongrv/mcp-server-for-mysql/internal/mysql"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// MigrateParams represents the parameters for database migration.
type MigrateParams struct {
	MigrationSQL string `json:"migration_sql"`
}

// MigrateHandler handles database migration operations.
type MigrateHandler struct {
	baseHandler
}

// NewMigrateHandler creates a new migrate handler.
func NewMigrateHandler(pool *mysql.Pool) *MigrateHandler {
	return &MigrateHandler{
		baseHandler: newBaseHandler(
			"mysql_migrate",
			"执行数据库迁移",
			pool,
		),
	}
}

// Handle processes migration requests.
func (h *MigrateHandler) Handle(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, any, error) {
	var params MigrateParams
	if err := json.Unmarshal(req.Params.Arguments, &params); err != nil {
		return nil, nil, fmt.Errorf("invalid parameters: %w", err)
	}

	if params.MigrationSQL == "" {
		return nil, nil, fmt.Errorf("migration_sql cannot be empty")
	}

	// Execute migration within a transaction
	err := h.pool.WithTransaction(ctx, nil, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, params.MigrationSQL)
		return err
	})

	if err != nil {
		return nil, nil, fmt.Errorf("failed to execute migration: %w", err)
	}

	// Prepare response
	response := map[string]interface{}{
		"migration_sql": params.MigrationSQL,
		"message":       "Migration executed successfully",
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: formatJSON(response)},
		},
	}, response, nil
}

// PoolStatusHandler handles database connection pool status requests.
type PoolStatusHandler struct {
	baseHandler
}

// NewPoolStatusHandler creates a new pool status handler.
func NewPoolStatusHandler(pool *mysql.Pool) *PoolStatusHandler {
	return &PoolStatusHandler{
		baseHandler: newBaseHandler(
			"mysql_pool_status",
			"获取数据库连接池状态",
			pool,
		),
	}
}

// Handle processes pool status requests.
func (h *PoolStatusHandler) Handle(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, any, error) {
	// Get pool statistics
	stats := h.pool.Stats()
	config := h.pool.Config()

	// Prepare response
	response := map[string]interface{}{
		"pool_config": map[string]interface{}{
			"max_open_connections": config.MaxOpenConns,
			"max_idle_connections": config.MaxIdleConns,
			"conn_max_lifetime":    config.ConnMaxLifetime.String(),
			"conn_max_idle_time":   config.ConnMaxIdleTime.String(),
		},
		"pool_stats": map[string]interface{}{
			"max_open_connections": stats.MaxOpenConnections,
			"open_connections":     stats.OpenConnections,
			"in_use":               stats.InUse,
			"idle":                 stats.Idle,
			"wait_count":           stats.WaitCount,
			"wait_duration":        stats.WaitDuration.String(),
			"max_idle_closed":      stats.MaxIdleClosed,
			"max_lifetime_closed":  stats.MaxLifetimeClosed,
		},
		"timestamp": time.Now().Format(time.RFC3339),
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: formatJSON(response)},
		},
	}, response, nil
}
