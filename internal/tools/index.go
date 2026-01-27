// Package tools provides MCP tool registration and management.
package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/dongrv/mcp-server-for-mysql/internal/mysql"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// CreateIndexParams represents the parameters for creating indexes.
type CreateIndexParams struct {
	TableName string `json:"table_name"`
	IndexName string `json:"index_name"`
	Columns   string `json:"columns"`
	IndexType string `json:"index_type,omitempty"`
}

// DropIndexParams represents the parameters for dropping indexes.
type DropIndexParams struct {
	TableName string `json:"table_name"`
	IndexName string `json:"index_name"`
}

// CreateIndexHandler handles creating indexes.
type CreateIndexHandler struct {
	baseHandler
}

// NewCreateIndexHandler creates a new create index handler.
func NewCreateIndexHandler(pool *mysql.Pool) *CreateIndexHandler {
	return &CreateIndexHandler{
		baseHandler: newBaseHandler(
			"mysql_create_index",
			"创建索引",
			pool,
		),
	}
}

// Handle processes create index requests.
func (h *CreateIndexHandler) Handle(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, any, error) {
	var params CreateIndexParams
	if err := json.Unmarshal(req.Params.Arguments, &params); err != nil {
		return nil, nil, fmt.Errorf("invalid parameters: %w", err)
	}

	if params.TableName == "" {
		return nil, nil, fmt.Errorf("table_name cannot be empty")
	}
	if params.IndexName == "" {
		return nil, nil, fmt.Errorf("index_name cannot be empty")
	}
	if params.Columns == "" {
		return nil, nil, fmt.Errorf("columns cannot be empty")
	}

	// Build CREATE INDEX SQL
	var sql string
	if params.IndexType != "" {
		sql = fmt.Sprintf("CREATE %s INDEX %s ON %s (%s)",
			params.IndexType, params.IndexName, params.TableName, params.Columns)
	} else {
		sql = fmt.Sprintf("CREATE INDEX %s ON %s (%s)",
			params.IndexName, params.TableName, params.Columns)
	}

	// Execute SQL
	_, err := h.pool.ExecContext(ctx, sql)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create index: %w", err)
	}

	// Prepare response
	response := map[string]interface{}{
		"table_name": params.TableName,
		"index_name": params.IndexName,
		"columns":    params.Columns,
		"index_type": params.IndexType,
		"sql":        sql,
		"message":    "Index created successfully",
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: formatJSON(response)},
		},
	}, response, nil
}

// DropIndexHandler handles dropping indexes.
type DropIndexHandler struct {
	baseHandler
}

// NewDropIndexHandler creates a new drop index handler.
func NewDropIndexHandler(pool *mysql.Pool) *DropIndexHandler {
	return &DropIndexHandler{
		baseHandler: newBaseHandler(
			"mysql_drop_index",
			"删除索引",
			pool,
		),
	}
}

// Handle processes drop index requests.
func (h *DropIndexHandler) Handle(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, any, error) {
	var params DropIndexParams
	if err := json.Unmarshal(req.Params.Arguments, &params); err != nil {
		return nil, nil, fmt.Errorf("invalid parameters: %w", err)
	}

	if params.TableName == "" {
		return nil, nil, fmt.Errorf("table_name cannot be empty")
	}
	if params.IndexName == "" {
		return nil, nil, fmt.Errorf("index_name cannot be empty")
	}

	// Build DROP INDEX SQL
	sql := fmt.Sprintf("DROP INDEX %s ON %s", params.IndexName, params.TableName)

	// Execute SQL
	_, err := h.pool.ExecContext(ctx, sql)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to drop index: %w", err)
	}

	// Prepare response
	response := map[string]interface{}{
		"table_name": params.TableName,
		"index_name": params.IndexName,
		"sql":        sql,
		"message":    "Index dropped successfully",
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: formatJSON(response)},
		},
	}, response, nil
}

// ListIndexesHandler handles listing indexes for a table.
type ListIndexesHandler struct {
	baseHandler
}

// NewListIndexesHandler creates a new list indexes handler.
func NewListIndexesHandler(pool *mysql.Pool) *ListIndexesHandler {
	return &ListIndexesHandler{
		baseHandler: newBaseHandler(
			"mysql_list_indexes",
			"列出表的所有索引",
			pool,
		),
	}
}

// Handle processes list indexes requests.
func (h *ListIndexesHandler) Handle(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, any, error) {
	var params TableParams
	if err := json.Unmarshal(req.Params.Arguments, &params); err != nil {
		return nil, nil, fmt.Errorf("invalid parameters: %w", err)
	}

	if params.TableName == "" {
		return nil, nil, fmt.Errorf("table_name cannot be empty")
	}

	// Execute SHOW INDEX query
	rows, err := h.pool.QueryContext(ctx, "SHOW INDEX FROM "+params.TableName)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list indexes: %w", err)
	}
	defer rows.Close()

	// Get column information
	columns, err := rows.Columns()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get columns: %w", err)
	}

	// Process results
	var results []map[string]interface{}
	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, nil, fmt.Errorf("failed to scan row: %w", err)
		}

		// Convert values to appropriate types
		rowData := make(map[string]interface{})
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
	response := map[string]interface{}{
		"table_name": params.TableName,
		"indexes":    results,
		"count":      len(results),
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: formatJSON(response)},
		},
	}, response, nil
}
