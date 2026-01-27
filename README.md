# MySQL MCP Server for Zed Editor

一个基于 Golang MCP 标准库的 MySQL 服务器，为 Zed 编辑器提供完整的数据库管理功能。

## 功能特性

### CRUD 操作
- **查询数据**: 执行 SELECT 查询并返回格式化结果
- **执行更新**: 执行 INSERT、UPDATE、DELETE 操作
- **参数化查询**: 支持安全的参数化查询

### 事务管理
- **开始事务**: 创建新的事务并返回事务 ID
- **提交事务**: 提交指定的事务
- **回滚事务**: 回滚指定的事务

### 数据库连接池管理
- **连接池配置**: 通过环境变量配置连接池参数
- **健康检查**: 自动连接健康检测
- **状态监控**: 实时监控连接池状态

### 数据库迁移管理
- **迁移执行**: 执行数据库迁移脚本
- **事务支持**: 迁移操作在事务中执行
- **错误回滚**: 迁移失败时自动回滚

### 数据库索引管理
- **创建索引**: 支持普通索引、唯一索引、全文索引等
- **删除索引**: 删除指定的索引
- **查看索引**: 列出表的所有索引信息

### 表管理
- **创建表**: 创建新表
- **删除表**: 删除现有表
- **查看表结构**: 显示表的列信息
- **列出所有表**: 显示数据库中的所有表
- **重命名表**: 修改表名称
- **复制表**: 复制表结构和数据
- **复制表结构**: 仅复制表结构（不复制数据）

### 表字段管理
- **添加字段**: 为表添加多个字段
- **删除字段**: 从表中删除多个字段
- **修改字段**: 修改表的多个字段定义

## 快速开始

### 环境要求
- Go 1.23 或更高版本
- MySQL 5.7 或更高版本

### 安装

1. 克隆项目：
```bash
git clone <repository-url>
cd mcp-mysql
```

2. 安装依赖：
```bash
go mod download
```

3. 配置环境变量：
```bash
# 复制示例配置文件
cp .env.example .env

# 编辑 .env 文件，设置您的数据库连接信息
```

### 环境变量配置

```bash
# MySQL 连接配置
MYSQL_HOST=localhost
MYSQL_PORT=3306
MYSQL_USER=root
MYSQL_PASSWORD=your_password
MYSQL_DATABASE=test

# 连接池配置
MYSQL_MAX_CONNS=10
MYSQL_IDLE_CONNS=5
MYSQL_CONN_LIFETIME_MINUTES=30
```

### 构建和运行

1. 构建项目：
```bash
go build -o mcp-mysql ./cmd
```

2. 运行服务器：
```bash
./mcp-mysql
```

3. 在 Zed 编辑器中配置 MCP 服务器：
```json
{
  "mcpServers": {
    "mysql": {
      "command": "path/to/mcp-mysql",
      "env": {
        "MYSQL_HOST": "localhost",
        "MYSQL_PORT": "3306",
        "MYSQL_USER": "root",
        "MYSQL_PASSWORD": "your_password",
        "MYSQL_DATABASE": "test"
      }
    }
  }
}
```

## 可用工具

### 查询工具
- `mysql_query`: 执行 SQL 查询语句
- `mysql_execute`: 执行 SQL 更新语句

### 事务工具
- `mysql_begin_transaction`: 开始一个新的事务
- `mysql_commit_transaction`: 提交当前事务
- `mysql_rollback_transaction`: 回滚当前事务

### 表管理工具
- `mysql_list_tables`: 列出数据库中的所有表
- `mysql_describe_table`: 描述表结构
- `mysql_create_table`: 创建新表
- `mysql_drop_table`: 删除表
- `mysql_rename_table`: 重命名表
- `mysql_copy_table`: 复制表结构和数据
- `mysql_copy_table_structure`: 仅复制表结构（不复制数据）

### 表字段管理工具
- `mysql_add_columns`: 为表添加多个字段
- `mysql_drop_columns`: 从表中删除多个字段
- `mysql_modify_columns`: 修改表的多个字段定义

### 索引管理工具
- `mysql_create_index`: 创建索引
- `mysql_drop_index`: 删除索引
- `mysql_list_indexes`: 列出表的所有索引

### 数据库管理工具
- `mysql_migrate`: 执行数据库迁移
- `mysql_pool_status`: 获取数据库连接池状态

## 使用示例

### 查询数据
```json
{
  "name": "mysql_query",
  "arguments": {
    "query": "SELECT * FROM users WHERE age > ?",
    "parameters": ["18"]
  }
}
```

### 创建表
```json
{
  "name": "mysql_create_table",
  "arguments": {
    "table_name": "products",
    "columns": "id INT AUTO_INCREMENT PRIMARY KEY, name VARCHAR(100), price DECIMAL(10,2), created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP"
  }
}
```

### 创建索引
```json
{
  "name": "mysql_create_index",
  "arguments": {
    "table_name": "users",
    "index_name": "idx_email",
    "columns": "email",
    "index_type": "UNIQUE"
  }
}
```

### 执行迁移
```json
{
  "name": "mysql_migrate",
  "arguments": {
    "migration_sql": "ALTER TABLE users ADD COLUMN last_login TIMESTAMP NULL"
  }
}
```

### 添加多个字段
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
      }
    ]
  }
}
```

### 删除多个字段
```json
{
  "name": "mysql_drop_columns",
  "arguments": {
    "table_name": "users",
    "columns": ["temp_field1", "temp_field2"]
  }
}
```

### 重命名表
```json
{
  "name": "mysql_rename_table",
  "arguments": {
    "old_table_name": "old_users",
    "new_table_name": "new_users"
  }
}
```

### 复制表
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

## 项目结构

```
mcp-mysql/
├── cmd/
│   └── main.go              # 主程序入口
├── pkg/
│   ├── database/
│   │   └── pool.go          # 数据库连接池管理
│   └── tools/
│       └── handler.go       # MCP 工具处理器
├── go.mod                   # Go 模块定义
├── go.sum                   # 依赖校验
├── README.md               # 项目文档
└── .env.example            # 环境变量示例
```

## 开发指南

### 添加新工具

1. 在 `pkg/tools/handler.go` 中的 `RegisterTools` 方法中添加新工具注册
2. 实现对应的处理函数
3. 更新 README 文档

### 测试

```bash
# 运行单元测试
go test ./...

# 运行集成测试（需要 MySQL 实例）
go test -tags=integration ./...
```

### 构建 Docker 镜像

```bash
docker build -t mcp-mysql .
docker run -e MYSQL_HOST=host.docker.internal -e MYSQL_USER=root -e MYSQL_PASSWORD=password mcp-mysql
```

## 故障排除

### 常见问题

1. **连接失败**: 检查环境变量配置和 MySQL 服务状态
2. **权限不足**: 确保数据库用户有足够的权限
3. **连接池耗尽**: 增加 `MYSQL_MAX_CONNS` 值

### 日志

服务器会在启动时显示连接信息，错误会记录到标准错误输出。

## 许可证

MIT License

## 贡献

欢迎提交 Issue 和 Pull Request！

1. Fork 项目
2. 创建功能分支
3. 提交更改
4. 推送到分支
5. 创建 Pull Request