// Package tools provides MCP tool registration and management.
package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/dongrv/mcp-server-for-mysql/internal/mysql"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// TableParams represents the parameters for table-related tools.
type TableParams struct {
	TableName string `json:"table_name"`
}

// CreateTableParams represents the parameters for creating tables.
type CreateTableParams struct {
	TableName string `json:"table_name"`
	Columns   string `json:"columns"`
}

// ListTablesHandler handles listing all tables in the database.
type ListTablesHandler struct {
	baseHandler
}

// NewListTablesHandler creates a new list tables handler.
func NewListTablesHandler(pool *mysql.Pool) *ListTablesHandler {
	return &ListTablesHandler{
		baseHandler: newBaseHandler(
			"mysql_list_tables",
			"列出数据库中的所有表",
			pool,
		),
	}
}

// Handle processes list tables requests.
func (h *ListTablesHandler) Handle(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, any, error) {
	// Execute SHOW TABLES query
	rows, err := h.pool.QueryContext(ctx, "SHOW TABLES")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list tables: %w", err)
	}
	defer rows.Close()

	// Process results
	var tables []string
	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			return nil, nil, fmt.Errorf("failed to scan table name: %w", err)
		}
		tables = append(tables, tableName)
	}

	if err := rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("rows iteration error: %w", err)
	}

	// Prepare response
	response := map[string]any{
		"tables": tables,
		"count":  len(tables),
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: formatJSON(response)},
		},
	}, response, nil
}

// DescribeTableHandler handles describing table structure.
type DescribeTableHandler struct {
	baseHandler
}

// NewDescribeTableHandler creates a new describe table handler.
func NewDescribeTableHandler(pool *mysql.Pool) *DescribeTableHandler {
	return &DescribeTableHandler{
		baseHandler: newBaseHandler(
			"mysql_describe_table",
			"描述表结构",
			pool,
		),
	}
}

// Handle processes describe table requests.
func (h *DescribeTableHandler) Handle(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, any, error) {
	var params TableParams
	if err := json.Unmarshal(req.Params.Arguments, &params); err != nil {
		return nil, nil, fmt.Errorf("invalid parameters: %w", err)
	}

	if params.TableName == "" {
		return nil, nil, fmt.Errorf("table_name cannot be empty")
	}

	// Execute DESCRIBE query
	rows, err := h.pool.QueryContext(ctx, "DESCRIBE "+params.TableName)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to describe table: %w", err)
	}
	defer rows.Close()

	// Get column information
	columns, err := rows.Columns()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get columns: %w", err)
	}

	// Process results
	var results []map[string]any
	for rows.Next() {
		values := make([]any, len(columns))
		valuePtrs := make([]any, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, nil, fmt.Errorf("failed to scan row: %w", err)
		}

		// Convert values to appropriate types
		rowData := make(map[string]any)
		for i, col := range columns {
			val := values[i]
			if b, ok := val.([]byte); ok {
				rowData[col] = string(b)
			} else {
				rowData[col] = val
			}
		}
		results = append(results, rowData)
	}

	if err := rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("rows iteration error: %w", err)
	}

	// Prepare response
	response := map[string]any{
		"table_name": params.TableName,
		"columns":    results,
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: formatJSON(response)},
		},
	}, response, nil
}

// CreateTableHandler handles creating new tables.
type CreateTableHandler struct {
	baseHandler
}

// NewCreateTableHandler creates a new create table handler.
func NewCreateTableHandler(pool *mysql.Pool) *CreateTableHandler {
	return &CreateTableHandler{
		baseHandler: newBaseHandler(
			"mysql_create_table",
			"创建新表",
			pool,
		),
	}
}

// Handle processes create table requests.
func (h *CreateTableHandler) Handle(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, any, error) {
	var params CreateTableParams
	if err := json.Unmarshal(req.Params.Arguments, &params); err != nil {
		return nil, nil, fmt.Errorf("invalid parameters: %w", err)
	}

	if params.TableName == "" {
		return nil, nil, fmt.Errorf("table_name cannot be empty")
	}
	if params.Columns == "" {
		return nil, nil, fmt.Errorf("columns cannot be empty")
	}

	// Build and execute CREATE TABLE SQL
	sql := fmt.Sprintf("CREATE TABLE %s (%s)", params.TableName, params.Columns)
	_, err := h.pool.ExecContext(ctx, sql)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create table: %w", err)
	}

	// Prepare response
	response := map[string]any{
		"table_name": params.TableName,
		"sql":        sql,
		"message":    "Table created successfully",
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: formatJSON(response)},
		},
	}, response, nil
}

// DropTableHandler handles dropping tables.
type DropTableHandler struct {
	baseHandler
}

// NewDropTableHandler creates a new drop table handler.
func NewDropTableHandler(pool *mysql.Pool) *DropTableHandler {
	return &DropTableHandler{
		baseHandler: newBaseHandler(
			"mysql_drop_table",
			"删除表",
			pool,
		),
	}
}

// Handle processes drop table requests.
func (h *DropTableHandler) Handle(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, any, error) {
	var params TableParams
	if err := json.Unmarshal(req.Params.Arguments, &params); err != nil {
		return nil, nil, fmt.Errorf("invalid parameters: %w", err)
	}

	if params.TableName == "" {
		return nil, nil, fmt.Errorf("table_name cannot be empty")
	}

	// Execute DROP TABLE SQL
	sql := fmt.Sprintf("DROP TABLE %s", params.TableName)
	_, err := h.pool.ExecContext(ctx, sql)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to drop table: %w", err)
	}

	// Prepare response
	response := map[string]any{
		"table_name": params.TableName,
		"sql":        sql,
		"message":    "Table dropped successfully",
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: formatJSON(response)},
		},
	}, response, nil
}
