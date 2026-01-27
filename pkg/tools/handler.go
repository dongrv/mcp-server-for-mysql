package tools

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/dongrv/mcp-mysql/pkg/database"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ToolHandler 处理所有 MCP 工具
type ToolHandler struct {
	dbPool *database.ConnectionPool
}

// NewToolHandler 创建新的工具处理器
func NewToolHandler(dbPool *database.ConnectionPool) *ToolHandler {
	return &ToolHandler{
		dbPool: dbPool,
	}
}

// RegisterTools 注册所有工具到 MCP 服务器
func (h *ToolHandler) RegisterTools(server *mcp.Server) {
	// 查询工具
	mcp.AddTool(server, &mcp.Tool{
		Name:        "mysql_query",
		Description: "执行 SQL 查询语句",
	}, h.handleQuery)

	// 执行工具
	mcp.AddTool(server, &mcp.Tool{
		Name:        "mysql_execute",
		Description: "执行 SQL 更新语句（INSERT, UPDATE, DELETE）",
	}, h.handleExecute)

	// 开始事务工具
	mcp.AddTool(server, &mcp.Tool{
		Name:        "mysql_begin_transaction",
		Description: "开始一个新的事务",
	}, h.handleBeginTransaction)

	// 提交事务工具
	mcp.AddTool(server, &mcp.Tool{
		Name:        "mysql_commit_transaction",
		Description: "提交当前事务",
	}, h.handleCommitTransaction)

	// 回滚事务工具
	mcp.AddTool(server, &mcp.Tool{
		Name:        "mysql_rollback_transaction",
		Description: "回滚当前事务",
	}, h.handleRollbackTransaction)

	// 列出所有表工具
	mcp.AddTool(server, &mcp.Tool{
		Name:        "mysql_list_tables",
		Description: "列出数据库中的所有表",
	}, h.handleListTables)

	// 描述表结构工具
	mcp.AddTool(server, &mcp.Tool{
		Name:        "mysql_describe_table",
		Description: "描述表结构",
	}, h.handleDescribeTable)

	// 创建表工具
	mcp.AddTool(server, &mcp.Tool{
		Name:        "mysql_create_table",
		Description: "创建新表",
	}, h.handleCreateTable)

	// 删除表工具
	mcp.AddTool(server, &mcp.Tool{
		Name:        "mysql_drop_table",
		Description: "删除表",
	}, h.handleDropTable)

	// 创建索引工具
	mcp.AddTool(server, &mcp.Tool{
		Name:        "mysql_create_index",
		Description: "创建索引",
	}, h.handleCreateIndex)

	// 删除索引工具
	mcp.AddTool(server, &mcp.Tool{
		Name:        "mysql_drop_index",
		Description: "删除索引",
	}, h.handleDropIndex)

	// 列出索引工具
	mcp.AddTool(server, &mcp.Tool{
		Name:        "mysql_list_indexes",
		Description: "列出表的所有索引",
	}, h.handleListIndexes)

	// 数据库迁移工具
	mcp.AddTool(server, &mcp.Tool{
		Name:        "mysql_migrate",
		Description: "执行数据库迁移",
	}, h.handleMigrate)

	// 连接池状态工具
	mcp.AddTool(server, &mcp.Tool{
		Name:        "mysql_pool_status",
		Description: "获取数据库连接池状态",
	}, h.handlePoolStatus)

	// 新增：表字段管理工具
	mcp.AddTool(server, &mcp.Tool{
		Name:        "mysql_add_columns",
		Description: "为表添加多个字段",
	}, h.handleAddColumns)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "mysql_drop_columns",
		Description: "从表中删除多个字段",
	}, h.handleDropColumns)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "mysql_modify_columns",
		Description: "修改表的多个字段",
	}, h.handleModifyColumns)

	// 新增：表重命名工具
	mcp.AddTool(server, &mcp.Tool{
		Name:        "mysql_rename_table",
		Description: "重命名表",
	}, h.handleRenameTable)

	// 新增：表复制工具
	mcp.AddTool(server, &mcp.Tool{
		Name:        "mysql_copy_table",
		Description: "复制表结构和数据",
	}, h.handleCopyTable)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "mysql_copy_table_structure",
		Description: "仅复制表结构（不复制数据）",
	}, h.handleCopyTableStructure)
}

// 查询参数
type QueryParams struct {
	Query      string   `json:"query"`
	Parameters []string `json:"parameters,omitempty"`
}

// 执行参数
type ExecuteParams struct {
	SQL        string   `json:"sql"`
	Parameters []string `json:"parameters,omitempty"`
}

// 事务参数
type TransactionParams struct {
	TransactionID string `json:"transaction_id"`
}

// 表参数
type TableParams struct {
	TableName string `json:"table_name"`
}

// 创建表参数
type CreateTableParams struct {
	TableName string `json:"table_name"`
	Columns   string `json:"columns"`
}

// 创建索引参数
type CreateIndexParams struct {
	TableName string `json:"table_name"`
	IndexName string `json:"index_name"`
	Columns   string `json:"columns"`
	IndexType string `json:"index_type,omitempty"`
}

// 删除索引参数
type DropIndexParams struct {
	TableName string `json:"table_name"`
	IndexName string `json:"index_name"`
}

// 迁移参数
type MigrateParams struct {
	MigrationSQL string `json:"migration_sql"`
}

// 新增：表字段管理参数
type ColumnDefinition struct {
	Name         string `json:"name"`
	Type         string `json:"type"`
	Nullable     bool   `json:"nullable,omitempty"`
	DefaultValue string `json:"default_value,omitempty"`
	AfterColumn  string `json:"after_column,omitempty"`
}

type AddColumnsParams struct {
	TableName string             `json:"table_name"`
	Columns   []ColumnDefinition `json:"columns"`
}

type DropColumnsParams struct {
	TableName string   `json:"table_name"`
	Columns   []string `json:"columns"`
}

type ModifyColumnsParams struct {
	TableName string             `json:"table_name"`
	Columns   []ColumnDefinition `json:"columns"`
}

// 新增：表重命名参数
type RenameTableParams struct {
	OldTableName string `json:"old_table_name"`
	NewTableName string `json:"new_table_name"`
}

// 新增：表复制参数
type CopyTableParams struct {
	SourceTable      string `json:"source_table"`
	DestinationTable string `json:"destination_table"`
	CopyData         bool   `json:"copy_data,omitempty"`
}

// 事务管理器
type transactionManager struct {
	transactions map[string]*sql.Tx
	mu           sync.RWMutex
}

var (
	transactions = &transactionManager{
		transactions: make(map[string]*sql.Tx),
	}
)

// handleQuery 处理 SQL 查询
func (h *ToolHandler) handleQuery(ctx context.Context, request *mcp.CallToolRequest, params QueryParams) (*mcp.CallToolResult, any, error) {
	db, err := h.dbPool.GetConnection()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get database connection: %w", err)
	}

	// 转换参数为 interface{} 类型
	args := make([]interface{}, len(params.Parameters))
	for i, param := range params.Parameters {
		args[i] = param
	}

	rows, err := db.QueryContext(ctx, params.Query, args...)
	if err != nil {
		return nil, nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get columns: %w", err)
	}

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

	response := map[string]interface{}{
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

// handleExecute 处理 SQL 更新
func (h *ToolHandler) handleExecute(ctx context.Context, request *mcp.CallToolRequest, params ExecuteParams) (*mcp.CallToolResult, any, error) {
	db, err := h.dbPool.GetConnection()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get database connection: %w", err)
	}

	// 转换参数为 interface{} 类型
	args := make([]interface{}, len(params.Parameters))
	for i, param := range params.Parameters {
		args[i] = param
	}

	result, err := db.ExecContext(ctx, params.SQL, args...)
	if err != nil {
		return nil, nil, fmt.Errorf("execute failed: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	lastInsertID, _ := result.LastInsertId()

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

// handleBeginTransaction 处理开始事务
func (h *ToolHandler) handleBeginTransaction(ctx context.Context, request *mcp.CallToolRequest, params any) (*mcp.CallToolResult, any, error) {
	tx, err := h.dbPool.BeginTransaction()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to begin transaction: %w", err)
	}

	// 生成事务 ID
	txID := fmt.Sprintf("tx_%d", time.Now().UnixNano())
	transactions.mu.Lock()
	transactions.transactions[txID] = tx
	transactions.mu.Unlock()

	response := map[string]interface{}{
		"transaction_id": txID,
		"message":        "Transaction started successfully",
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: formatJSON(response)},
		},
	}, response, nil
}

// handleCommitTransaction 处理提交事务
func (h *ToolHandler) handleCommitTransaction(ctx context.Context, request *mcp.CallToolRequest, params TransactionParams) (*mcp.CallToolResult, any, error) {
	transactions.mu.Lock()
	tx, exists := transactions.transactions[params.TransactionID]
	if !exists {
		transactions.mu.Unlock()
		return nil, nil, fmt.Errorf("transaction not found: %s", params.TransactionID)
	}
	delete(transactions.transactions, params.TransactionID)
	transactions.mu.Unlock()

	if err := tx.Commit(); err != nil {
		return nil, nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	response := map[string]interface{}{
		"transaction_id": params.TransactionID,
		"message":        "Transaction committed successfully",
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: formatJSON(response)},
		},
	}, response, nil
}

// handleRollbackTransaction 处理回滚事务
func (h *ToolHandler) handleRollbackTransaction(ctx context.Context, request *mcp.CallToolRequest, params TransactionParams) (*mcp.CallToolResult, any, error) {
	transactions.mu.Lock()
	tx, exists := transactions.transactions[params.TransactionID]
	if !exists {
		transactions.mu.Unlock()
		return nil, nil, fmt.Errorf("transaction not found: %s", params.TransactionID)
	}
	delete(transactions.transactions, params.TransactionID)
	transactions.mu.Unlock()

	if err := tx.Rollback(); err != nil {
		return nil, nil, fmt.Errorf("failed to rollback transaction: %w", err)
	}

	response := map[string]interface{}{
		"transaction_id": params.TransactionID,
		"message":        "Transaction rolled back successfully",
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: formatJSON(response)},
		},
	}, response, nil
}

// handleListTables 处理列出所有表
func (h *ToolHandler) handleListTables(ctx context.Context, request *mcp.CallToolRequest, params any) (*mcp.CallToolResult, any, error) {
	db, err := h.dbPool.GetConnection()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get database connection: %w", err)
	}

	rows, err := db.QueryContext(ctx, "SHOW TABLES")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list tables: %w", err)
	}
	defer rows.Close()

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

	response := map[string]interface{}{
		"tables": tables,
		"count":  len(tables),
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: formatJSON(response)},
		},
	}, response, nil
}

// handleDescribeTable 处理描述表结构
func (h *ToolHandler) handleDescribeTable(ctx context.Context, request *mcp.CallToolRequest, params TableParams) (*mcp.CallToolResult, any, error) {
	db, err := h.dbPool.GetConnection()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get database connection: %w", err)
	}

	rows, err := db.QueryContext(ctx, "DESCRIBE "+params.TableName)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to describe table: %w", err)
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get columns: %w", err)
	}

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

	response := map[string]interface{}{
		"table_name": params.TableName,
		"columns":    results,
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: formatJSON(response)},
		},
	}, response, nil
}

// handleCreateTable 处理创建表
func (h *ToolHandler) handleCreateTable(ctx context.Context, request *mcp.CallToolRequest, params CreateTableParams) (*mcp.CallToolResult, any, error) {
	db, err := h.dbPool.GetConnection()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get database connection: %w", err)
	}

	sql := fmt.Sprintf("CREATE TABLE %s (%s)", params.TableName, params.Columns)
	_, err = db.ExecContext(ctx, sql)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create table: %w", err)
	}

	response := map[string]interface{}{
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

// handleDropTable 处理删除表
func (h *ToolHandler) handleDropTable(ctx context.Context, request *mcp.CallToolRequest, params TableParams) (*mcp.CallToolResult, any, error) {
	db, err := h.dbPool.GetConnection()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get database connection: %w", err)
	}

	sql := fmt.Sprintf("DROP TABLE %s", params.TableName)
	_, err = db.ExecContext(ctx, sql)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to drop table: %w", err)
	}

	response := map[string]interface{}{
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

// handleCreateIndex 处理创建索引
func (h *ToolHandler) handleCreateIndex(ctx context.Context, request *mcp.CallToolRequest, params CreateIndexParams) (*mcp.CallToolResult, any, error) {
	db, err := h.dbPool.GetConnection()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get database connection: %w", err)
	}

	var sql string
	if params.IndexType != "" {
		sql = fmt.Sprintf("CREATE %s INDEX %s ON %s (%s)", params.IndexType, params.IndexName, params.TableName, params.Columns)
	} else {
		sql = fmt.Sprintf("CREATE INDEX %s ON %s (%s)", params.IndexName, params.TableName, params.Columns)
	}

	_, err = db.ExecContext(ctx, sql)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create index: %w", err)
	}

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

// handleDropIndex 处理删除索引
func (h *ToolHandler) handleDropIndex(ctx context.Context, request *mcp.CallToolRequest, params DropIndexParams) (*mcp.CallToolResult, any, error) {
	db, err := h.dbPool.GetConnection()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get database connection: %w", err)
	}

	sql := fmt.Sprintf("DROP INDEX %s ON %s", params.IndexName, params.TableName)
	_, err = db.ExecContext(ctx, sql)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to drop index: %w", err)
	}

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

// handleListIndexes 处理列出索引
func (h *ToolHandler) handleListIndexes(ctx context.Context, request *mcp.CallToolRequest, params TableParams) (*mcp.CallToolResult, any, error) {
	db, err := h.dbPool.GetConnection()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get database connection: %w", err)
	}

	rows, err := db.QueryContext(ctx, "SHOW INDEX FROM "+params.TableName)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list indexes: %w", err)
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get columns: %w", err)
	}

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

// handleMigrate 处理数据库迁移
func (h *ToolHandler) handleMigrate(ctx context.Context, request *mcp.CallToolRequest, params MigrateParams) (*mcp.CallToolResult, any, error) {
	db, err := h.dbPool.GetConnection()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get database connection: %w", err)
	}

	// 开始事务
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to begin transaction: %w", err)
	}

	// 执行迁移 SQL
	_, err = tx.ExecContext(ctx, params.MigrationSQL)
	if err != nil {
		tx.Rollback()
		return nil, nil, fmt.Errorf("failed to execute migration: %w", err)
	}

	// 提交事务
	if err := tx.Commit(); err != nil {
		return nil, nil, fmt.Errorf("failed to commit migration: %w", err)
	}

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

// handlePoolStatus 处理连接池状态
func (h *ToolHandler) handlePoolStatus(ctx context.Context, request *mcp.CallToolRequest, params any) (*mcp.CallToolResult, any, error) {
	stats := h.dbPool.Stats()
	config := h.dbPool.GetConfig()

	response := map[string]interface{}{
		"pool_config": map[string]interface{}{
			"max_connections":     config.MaxConns,
			"idle_connections":    config.IdleConns,
			"connection_lifetime": config.ConnLifetime.String(),
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

// handleAddColumns 处理添加多个字段
func (h *ToolHandler) handleAddColumns(ctx context.Context, request *mcp.CallToolRequest, params AddColumnsParams) (*mcp.CallToolResult, any, error) {
	db, err := h.dbPool.GetConnection()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get database connection: %w", err)
	}

	// 开始事务
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to begin transaction: %w", err)
	}

	var executedSQLs []string
	var errors []string

	// 逐个添加字段
	for _, column := range params.Columns {
		// 构建字段定义
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

		// 执行 ALTER TABLE
		sql := fmt.Sprintf("ALTER TABLE `%s` ADD COLUMN %s", params.TableName, columnDef)
		_, err := tx.ExecContext(ctx, sql)

		if err != nil {
			errors = append(errors, fmt.Sprintf("添加字段 %s 失败: %v", column.Name, err))
		} else {
			executedSQLs = append(executedSQLs, sql)
		}
	}

	// 如果有错误，回滚事务
	if len(errors) > 0 {
		tx.Rollback()
		return nil, nil, fmt.Errorf("添加字段失败: %v", errors)
	}

	// 提交事务
	if err := tx.Commit(); err != nil {
		return nil, nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

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

// handleDropColumns 处理删除多个字段
func (h *ToolHandler) handleDropColumns(ctx context.Context, request *mcp.CallToolRequest, params DropColumnsParams) (*mcp.CallToolResult, any, error) {
	db, err := h.dbPool.GetConnection()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get database connection: %w", err)
	}

	// 开始事务
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to begin transaction: %w", err)
	}

	var executedSQLs []string
	var errors []string

	// 逐个删除字段
	for _, column := range params.Columns {
		sql := fmt.Sprintf("ALTER TABLE `%s` DROP COLUMN `%s`", params.TableName, column)
		_, err := tx.ExecContext(ctx, sql)

		if err != nil {
			errors = append(errors, fmt.Sprintf("删除字段 %s 失败: %v", column, err))
		} else {
			executedSQLs = append(executedSQLs, sql)
		}
	}

	// 如果有错误，回滚事务
	if len(errors) > 0 {
		tx.Rollback()
		return nil, nil, fmt.Errorf("删除字段失败: %v", errors)
	}

	// 提交事务
	if err := tx.Commit(); err != nil {
		return nil, nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

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

// handleModifyColumns 处理修改多个字段
func (h *ToolHandler) handleModifyColumns(ctx context.Context, request *mcp.CallToolRequest, params ModifyColumnsParams) (*mcp.CallToolResult, any, error) {
	db, err := h.dbPool.GetConnection()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get database connection: %w", err)
	}

	// 开始事务
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to begin transaction: %w", err)
	}

	var executedSQLs []string
	var errors []string

	// 逐个修改字段
	for _, column := range params.Columns {
		// 构建字段定义
		columnDef := fmt.Sprintf("`%s` %s", column.Name, column.Type)

		if !column.Nullable {
			columnDef += " NOT NULL"
		}

		if column.DefaultValue != "" {
			columnDef += fmt.Sprintf(" DEFAULT '%s'", column.DefaultValue)
		}

		// 执行 ALTER TABLE
		sql := fmt.Sprintf("ALTER TABLE `%s` MODIFY COLUMN %s", params.TableName, columnDef)
		_, err := tx.ExecContext(ctx, sql)

		if err != nil {
			errors = append(errors, fmt.Sprintf("修改字段 %s 失败: %v", column.Name, err))
		} else {
			executedSQLs = append(executedSQLs, sql)
		}
	}

	// 如果有错误，回滚事务
	if len(errors) > 0 {
		tx.Rollback()
		return nil, nil, fmt.Errorf("修改字段失败: %v", errors)
	}

	// 提交事务
	if err := tx.Commit(); err != nil {
		return nil, nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

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

// handleRenameTable 处理重命名表
func (h *ToolHandler) handleRenameTable(ctx context.Context, request *mcp.CallToolRequest, params RenameTableParams) (*mcp.CallToolResult, any, error) {
	db, err := h.dbPool.GetConnection()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get database connection: %w", err)
	}

	sql := fmt.Sprintf("RENAME TABLE `%s` TO `%s`", params.OldTableName, params.NewTableName)
	_, err = db.ExecContext(ctx, sql)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to rename table: %w", err)
	}

	response := map[string]interface{}{
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

// handleCopyTable 处理复制表结构和数据
func (h *ToolHandler) handleCopyTable(ctx context.Context, request *mcp.CallToolRequest, params CopyTableParams) (*mcp.CallToolResult, any, error) {
	db, err := h.dbPool.GetConnection()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get database connection: %w", err)
	}

	// 开始事务
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to begin transaction: %w", err)
	}

	// 1. 复制表结构
	createTableSQL := fmt.Sprintf("CREATE TABLE `%s` LIKE `%s`", params.DestinationTable, params.SourceTable)
	_, err = tx.ExecContext(ctx, createTableSQL)
	if err != nil {
		tx.Rollback()
		return nil, nil, fmt.Errorf("failed to copy table structure: %w", err)
	}

	// 2. 如果需要复制数据
	if params.CopyData {
		copyDataSQL := fmt.Sprintf("INSERT INTO `%s` SELECT * FROM `%s`", params.DestinationTable, params.SourceTable)
		_, err = tx.ExecContext(ctx, copyDataSQL)
		if err != nil {
			tx.Rollback()
			return nil, nil, fmt.Errorf("failed to copy table data: %w", err)
		}
	}

	// 提交事务
	if err := tx.Commit(); err != nil {
		return nil, nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	response := map[string]interface{}{
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

// handleCopyTableStructure 处理仅复制表结构
func (h *ToolHandler) handleCopyTableStructure(ctx context.Context, request *mcp.CallToolRequest, params CopyTableParams) (*mcp.CallToolResult, any, error) {
	// 设置不复制数据
	params.CopyData = false
	return h.handleCopyTable(ctx, request, params)
}

// formatJSON 格式化 JSON 输出
func formatJSON(data interface{}) string {
	jsonBytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Sprintf("{\"error\": \"Failed to marshal JSON: %v\"}", err)
	}
	return string(jsonBytes)
}
