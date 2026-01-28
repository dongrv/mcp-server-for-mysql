// Package tools provides MCP tool registration and management.
package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/dongrv/mcp-server-for-mysql/internal/mysql"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// QueryParams represents the parameters for the mysql_query tool.
type QueryParams struct {
	Query      string   `json:"query"`
	Parameters []string `json:"parameters,omitempty"`
}

// QueryHandler handles SQL query operations.
type QueryHandler struct {
	baseHandler
}

// NewQueryHandler creates a new query handler.
func NewQueryHandler(pool *mysql.Pool) *QueryHandler {
	return &QueryHandler{
		baseHandler: newBaseHandler(
			"mysql_query",
			"执行 SQL 查询语句",
			pool,
		),
	}
}

// Handle processes SQL query requests.
func (h *QueryHandler) Handle(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, any, error) {
	var params QueryParams
	if err := json.Unmarshal(req.Params.Arguments, &params); err != nil {
		return nil, nil, fmt.Errorf("invalid parameters: %w", err)
	}

	if params.Query == "" {
		return nil, nil, fmt.Errorf("query cannot be empty")
	}

	// Execute query
	rows, err := h.pool.QueryContext(ctx, params.Query, convertParams(params.Parameters)...)
	if err != nil {
		return nil, nil, fmt.Errorf("query failed: %w", err)
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
		"results": results,
		"count":   len(results),
		"query":   params.Query,
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: formatJSON(response)},
		},
	}, response, nil
}
