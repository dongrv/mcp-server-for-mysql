// Package tools provides MCP tool registration and management.
package tools

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/dongrv/mcp-server-for-mysql/internal/mysql"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// RenameTableParams represents the parameters for renaming tables.
type RenameTableParams struct {
	OldTableName string `json:"old_table_name"`
	NewTableName string `json:"new_table_name"`
}

// CopyTableParams represents the parameters for copying tables.
type CopyTableParams struct {
	SourceTable      string `json:"source_table"`
	DestinationTable string `json:"destination_table"`
	CopyData         bool   `json:"copy_data,omitempty"`
}

// RenameTableHandler handles renaming tables.
type RenameTableHandler struct {
	baseHandler
}

// NewRenameTableHandler creates a new rename table handler.
func NewRenameTableHandler(pool *mysql.Pool) *RenameTableHandler {
	return &RenameTableHandler{
		baseHandler: newBaseHandler(
			"mysql_rename_table",
			"重命名表",
			pool,
		),
	}
}

// Handle processes rename table requests.
func (h *RenameTableHandler) Handle(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, any, error) {
	var params RenameTableParams
	if err := json.Unmarshal(req.Params.Arguments, &params); err != nil {
		return nil, nil, fmt.Errorf("invalid parameters: %w", err)
	}

	if params.OldTableName == "" {
		return nil, nil, fmt.Errorf("old_table_name cannot be empty")
	}
	if params.NewTableName == "" {
		return nil, nil, fmt.Errorf("new_table_name cannot be empty")
	}
	if params.OldTableName == params.NewTableName {
		return nil, nil, fmt.Errorf("old_table_name and new_table_name cannot be the same")
	}

	// Execute RENAME TABLE SQL
	sql := fmt.Sprintf("RENAME TABLE `%s` TO `%s`", params.OldTableName, params.NewTableName)
	_, err := h.pool.ExecContext(ctx, sql)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to rename table: %w", err)
	}

	// Prepare response
	response := map[string]any{
		"old_table_name": params.OldTableName,
		"new_table_name": params.NewTableName,
		"sql":            sql,
		"message":        "表重命名成功",
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: formatJSON(response)},
		},
	}, response, nil
}

// CopyTableHandler handles copying table structure and optionally data.
type CopyTableHandler struct {
	baseHandler
}

// NewCopyTableHandler creates a new copy table handler.
func NewCopyTableHandler(pool *mysql.Pool) *CopyTableHandler {
	return &CopyTableHandler{
		baseHandler: newBaseHandler(
			"mysql_copy_table",
			"复制表结构和数据",
			pool,
		),
	}
}

// Handle processes copy table requests.
func (h *CopyTableHandler) Handle(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, any, error) {
	var params CopyTableParams
	if err := json.Unmarshal(req.Params.Arguments, &params); err != nil {
		return nil, nil, fmt.Errorf("invalid parameters: %w", err)
	}

	if params.SourceTable == "" {
		return nil, nil, fmt.Errorf("source_table cannot be empty")
	}
	if params.DestinationTable == "" {
		return nil, nil, fmt.Errorf("destination_table cannot be empty")
	}
	if params.SourceTable == params.DestinationTable {
		return nil, nil, fmt.Errorf("source_table and destination_table cannot be the same")
	}

	// Execute within a transaction
	err := h.pool.WithTransaction(ctx, nil, func(tx *sql.Tx) error {
		// 1. Copy table structure
		createTableSQL := fmt.Sprintf("CREATE TABLE `%s` LIKE `%s`", params.DestinationTable, params.SourceTable)
		if _, err := tx.ExecContext(ctx, createTableSQL); err != nil {
			return fmt.Errorf("failed to copy table structure: %w", err)
		}

		// 2. Copy data if requested
		if params.CopyData {
			copyDataSQL := fmt.Sprintf("INSERT INTO `%s` SELECT * FROM `%s`", params.DestinationTable, params.SourceTable)
			if _, err := tx.ExecContext(ctx, copyDataSQL); err != nil {
				return fmt.Errorf("failed to copy table data: %w", err)
			}
		}

		return nil
	})

	if err != nil {
		return nil, nil, fmt.Errorf("failed to copy table: %w", err)
	}

	// Prepare response
	response := map[string]any{
		"source_table":      params.SourceTable,
		"destination_table": params.DestinationTable,
		"copy_data":         params.CopyData,
		"message":           "表复制成功",
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: formatJSON(response)},
		},
	}, response, nil
}

// CopyTableStructureHandler handles copying only table structure (without data).
type CopyTableStructureHandler struct {
	baseHandler
}

// NewCopyTableStructureHandler creates a new copy table structure handler.
func NewCopyTableStructureHandler(pool *mysql.Pool) *CopyTableStructureHandler {
	return &CopyTableStructureHandler{
		baseHandler: newBaseHandler(
			"mysql_copy_table_structure",
			"仅复制表结构（不复制数据）",
			pool,
		),
	}
}

// Handle processes copy table structure requests.
func (h *CopyTableStructureHandler) Handle(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, any, error) {
	var params CopyTableParams
	if err := json.Unmarshal(req.Params.Arguments, &params); err != nil {
		return nil, nil, fmt.Errorf("invalid parameters: %w", err)
	}

	// Set copy_data to false for structure-only copy
	params.CopyData = false

	// Delegate to CopyTableHandler
	handler := NewCopyTableHandler(h.pool)
	return handler.Handle(ctx, req)
}
