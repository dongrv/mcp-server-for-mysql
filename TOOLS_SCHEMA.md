# MySQL MCP Tools Schema Documentation

## Overview

This document describes the JSON Schema for all MySQL MCP tools. These schemas help AI models understand how to properly call each tool with the correct parameters.

## Tool Categories

### Query & Execution

- **mysql_query**: 执行 SQL 查询语句
- **mysql_execute**: 执行 SQL 更新语句（INSERT, UPDATE, DELETE）

### Transaction Management

- **mysql_begin_transaction**: 开始一个新的事务
- **mysql_commit_transaction**: 提交当前事务
- **mysql_rollback_transaction**: 回滚当前事务

### Table Operations

- **mysql_list_tables**: 列出数据库中的所有表
- **mysql_describe_table**: 描述表结构
- **mysql_create_table**: 创建新表
- **mysql_drop_table**: 删除表
- **mysql_rename_table**: 重命名表
- **mysql_copy_table**: 复制表结构和数据
- **mysql_copy_table_structure**: 仅复制表结构（不复制数据）

### Column Operations

- **mysql_add_columns**: 为表添加多个字段
- **mysql_drop_columns**: 从表中删除多个字段
- **mysql_modify_columns**: 修改表的多个字段

### Index Operations

- **mysql_create_index**: 创建索引
- **mysql_drop_index**: 删除索引
- **mysql_list_indexes**: 列出表的所有索引

### Database Management

- **mysql_migrate**: 执行数据库迁移
- **mysql_pool_status**: 获取数据库连接池状态

## Detailed Schema Reference

### Query & Execution

#### mysql_query

**Description**: 执行 SQL 查询语句

**Input Schema**:

```json
{
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
  "required": [
    "query"
  ],
  "additionalProperties": false
}
```

**Required Fields**:

- `query`

**Properties**:

- `query`:
  - **Description**: 要执行的SQL查询语句
  - **Type**: `string`

- `parameters`:
  - **Description**: 查询参数（可选）
  - **Type**: `array`

---

#### mysql_execute

**Description**: 执行 SQL 更新语句（INSERT, UPDATE, DELETE）

**Input Schema**:

```json
{
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
  "required": [
    "query"
  ],
  "additionalProperties": false
}
```

**Required Fields**:

- `query`

**Properties**:

- `query`:
  - **Description**: 要执行的SQL更新语句
  - **Type**: `string`

- `parameters`:
  - **Description**: 查询参数（可选）
  - **Type**: `array`

---

### Transaction Management

#### mysql_begin_transaction

**Description**: 开始一个新的事务

**Input Schema**:

```json
{
  "type": "object",
  "properties": {},
  "additionalProperties": false
}
```

---

#### mysql_commit_transaction

**Description**: 提交当前事务

**Input Schema**:

```json
{
  "type": "object",
  "properties": {},
  "additionalProperties": false
}
```

---

#### mysql_rollback_transaction

**Description**: 回滚当前事务

**Input Schema**:

```json
{
  "type": "object",
  "properties": {},
  "additionalProperties": false
}
```

---

### Table Operations

#### mysql_list_tables

**Description**: 列出数据库中的所有表

**Input Schema**:

```json
{
  "type": "object",
  "properties": {},
  "additionalProperties": false
}
```

---

#### mysql_describe_table

**Description**: 描述表结构

**Input Schema**:

```json
{
  "type": "object",
  "properties": {
    "table_name": {
      "type": "string",
      "description": "要描述的表名"
    }
  },
  "required": [
    "table_name"
  ],
  "additionalProperties": false
}
```

**Required Fields**:

- `table_name`

**Properties**:

- `table_name`:
  - **Description**: 要描述的表名
  - **Type**: `string`

---

#### mysql_create_table

**Description**: 创建新表

**Input Schema**:

```json
{
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
  "required": [
    "table_name",
    "columns"
  ],
  "additionalProperties": false
}
```

**Required Fields**:

- `table_name`
- `columns`

**Properties**:

- `table_name`:
  - **Description**: 要创建的表名
  - **Type**: `string`

- `columns`:
  - **Description**: 列定义，例如：id INT PRIMARY KEY AUTO_INCREMENT, name VARCHAR(100) NOT NULL
  - **Type**: `string`

---

#### mysql_drop_table

**Description**: 删除表

**Input Schema**:

```json
{
  "type": "object",
  "properties": {
    "table_name": {
      "type": "string",
      "description": "要删除的表名"
    }
  },
  "required": [
    "table_name"
  ],
  "additionalProperties": false
}
```

**Required Fields**:

- `table_name`

**Properties**:

- `table_name`:
  - **Description**: 要删除的表名
  - **Type**: `string`

---

#### mysql_rename_table

**Description**: 重命名表

**Input Schema**:

```json
{
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
  "required": [
    "old_table_name",
    "new_table_name"
  ],
  "additionalProperties": false
}
```

**Required Fields**:

- `old_table_name`
- `new_table_name`

**Properties**:

- `old_table_name`:
  - **Description**: 原表名
  - **Type**: `string`

- `new_table_name`:
  - **Description**: 新表名
  - **Type**: `string`

---

#### mysql_copy_table

**Description**: 复制表结构和数据

**Input Schema**:

```json
{
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
  "required": [
    "source_table",
    "target_table"
  ],
  "additionalProperties": false
}
```

**Required Fields**:

- `source_table`
- `target_table`

**Properties**:

- `source_table`:
  - **Description**: 源表名
  - **Type**: `string`

- `target_table`:
  - **Description**: 目标表名
  - **Type**: `string`

---

#### mysql_copy_table_structure

**Description**: 仅复制表结构（不复制数据）

**Input Schema**:

```json
{
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
  "required": [
    "source_table",
    "target_table"
  ],
  "additionalProperties": false
}
```

**Required Fields**:

- `source_table`
- `target_table`

**Properties**:

- `source_table`:
  - **Description**: 源表名
  - **Type**: `string`

- `target_table`:
  - **Description**: 目标表名
  - **Type**: `string`

---

### Column Operations

#### mysql_add_columns

**Description**: 为表添加多个字段

**Input Schema**:

```json
{
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
        "required": [
          "name",
          "type"
        ],
        "additionalProperties": false
      },
      "description": "要添加的列定义数组"
    }
  },
  "required": [
    "table_name",
    "columns"
  ],
  "additionalProperties": false
}
```

**Required Fields**:

- `table_name`
- `columns`

**Properties**:

- `table_name`:
  - **Description**: 表名
  - **Type**: `string`

- `columns`:
  - **Description**: 要添加的列定义数组
  - **Type**: `array`

---

#### mysql_drop_columns

**Description**: 从表中删除多个字段

**Input Schema**:

```json
{
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
  "required": [
    "table_name",
    "columns"
  ],
  "additionalProperties": false
}
```

**Required Fields**:

- `table_name`
- `columns`

**Properties**:

- `table_name`:
  - **Description**: 表名
  - **Type**: `string`

- `columns`:
  - **Description**: 要删除的列名数组
  - **Type**: `array`

---

#### mysql_modify_columns

**Description**: 修改表的多个字段

**Input Schema**:

```json
{
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
        "required": [
          "name",
          "type"
        ],
        "additionalProperties": false
      },
      "description": "要修改的列定义数组"
    }
  },
  "required": [
    "table_name",
    "columns"
  ],
  "additionalProperties": false
}
```

**Required Fields**:

- `table_name`
- `columns`

**Properties**:

- `table_name`:
  - **Description**: 表名
  - **Type**: `string`

- `columns`:
  - **Description**: 要修改的列定义数组
  - **Type**: `array`

---

### Index Operations

#### mysql_create_index

**Description**: 创建索引

**Input Schema**:

```json
{
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
      "enum": [
        "INDEX",
        "UNIQUE",
        "FULLTEXT",
        "SPATIAL"
      ],
      "description": "索引类型，默认为INDEX"
    }
  },
  "required": [
    "table_name",
    "index_name",
    "columns"
  ],
  "additionalProperties": false
}
```

**Required Fields**:

- `table_name`
- `index_name`
- `columns`

**Properties**:

- `table_name`:
  - **Description**: 表名
  - **Type**: `string`

- `index_name`:
  - **Description**: 索引名称
  - **Type**: `string`

- `columns`:
  - **Description**: 要创建索引的列名数组
  - **Type**: `array`

- `index_type`:
  - **Description**: 索引类型，默认为INDEX
  - **Type**: `string`
  - **Allowed Values**: `INDEX`, `UNIQUE`, `FULLTEXT`, `SPATIAL`

---

#### mysql_drop_index

**Description**: 删除索引

**Input Schema**:

```json
{
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
  "required": [
    "table_name",
    "index_name"
  ],
  "additionalProperties": false
}
```

**Required Fields**:

- `table_name`
- `index_name`

**Properties**:

- `table_name`:
  - **Description**: 表名
  - **Type**: `string`

- `index_name`:
  - **Description**: 要删除的索引名称
  - **Type**: `string`

---

#### mysql_list_indexes

**Description**: 列出表的所有索引

**Input Schema**:

```json
{
  "type": "object",
  "properties": {
    "table_name": {
      "type": "string",
      "description": "表名"
    }
  },
  "required": [
    "table_name"
  ],
  "additionalProperties": false
}
```

**Required Fields**:

- `table_name`

**Properties**:

- `table_name`:
  - **Description**: 表名
  - **Type**: `string`

---

### Database Management

#### mysql_migrate

**Description**: 执行数据库迁移

**Input Schema**:

```json
{
  "type": "object",
  "properties": {
    "migration_sql": {
      "type": "string",
      "description": "迁移SQL语句"
    }
  },
  "required": [
    "migration_sql"
  ],
  "additionalProperties": false
}
```

**Required Fields**:

- `migration_sql`

**Properties**:

- `migration_sql`:
  - **Description**: 迁移SQL语句
  - **Type**: `string`

---

#### mysql_pool_status

**Description**: 获取数据库连接池状态

**Input Schema**:

```json
{
  "type": "object",
  "properties": {},
  "additionalProperties": false
}
```

---

## Usage Examples

### Example 1: Query Data

```json
{
  "query": "SELECT * FROM users WHERE status = ?",
  "parameters": ["active"]
}
```

### Example 2: Create Table

```json
{
  "table_name": "products",
  "columns": "id INT PRIMARY KEY AUTO_INCREMENT, name VARCHAR(100) NOT NULL, price DECIMAL(10,2) NOT NULL, created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP"
}
```

### Example 3: Add Columns

```json
{
  "table_name": "users",
  "columns": [
    {
      "name": "email",
      "type": "VARCHAR(255)",
      "nullable": false,
      "default_value": ""
    },
    {
      "name": "phone",
      "type": "VARCHAR(20)",
      "nullable": true,
      "after_column": "email"
    }
  ]
}
```

### Example 4: Create Index

```json
{
  "table_name": "orders",
  "index_name": "idx_customer_date",
  "columns": ["customer_id", "order_date"],
  "index_type": "INDEX"
}
```

## Best Practices for AI Models

1. **Always check the schema**: Before calling a tool, review the schema to understand required parameters and their types.

2. **Use appropriate parameter types**: Ensure parameters match the expected types (string, array, boolean, etc.).

3. **Handle optional parameters**: Some tools have optional parameters - only include them when needed.

4. **Validate input**: Use the schema to validate your input before making the tool call.

5. **Error handling**: Be prepared to handle errors gracefully when tool calls fail.

6. **Transaction management**: Use transaction tools (`mysql_begin_transaction`, `mysql_commit_transaction`, `mysql_rollback_transaction`) for operations that need to be atomic.

7. **Batch operations**: Use tools like `mysql_add_columns`, `mysql_drop_columns`, and `mysql_modify_columns` for batch column operations instead of making multiple individual calls.

## Schema Validation

The JSON Schema