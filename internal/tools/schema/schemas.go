// Package schema provides JSON Schema definitions for all MCP tools.
package schema

import (
	"encoding/json"
	"fmt"
)

// ToolSchema represents the JSON Schema for a tool.
type ToolSchema struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"inputSchema"`
}

// GetToolSchemas returns JSON Schema definitions for all MCP tools.
func GetToolSchemas() map[string]ToolSchema {
	return map[string]ToolSchema{
		"mysql_query": {
			Name:        "mysql_query",
			Description: "执行 SQL 查询语句",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"query": {
						"type": "string",
						"description": "要执行的SQL查询语句"
					},
					"parameters": {
						"type": "array",
						"items": {
							"type": "string"
						},
						"description": "查询参数（可选）"
					}
				},
				"required": ["query"],
				"additionalProperties": false
			}`),
		},
		"mysql_execute": {
			Name:        "mysql_execute",
			Description: "执行 SQL 更新语句（INSERT, UPDATE, DELETE）",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"query": {
						"type": "string",
						"description": "要执行的SQL更新语句"
					},
					"parameters": {
						"type": "array",
						"items": {
							"type": "string"
						},
						"description": "查询参数（可选）"
					}
				},
				"required": ["query"],
				"additionalProperties": false
			}`),
		},
		"mysql_begin_transaction": {
			Name:        "mysql_begin_transaction",
			Description: "开始一个新的事务",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {},
				"additionalProperties": false
			}`),
		},
		"mysql_commit_transaction": {
			Name:        "mysql_commit_transaction",
			Description: "提交当前事务",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"transaction_id": {
						"type": "string",
						"description": "事务ID"
					}
				},
				"required": ["transaction_id"],
				"additionalProperties": false
			}`),
		},
		"mysql_rollback_transaction": {
			Name:        "mysql_rollback_transaction",
			Description: "回滚当前事务",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"transaction_id": {
						"type": "string",
						"description": "事务ID"
					}
				},
				"required": ["transaction_id"],
				"additionalProperties": false
			}`),
		},
		"mysql_list_tables": {
			Name:        "mysql_list_tables",
			Description: "列出数据库中的所有表",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {},
				"additionalProperties": false
			}`),
		},
		"mysql_describe_table": {
			Name:        "mysql_describe_table",
			Description: "描述表结构",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"table_name": {
						"type": "string",
						"description": "要描述的表名"
					}
				},
				"required": ["table_name"],
				"additionalProperties": false
			}`),
		},
		"mysql_create_table": {
			Name:        "mysql_create_table",
			Description: "创建新表",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"table_name": {
						"type": "string",
						"description": "要创建的表名"
					},
					"columns": {
						"type": "string",
						"description": "列定义，例如：id INT PRIMARY KEY AUTO_INCREMENT, name VARCHAR(100) NOT NULL"
					}
				},
				"required": ["table_name", "columns"],
				"additionalProperties": false
			}`),
		},
		"mysql_drop_table": {
			Name:        "mysql_drop_table",
			Description: "删除表",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"table_name": {
						"type": "string",
						"description": "要删除的表名"
					}
				},
				"required": ["table_name"],
				"additionalProperties": false
			}`),
		},
		"mysql_create_index": {
			Name:        "mysql_create_index",
			Description: "创建索引",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"table_name": {
						"type": "string",
						"description": "表名"
					},
					"index_name": {
						"type": "string",
						"description": "索引名称"
					},
					"columns": {
						"type": "array",
						"items": {
							"type": "string"
						},
						"description": "要创建索引的列名数组"
					},
					"index_type": {
						"type": "string",
						"enum": ["INDEX", "UNIQUE", "FULLTEXT", "SPATIAL"],
						"description": "索引类型，默认为INDEX"
					}
				},
				"required": ["table_name", "index_name", "columns"],
				"additionalProperties": false
			}`),
		},
		"mysql_drop_index": {
			Name:        "mysql_drop_index",
			Description: "删除索引",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"table_name": {
						"type": "string",
						"description": "表名"
					},
					"index_name": {
						"type": "string",
						"description": "要删除的索引名称"
					}
				},
				"required": ["table_name", "index_name"],
				"additionalProperties": false
			}`),
		},
		"mysql_list_indexes": {
			Name:        "mysql_list_indexes",
			Description: "列出表的所有索引",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"table_name": {
						"type": "string",
						"description": "表名"
					}
				},
				"required": ["table_name"],
				"additionalProperties": false
			}`),
		},
		"mysql_migrate": {
			Name:        "mysql_migrate",
			Description: "执行数据库迁移",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"migration_sql": {
						"type": "string",
						"description": "迁移SQL语句"
					}
				},
				"required": ["migration_sql"],
				"additionalProperties": false
			}`),
		},
		"mysql_pool_status": {
			Name:        "mysql_pool_status",
			Description: "获取数据库连接池状态",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {},
				"additionalProperties": false
			}`),
		},
		"mysql_add_columns": {
			Name:        "mysql_add_columns",
			Description: "为表添加多个字段",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"table_name": {
						"type": "string",
						"description": "表名"
					},
					"columns": {
						"type": "array",
						"items": {
							"type": "object",
							"properties": {
								"name": {
									"type": "string",
									"description": "列名"
								},
								"type": {
									"type": "string",
									"description": "列类型，例如：VARCHAR(255), INT, DATETIME"
								},
								"nullable": {
									"type": "boolean",
									"description": "是否允许NULL，默认为true"
								},
								"default_value": {
									"type": "string",
									"description": "默认值"
								},
								"after_column": {
									"type": "string",
									"description": "在哪个列之后添加"
								}
							},
							"required": ["name", "type"],
							"additionalProperties": false
						},
						"description": "要添加的列定义数组"
					}
				},
				"required": ["table_name", "columns"],
				"additionalProperties": false
			}`),
		},
		"mysql_drop_columns": {
			Name:        "mysql_drop_columns",
			Description: "从表中删除多个字段",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"table_name": {
						"type": "string",
						"description": "表名"
					},
					"columns": {
						"type": "array",
						"items": {
							"type": "string"
						},
						"description": "要删除的列名数组"
					}
				},
				"required": ["table_name", "columns"],
				"additionalProperties": false
			}`),
		},
		"mysql_modify_columns": {
			Name:        "mysql_modify_columns",
			Description: "修改表的多个字段",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"table_name": {
						"type": "string",
						"description": "表名"
					},
					"columns": {
						"type": "array",
						"items": {
							"type": "object",
							"properties": {
								"name": {
									"type": "string",
									"description": "列名"
								},
								"type": {
									"type": "string",
									"description": "列类型，例如：VARCHAR(255), INT, DATETIME"
								},
								"nullable": {
									"type": "boolean",
									"description": "是否允许NULL，默认为true"
								},
								"default_value": {
									"type": "string",
									"description": "默认值"
								}
							},
							"required": ["name", "type"],
							"additionalProperties": false
						},
						"description": "要修改的列定义数组"
					}
				},
				"required": ["table_name", "columns"],
				"additionalProperties": false
			}`),
		},
		"mysql_rename_table": {
			Name:        "mysql_rename_table",
			Description: "重命名表",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"old_table_name": {
						"type": "string",
						"description": "原表名"
					},
					"new_table_name": {
						"type": "string",
						"description": "新表名"
					}
				},
				"required": ["old_table_name", "new_table_name"],
				"additionalProperties": false
			}`),
		},
		"mysql_copy_table": {
			Name:        "mysql_copy_table",
			Description: "复制表结构和数据",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"source_table": {
						"type": "string",
						"description": "源表名"
					},
					"target_table": {
						"type": "string",
						"description": "目标表名"
					}
				},
				"required": ["source_table", "target_table"],
				"additionalProperties": false
			}`),
		},
		"mysql_copy_table_structure": {
			Name:        "mysql_copy_table_structure",
			Description: "仅复制表结构（不复制数据）",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"source_table": {
						"type": "string",
						"description": "源表名"
					},
					"target_table": {
						"type": "string",
						"description": "目标表名"
					}
				},
				"required": ["source_table", "target_table"],
				"additionalProperties": false
			}`),
		},
	}
}

// GetToolSchema returns the JSON Schema for a specific tool.
func GetToolSchema(toolName string) (ToolSchema, error) {
	schemas := GetToolSchemas()
	schema, exists := schemas[toolName]
	if !exists {
		return ToolSchema{}, fmt.Errorf("tool schema not found: %s", toolName)
	}
	return schema, nil
}

// GetToolNames returns a list of all available tool names.
func GetToolNames() []string {
	schemas := GetToolSchemas()
	names := make([]string, 0, len(schemas))
	for name := range schemas {
		names = append(names, name)
	}
	return names
}

// GetToolDescriptions returns a map of tool names to their descriptions.
func GetToolDescriptions() map[string]string {
	schemas := GetToolSchemas()
	descriptions := make(map[string]string, len(schemas))
	for name, schema := range schemas {
		descriptions[name] = schema.Description
	}
	return descriptions
}
