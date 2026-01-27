// Package tools provides MCP tool registration and management.
package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/dongrv/mcp-server-for-mysql/internal/mysql"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ExecuteParams represents the parameters for the mysql_execute tool.
type ExecuteParams struct {
	SQL        string   `json:"sql"`
	Parameters []string `json:"parameters,omitempty"`
}

// ExecuteHandler handles SQL execution operations (INSERT, UPDATE, DELETE).
type ExecuteHandler struct {
	baseHandler
}

// NewExecuteHandler creates a new execute handler.
func NewExecuteHandler(pool *mysql.Pool) *ExecuteHandler {
	return &ExecuteHandler{
		baseHandler: newBaseHandler(
			"mysql_execute",
			"执行 SQL 更新语句（INSERT, UPDATE, DELETE）",
			pool,
		),
	}
}

// Handle processes SQL execution requests.
func (h *ExecuteHandler) Handle(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, any, error) {
	var params ExecuteParams
	if err := json.Unmarshal(req.Params.Arguments, &params); err != nil {
		return nil, nil, fmt.Errorf("invalid parameters: %w", err)
	}

	if params.SQL == "" {
		return nil, nil, fmt.Errorf("SQL cannot be empty")
	}

	// Execute SQL
	result, err := h.pool.ExecContext(ctx, params.SQL, convertParams(params.Parameters)...)
	if err != nil {
		return nil, nil, fmt.Errorf("execute failed: %w", err)
	}

	// Get execution results
	rowsAffected, _ := result.RowsAffected()
	lastInsertID, _ := result.LastInsertId()

	// Prepare response
	response := map[string]interface{}{
		"rows_affected":  rowsAffected,
		"last_insert_id": lastInsertID,
		"sql":            params.SQL,
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: formatJSON(response)},
		},
	}, response, nil
}
