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

// ColumnDefinition represents a single column definition.
type ColumnDefinition struct {
	Name         string `json:"name"`
	Type         string `json:"type"`
	Nullable     bool   `json:"nullable,omitempty"`
	DefaultValue string `json:"default_value,omitempty"`
	AfterColumn  string `json:"after_column,omitempty"`
}

// AddColumnsParams represents the parameters for adding columns.
type AddColumnsParams struct {
	TableName string             `json:"table_name"`
	Columns   []ColumnDefinition `json:"columns"`
}

// DropColumnsParams represents the parameters for dropping columns.
type DropColumnsParams struct {
	TableName string   `json:"table_name"`
	Columns   []string `json:"columns"`
}

// ModifyColumnsParams represents the parameters for modifying columns.
type ModifyColumnsParams struct {
	TableName string             `json:"table_name"`
	Columns   []ColumnDefinition `json:"columns"`
}

// AddColumnsHandler handles adding multiple columns to a table.
type AddColumnsHandler struct {
	baseHandler
}

// NewAddColumnsHandler creates a new add columns handler.
func NewAddColumnsHandler(pool *mysql.Pool) *AddColumnsHandler {
	return &AddColumnsHandler{
		baseHandler: newBaseHandler(
			"mysql_add_columns",
			"为表添加多个字段",
			pool,
		),
	}
}

// Handle processes add columns requests.
func (h *AddColumnsHandler) Handle(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, any, error) {
	var params AddColumnsParams
	if err := json.Unmarshal(req.Params.Arguments, &params); err != nil {
		return nil, nil, fmt.Errorf("invalid parameters: %w", err)
	}

	if params.TableName == "" {
		return nil, nil, fmt.Errorf("table_name cannot be empty")
	}
	if len(params.Columns) == 0 {
		return nil, nil, fmt.Errorf("columns cannot be empty")
	}

	// Execute within a transaction
	var executedSQLs []string
	err := h.pool.WithTransaction(ctx, nil, func(tx *sql.Tx) error {
		for _, column := range params.Columns {
			if column.Name == "" {
				return fmt.Errorf("column name cannot be empty")
			}
			if column.Type == "" {
				return fmt.Errorf("column type cannot be empty for column %s", column.Name)
			}

			// Build column definition
			columnDef := fmt.Sprintf("`%s` %s", column.Name, column.Type)
			if !column.Nullable {
				columnDef += " NOT NULL"
			}
			if column.DefaultValue != "" {
				columnDef += fmt.Sprintf(" DEFAULT '%s'", column.DefaultValue)
			}
			if column.AfterColumn != "" {
				columnDef += fmt.Sprintf(" AFTER `%s`", column.AfterColumn)
			}

			// Execute ALTER TABLE
			sql := fmt.Sprintf("ALTER TABLE `%s` ADD COLUMN %s", params.TableName, columnDef)
			if _, err := tx.ExecContext(ctx, sql); err != nil {
				return fmt.Errorf("failed to add column %s: %w", column.Name, err)
			}
			executedSQLs = append(executedSQLs, sql)
		}
		return nil
	})

	if err != nil {
		return nil, nil, fmt.Errorf("failed to add columns: %w", err)
	}

	// Prepare response
	response := map[string]interface{}{
		"table_name":    params.TableName,
		"added_columns": len(params.Columns),
		"executed_sqls": executedSQLs,
		"message":       fmt.Sprintf("成功添加 %d 个字段", len(params.Columns)),
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: formatJSON(response)},
		},
	}, response, nil
}

// DropColumnsHandler handles dropping multiple columns from a table.
type DropColumnsHandler struct {
	baseHandler
}

// NewDropColumnsHandler creates a new drop columns handler.
func NewDropColumnsHandler(pool *mysql.Pool) *DropColumnsHandler {
	return &DropColumnsHandler{
		baseHandler: newBaseHandler(
			"mysql_drop_columns",
			"从表中删除多个字段",
			pool,
		),
	}
}

// Handle processes drop columns requests.
func (h *DropColumnsHandler) Handle(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, any, error) {
	var params DropColumnsParams
	if err := json.Unmarshal(req.Params.Arguments, &params); err != nil {
		return nil, nil, fmt.Errorf("invalid parameters: %w", err)
	}

	if params.TableName == "" {
		return nil, nil, fmt.Errorf("table_name cannot be empty")
	}
	if len(params.Columns) == 0 {
		return nil, nil, fmt.Errorf("columns cannot be empty")
	}

	// Execute within a transaction
	var executedSQLs []string
	err := h.pool.WithTransaction(ctx, nil, func(tx *sql.Tx) error {
		for _, column := range params.Columns {
			if column == "" {
				return fmt.Errorf("column name cannot be empty")
			}

			// Execute ALTER TABLE
			sql := fmt.Sprintf("ALTER TABLE `%s` DROP COLUMN `%s`", params.TableName, column)
			if _, err := tx.ExecContext(ctx, sql); err != nil {
				return fmt.Errorf("failed to drop column %s: %w", column, err)
			}
			executedSQLs = append(executedSQLs, sql)
		}
		return nil
	})

	if err != nil {
		return nil, nil, fmt.Errorf("failed to drop columns: %w", err)
	}

	// Prepare response
	response := map[string]interface{}{
		"table_name":      params.TableName,
		"dropped_columns": len(params.Columns),
		"executed_sqls":   executedSQLs,
		"message":         fmt.Sprintf("成功删除 %d 个字段", len(params.Columns)),
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: formatJSON(response)},
		},
	}, response, nil
}

// ModifyColumnsHandler handles modifying multiple columns in a table.
type ModifyColumnsHandler struct {
	baseHandler
}

// NewModifyColumnsHandler creates a new modify columns handler.
func NewModifyColumnsHandler(pool *mysql.Pool) *ModifyColumnsHandler {
	return &ModifyColumnsHandler{
		baseHandler: newBaseHandler(
			"mysql_modify_columns",
			"修改表的多个字段",
			pool,
		),
	}
}

// Handle processes modify columns requests.
func (h *ModifyColumnsHandler) Handle(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, any, error) {
	var params ModifyColumnsParams
	if err := json.Unmarshal(req.Params.Arguments, &params); err != nil {
		return nil, nil, fmt.Errorf("invalid parameters: %w", err)
	}

	if params.TableName == "" {
		return nil, nil, fmt.Errorf("table_name cannot be empty")
	}
	if len(params.Columns) == 0 {
		return nil, nil, fmt.Errorf("columns cannot be empty")
	}

	// Execute within a transaction
	var executedSQLs []string
	err := h.pool.WithTransaction(ctx, nil, func(tx *sql.Tx) error {
		for _, column := range params.Columns {
			if column.Name == "" {
				return fmt.Errorf("column name cannot be empty")
			}
			if column.Type == "" {
				return fmt.Errorf("column type cannot be empty for column %s", column.Name)
			}

			// Build column definition
			columnDef := fmt.Sprintf("`%s` %s", column.Name, column.Type)
			if !column.Nullable {
				columnDef += " NOT NULL"
			}
			if column.DefaultValue != "" {
				columnDef += fmt.Sprintf(" DEFAULT '%s'", column.DefaultValue)
			}

			// Execute ALTER TABLE
			sql := fmt.Sprintf("ALTER TABLE `%s` MODIFY COLUMN %s", params.TableName, columnDef)
			if _, err := tx.ExecContext(ctx, sql); err != nil {
				return fmt.Errorf("failed to modify column %s: %w", column.Name, err)
			}
			executedSQLs = append(executedSQLs, sql)
		}
		return nil
	})

	if err != nil {
		return nil, nil, fmt.Errorf("failed to modify columns: %w", err)
	}

	// Prepare response
	response := map[string]interface{}{
		"table_name":       params.TableName,
		"modified_columns": len(params.Columns),
		"executed_sqls":    executedSQLs,
		"message":          fmt.Sprintf("成功修改 %d 个字段", len(params.Columns)),
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: formatJSON(response)},
		},
	}, response, nil
}
