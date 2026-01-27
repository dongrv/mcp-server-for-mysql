# MySQL MCP Server 新功能使用示例

本文档提供 MySQL MCP Server 新增功能的使用示例，包括表字段管理、表重命名和表复制功能。

## 目录
1. [表字段管理](#表字段管理)
   - [添加多个字段](#添加多个字段)
   - [删除多个字段](#删除多个字段)
   - [修改多个字段](#修改多个字段)
2. [表重命名](#表重命名)
3. [表复制](#表复制)
   - [复制表结构和数据](#复制表结构和数据)
   - [仅复制表结构](#仅复制表结构)
4. [完整工作流示例](#完整工作流示例)
5. [错误处理](#错误处理)
6. [最佳实践](#最佳实践)

## 表字段管理

### 添加多个字段

**场景**: 为 `users` 表添加 `age`、`address` 和 `phone` 字段

**JSON 请求示例**:
```json
{
  "name": "mysql_add_columns",
  "arguments": {
    "table_name": "users",
    "columns": [
      {
        "name": "age",
        "type": "INT",
        "nullable": true,
        "default_value": "0",
        "after_column": "email"
      },
      {
        "name": "address",
        "type": "VARCHAR(255)",
        "nullable": true,
        "default_value": ""
      },
      {
        "name": "phone",
        "type": "VARCHAR(20)",
        "nullable": true,
        "default_value": ""
      }
    ]
  }
}
```

**参数说明**:
- `table_name`: 目标表名
- `columns`: 要添加的字段数组
  - `name`: 字段名
  - `type`: 字段类型（如 INT, VARCHAR(255), DATETIME 等）
  - `nullable`: 是否允许 NULL 值
  - `default_value`: 默认值
  - `after_column`: 字段位置（可选，指定在哪个字段之后）

**响应示例**:
```json
{
  "table_name": "users",
  "added_columns": 3,
  "executed_sqls": [
    "ALTER TABLE `users` ADD COLUMN `age` INT DEFAULT '0' AFTER `email`",
    "ALTER TABLE `users` ADD COLUMN `address` VARCHAR(255) DEFAULT ''",
    "ALTER TABLE `users` ADD COLUMN `phone` VARCHAR(20) DEFAULT ''"
  ],
  "message": "成功添加 3 个字段"
}
```

### 删除多个字段

**场景**: 从 `users` 表中删除 `temp_field1` 和 `temp_field2` 字段

**JSON 请求示例**:
```json
{
  "name": "mysql_drop_columns",
  "arguments": {
    "table_name": "users",
    "columns": ["temp_field1", "temp_field2"]
  }
}
```

**响应示例**:
```json
{
  "table_name": "users",
  "dropped_columns": 2,
  "executed_sqls": [
    "ALTER TABLE `users` DROP COLUMN `temp_field1`",
    "ALTER TABLE `users` DROP COLUMN `temp_field2`"
  ],
  "message": "成功删除 2 个字段"
}
```

### 修改多个字段

**场景**: 修改 `users` 表的 `age` 和 `address` 字段定义

**JSON 请求示例**:
```json
{
  "name": "mysql_modify_columns",
  "arguments": {
    "table_name": "users",
    "columns": [
      {
        "name": "age",
        "type": "INT UNSIGNED",
        "nullable": false,
        "default_value": "18"
      },
      {
        "name": "address",
        "type": "VARCHAR(500)",
        "nullable": true,
        "default_value": ""
      }
    ]
  }
}
```

**响应示例**:
```json
{
  "table_name": "users",
  "modified_columns": 2,
  "executed_sqls": [
    "ALTER TABLE `users` MODIFY COLUMN `age` INT UNSIGNED NOT NULL DEFAULT '18'",
    "ALTER TABLE `users` MODIFY COLUMN `address` VARCHAR(500) DEFAULT ''"
  ],
  "message": "成功修改 2 个字段"
}
```

## 表重命名

**场景**: 将表 `old_users` 重命名为 `new_users`

**JSON 请求示例**:
```json
{
  "name": "mysql_rename_table",
  "arguments": {
    "old_table_name": "old_users",
    "new_table_name": "new_users"
  }
}
```

**响应示例**:
```json
{
  "old_table_name": "old_users",
  "new_table_name": "new_users",
  "sql": "RENAME TABLE `old_users` TO `new_users`",
  "message": "表重命名成功"
}
```

**注意事项**:
- 重命名操作是原子的
- 所有引用该表的外键、视图和存储过程需要相应更新
- 需要具有 ALTER 和 DROP 权限

## 表复制

### 复制表结构和数据

**场景**: 将 `users` 表完整复制到 `users_backup`（包括结构和数据）

**JSON 请求示例**:
```json
{
  "name": "mysql_copy_table",
  "arguments": {
    "source_table": "users",
    "destination_table": "users_backup",
    "copy_data": true
  }
}
```

**响应示例**:
```json
{
  "source_table": "users",
  "destination_table": "users_backup",
  "copy_data": true,
  "message": "表复制成功"
}
```

### 仅复制表结构

**场景**: 仅复制 `users` 表的结构到 `users_template`（不复制数据）

**方法 1**: 使用 `mysql_copy_table_structure`
```json
{
  "name": "mysql_copy_table_structure",
  "arguments": {
    "source_table": "users",
    "destination_table": "users_template"
  }
}
```

**方法 2**: 使用 `mysql_copy_table` 并设置 `copy_data: false`
```json
{
  "name": "mysql_copy_table",
  "arguments": {
    "source_table": "users",
    "destination_table": "users_template",
    "copy_data": false
  }
}
```

**响应示例**:
```json
{
  "source_table": "users",
  "destination_table": "users_template",
  "copy_data": false,
  "message": "表复制成功"
}
```

## 完整工作流示例

### 场景: 数据库表重构

**需求**: 
1. 为 `products` 表添加新字段
2. 修改现有字段定义
3. 创建备份表
4. 重命名原表
5. 创建新版本表

**步骤 1**: 创建备份
```json
{
  "name": "mysql_copy_table",
  "arguments": {
    "source_table": "products",
    "destination_table": "products_backup_202401",
    "copy_data": true
  }
}
```

**步骤 2**: 添加新字段到原表
```json
{
  "name": "mysql_add_columns",
  "arguments": {
    "table_name": "products",
    "columns": [
      {
        "name": "sku_code",
        "type": "VARCHAR(50)",
        "nullable": false,
        "default_value": "",
        "after_column": "id"
      },
      {
        "name": "weight",
        "type": "DECIMAL(10,2)",
        "nullable": true,
        "default_value": "0.00"
      }
    ]
  }
}
```

**步骤 3**: 修改现有字段
```json
{
  "name": "mysql_modify_columns",
  "arguments": {
    "table_name": "products",
    "columns": [
      {
        "name": "price",
        "type": "DECIMAL(12,2)",
        "nullable": false,
        "default_value": "0.00"
      }
    ]
  }
}
```

**步骤 4**: 重命名原表
```json
{
  "name": "mysql_rename_table",
  "arguments": {
    "old_table_name": "products",
    "new_table_name": "products_old"
  }
}
```

**步骤 5**: 创建新版本表（从备份恢复结构）
```json
{
  "name": "mysql_copy_table_structure",
  "arguments": {
    "source_table": "products_backup_202401",
    "destination_table": "products"
  }
}
```

## 错误处理

### 常见错误及解决方案

1. **字段已存在**
   ```
   错误: 添加字段 age 失败: Error 1060: Duplicate column name 'age'
   ```
   **解决方案**: 检查字段是否已存在，或使用 `mysql_modify_columns` 修改现有字段

2. **字段不存在**
   ```
   错误: 删除字段 temp_field 失败: Error 1091: Can't DROP 'temp_field'; check that column/key exists
   ```
   **解决方案**: 确认字段名是否正确，或先检查表结构

3. **表不存在**
   ```
   错误: 重命名表失败: Error 1146: Table 'database.nonexistent_table' doesn't exist
   ```
   **解决方案**: 使用 `mysql_list_tables` 确认表名

4. **权限不足**
   ```
   错误: 修改字段失败: Error 1142: ALTER command denied to user
   ```
   **解决方案**: 检查数据库用户权限

### 事务安全
所有字段管理操作都在事务中执行，确保原子性：
- 添加多个字段：要么全部成功，要么全部失败
- 删除多个字段：要么全部成功，要么全部失败
- 修改多个字段：要么全部成功，要么全部失败

## 最佳实践

### 1. 备份优先
在执行任何表结构修改前，先创建备份：
```json
{
  "name": "mysql_copy_table",
  "arguments": {
    "source_table": "target_table",
    "destination_table": "target_table_backup",
    "copy_data": true
  }
}
```

### 2. 分步验证
1. 先检查表结构：`mysql_describe_table`
2. 小范围测试：先在一个字段上测试
3. 验证结果：修改后再次检查表结构

### 3. 生产环境操作
- 在业务低峰期执行
- 确保有完整的回滚方案
- 监控数据库性能影响
- 记录所有操作日志

### 4. 字段命名规范
- 使用有意义的字段名
- 保持命名一致性
- 避免使用 MySQL 保留字
- 使用下划线分隔单词（如 `user_name`）

### 5. 数据类型选择
- 整数：根据范围选择合适的 INT 类型
- 字符串：根据实际长度选择 VARCHAR 大小
- 时间：使用 TIMESTAMP 或 DATETIME
- 小数：使用 DECIMAL 避免浮点数精度问题

### 6. 使用示例脚本
项目提供了测试脚本，可在非生产环境验证操作：
```bash
# Linux/macOS
./test_new_features.sh run

# Windows
test_new_features.bat run
```

## 与现有功能的结合

### 结合事务管理
```json
{
  "name": "mysql_begin_transaction",
  "arguments": {}
}
```

执行一系列表结构修改...

```json
{
  "name": "mysql_commit_transaction",
  "arguments": {
    "transaction_id": "tx_123456789"
  }
}
```

### 结合索引管理
添加字段后创建索引：
```json
{
  "name": "mysql_create_index",
  "arguments": {
    "table_name": "users",
    "index_name": "idx_email_phone",
    "columns": "email, phone",
    "index_type": "UNIQUE"
  }
}
```

### 结合数据迁移
复杂的数据迁移工作流：
1. 创建临时表
2. 复制数据
3. 修改原表结构
4. 迁移数据回原表
5. 删除临时表

## 性能考虑

### 大型表操作
- 添加/删除字段：对于大型表可能较慢
- 建议：在维护窗口执行
- 监控：使用 `mysql_pool_status` 监控连接池

### 索引影响
- 添加字段不会影响现有索引
- 修改字段类型可能使索引失效
- 删除字段会同时删除相关索引

### 存储空间
- 添加字段会增加存储空间
- 删除字段不会立即释放空间（需要 OPTIMIZE TABLE）
- 修改字段类型可能改变存储需求

## 总结

新功能提供了完整的表结构管理能力，使数据库维护更加便捷和安全。通过事务支持和原子操作，确保了数据的一致性。建议在生产环境使用前充分测试，并遵循最佳实践。

如需进一步帮助，请参考：
- [README.md](README.md) - 项目完整文档
- [PROJECT_SUMMARY.md](PROJECT_SUMMARY.md) - 项目总结
- 测试脚本：`test_new_features.sh` 或 `test_new_features.bat`
