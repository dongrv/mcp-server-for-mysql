// Package tools provides MCP tool registration and management.
package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/dongrv/mcp-server-for-mysql/internal/mysql"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// TransactionParams represents the parameters for transaction tools.
type TransactionParams struct {
	TransactionID string `json:"transaction_id"`
}

// BeginTransactionHandler handles beginning new transactions.
type BeginTransactionHandler struct {
	baseHandler
	txManager *mysql.TxManager
}

// NewBeginTransactionHandler creates a new begin transaction handler.
func NewBeginTransactionHandler(txManager *mysql.TxManager) *BeginTransactionHandler {
	return &BeginTransactionHandler{
		baseHandler: newBaseHandler(
			"mysql_begin_transaction",
			"开始一个新的事务",
			nil, // No pool needed for this handler
		),
		txManager: txManager,
	}
}

// Handle processes begin transaction requests.
func (h *BeginTransactionHandler) Handle(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, any, error) {
	// No parameters needed for begin transaction
	txID, err := h.txManager.Begin(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to begin transaction: %w", err)
	}

	response := map[string]any{
		"transaction_id": txID,
		"message":        "Transaction started successfully",
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: formatJSON(response)},
		},
	}, response, nil
}

// CommitTransactionHandler handles committing transactions.
type CommitTransactionHandler struct {
	baseHandler
	txManager *mysql.TxManager
}

// NewCommitTransactionHandler creates a new commit transaction handler.
func NewCommitTransactionHandler(txManager *mysql.TxManager) *CommitTransactionHandler {
	return &CommitTransactionHandler{
		baseHandler: newBaseHandler(
			"mysql_commit_transaction",
			"提交当前事务",
			nil, // No pool needed for this handler
		),
		txManager: txManager,
	}
}

// Handle processes commit transaction requests.
func (h *CommitTransactionHandler) Handle(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, any, error) {
	var params TransactionParams
	if err := json.Unmarshal(req.Params.Arguments, &params); err != nil {
		return nil, nil, fmt.Errorf("invalid parameters: %w", err)
	}

	if params.TransactionID == "" {
		return nil, nil, fmt.Errorf("transaction_id cannot be empty")
	}

	if err := h.txManager.Commit(ctx, params.TransactionID); err != nil {
		return nil, nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	response := map[string]any{
		"transaction_id": params.TransactionID,
		"message":        "Transaction committed successfully",
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: formatJSON(response)},
		},
	}, response, nil
}

// RollbackTransactionHandler handles rolling back transactions.
type RollbackTransactionHandler struct {
	baseHandler
	txManager *mysql.TxManager
}

// NewRollbackTransactionHandler creates a new rollback transaction handler.
func NewRollbackTransactionHandler(txManager *mysql.TxManager) *RollbackTransactionHandler {
	return &RollbackTransactionHandler{
		baseHandler: newBaseHandler(
			"mysql_rollback_transaction",
			"回滚当前事务",
			nil, // No pool needed for this handler
		),
		txManager: txManager,
	}
}

// Handle processes rollback transaction requests.
func (h *RollbackTransactionHandler) Handle(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, any, error) {
	var params TransactionParams
	if err := json.Unmarshal(req.Params.Arguments, &params); err != nil {
		return nil, nil, fmt.Errorf("invalid parameters: %w", err)
	}

	if params.TransactionID == "" {
		return nil, nil, fmt.Errorf("transaction_id cannot be empty")
	}

	if err := h.txManager.Rollback(ctx, params.TransactionID); err != nil {
		return nil, nil, fmt.Errorf("failed to rollback transaction: %w", err)
	}

	response := map[string]any{
		"transaction_id": params.TransactionID,
		"message":        "Transaction rolled back successfully",
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: formatJSON(response)},
		},
	}, response, nil
}
