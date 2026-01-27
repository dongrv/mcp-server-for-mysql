# MySQL MCP Server for Zed Editor

[![Go Version](https://img.shields.io/badge/Go-1.23%2B-blue)](https://golang.org/)
[![License](https://img.shields.io/badge/License-MIT-green)](LICENSE)
[![MCP Protocol](https://img.shields.io/badge/MCP-Protocol-blueviolet)](https://spec.modelcontextprotocol.io/)

一个基于 Golang MCP 标准库的 MySQL 服务器，为 Zed 编辑器提供完整的数据库管理功能。采用 Google Go 最佳实践设计，代码结构清晰，易于维护和扩展。

## ✨ 功能特性

### 🗃️ CRUD 操作
- **查询数据**: 执行 SELECT 查询并返回格式化结果
- **执行更新**: 执行 INSERT、UPDATE、DELETE 操作
- **参数化查询**: 支持安全的参数化查询

### 🔄 事务管理
- **开始事务**: 创建新的事务并返回事务 ID
- **提交事务**: 提交指定的事务
- **回滚事务**: 回滚指定的事务

### 🏊 数据库连接池管理
- **连接池配置**: 通过环境变量配置连接池参数
- **健康检查**: 自动连接健康检测
- **状态监控**: 实时监控连接池状态
- **连接复用**: 智能连接复用，提高性能

### 📦 数据库迁移管理
- **迁移执行**: 执行数据库迁移脚本
- **事务支持**: 迁移操作在事务中执行
- **错误回滚**: 迁移失败时自动回滚
- **原子操作**: 确保迁移的原子性和一致性

### 🔍 数据库索引管理
- **创建索引**: 支持普通索引、唯一索引、全文索引等
- **删除索引**: 删除指定的索引
- **查看索引**: 列出表的所有索引信息
- **索引优化**: 自动索引管理建议

### 📊 表管理
- **创建表**: 创建新表
- **删除表**: 删除现有表
- **查看表结构**: 显示表的列信息
- **列出所有表**: 显示数据库中的所有表
- **重命名表**: 修改表名称
- **复制表**: 复制表结构和数据
- **复制表结构**: 仅复制表结构（不复制数据）

### 🛠️ 表字段管理
- **添加字段**: 为表添加多个字段（支持批量操作）
- **删除字段**: 从表中删除多个字段（支持批量操作）
- **修改字段**: 修改表的多个字段定义（支持批量操作）
- **字段位置**: 支持指定字段位置（AFTER 子句）

## 🚀 快速开始

### 环境要求
- **Go 1.23+**: 编程语言运行时
- **MySQL 5.7+**: 数据库服务器
- **Zed Editor**: 支持 MCP 协议的编辑器

### 📦 安装

1. **克隆项目**:
```bash
git clone <repository-url>
cd mcp-server-for-mysql
```

2. **安装依赖**:
```bash
go mod download
```

3. **配置环境变量**:
```bash
# 复制示例配置文件
cp env.example .env

# 编辑 .env 文件，设置您的数据库连接信息
vim .env  # 或使用其他编辑器
```

### ⚙️ 环境变量配置

```bash
# MySQL 连接配置
MYSQL_HOST=localhost
MYSQL_PORT=3306
MYSQL_USER=root
MYSQL_PASSWORD=your_password
MYSQL_DATABASE=test

# 连接池配置（优化建议）
MYSQL_MAX_OPEN_CONNS=10      # 最大打开连接数
MYSQL_MAX_IDLE_CONNS=5       # 最大空闲连接数
MYSQL_CONN_MAX_LIFETIME_MINUTES=30  # 连接最大生命周期
MYSQL_CONN_MAX_IDLE_TIME_MINUTES=5  # 连接最大空闲时间

# 服务器配置（可选）
LOG_LEVEL=info               # 日志级别: debug, info, warn, error
LOG_FORMAT=text              # 日志格式: text, json
```

### 🔧 构建和运行

#### 方法一：使用构建脚本（推荐）
```bash
# Linux/macOS
./start.sh build    # 构建项目
./start.sh run      # 运行服务器

# Windows
start.bat build     # 构建项目
start.bat run       # 运行服务器
```

#### 方法二：手动构建
```bash
# 构建项目
go build -o mcp-mysql.exe ./cmd

# 运行服务器
./mcp-mysql.exe
```

#### 方法三：Docker 部署
```bash
# 构建 Docker 镜像
docker build -t mcp-mysql .

# 运行 Docker 容器
docker run --rm -it \
  -e MYSQL_HOST=host.docker.internal \
  -e MYSQL_PORT=3306 \
  -e MYSQL_USER=root \
  -e MYSQL_PASSWORD=your_password \
  -e MYSQL_DATABASE=test \
  mcp-mysql
```

### 🔌 Zed 编辑器配置

将以下配置添加到 Zed 编辑器的 MCP 服务器配置中：

```json
{
  "mcpServers": {
    "mysql": {
      "command": "path/to/mcp-mysql.exe",  # Windows
      // "command": "path/to/mcp-mysql",   # Linux/macOS
      "env": {
        "MYSQL_HOST": "localhost",
        "MYSQL_PORT": "3306",
        "MYSQL_USER": "root",
        "MYSQL_PASSWORD": "your_password",
        "MYSQL_DATABASE": "test",
        "MYSQL_MAX_OPEN_CONNS": "10",
        "MYSQL_MAX_IDLE_CONNS": "5"
      }
    }
  }
}
```

完整配置示例请参考 `zed-config-example.json`。

## 🛠️ 可用工具

### 🔍 查询工具
- `mysql_query`: 执行 SQL 查询语句（支持参数化查询）
- `mysql_execute`: 执行 SQL 更新语句（INSERT, UPDATE, DELETE）

### 🔄 事务工具
- `mysql_begin_transaction`: 开始一个新的事务
- `mysql_commit_transaction`: 提交当前事务
- `mysql_rollback_transaction`: 回滚当前事务

### 📊 表管理工具
- `mysql_list_tables`: 列出数据库中的所有表
- `mysql_describe_table`: 描述表结构
- `mysql_create_table`: 创建新表
- `mysql_drop_table`: 删除表
- `mysql_rename_table`: 重命名表
- `mysql_copy_table`: 复制表结构和数据
- `mysql_copy_table_structure`: 仅复制表结构（不复制数据）

### 🛠️ 表字段管理工具
- `mysql_add_columns`: 为表添加多个字段（批量操作）
- `mysql_drop_columns`: 从表中删除多个字段（批量操作）
- `mysql_modify_columns`: 修改表的多个字段定义（批量操作）

### 🔍 索引管理工具
- `mysql_create_index`: 创建索引（支持 UNIQUE, FULLTEXT 等类型）
- `mysql_drop_index`: 删除索引
- `mysql_list_indexes`: 列出表的所有索引

### ⚙️ 数据库管理工具
- `mysql_migrate`: 执行数据库迁移（事务安全）
- `mysql_pool_status`: 获取数据库连接池状态

## 📝 使用示例

### 🔍 查询数据
```json
{
  "name": "mysql_query",
  "arguments": {
    "query": "SELECT * FROM users WHERE age > ?",
    "parameters": ["18"]
  }
}
```

### 📊 创建表
```json
{
  "name": "mysql_create_table",
  "arguments": {
    "table_name": "products",
    "columns": "id INT AUTO_INCREMENT PRIMARY KEY, name VARCHAR(100), price DECIMAL(10,2), created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP"
  }
}
```

### 🔍 创建索引
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

### ⚙️ 执行迁移
```json
{
  "name": "mysql_migrate",
  "arguments": {
    "migration_sql": "ALTER TABLE users ADD COLUMN last_login TIMESTAMP NULL"
  }
}
```

### 🛠️ 添加多个字段
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

### 🗑️ 删除多个字段
```json
{
  "name": "mysql_drop_columns",
  "arguments": {
    "table_name": "users",
    "columns": ["temp_field1", "temp_field2"]
  }
}
```

### 🔄 重命名表
```json
{
  "name": "mysql_rename_table",
  "arguments": {
    "old_table_name": "old_users",
    "new_table_name": "new_users"
  }
}
```

### 📋 复制表
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

更多详细示例请参考 `NEW_FEATURES_EXAMPLES.md`。

## 🏗️ 项目结构

```
mcp-server-for-mysql/
├── cmd/                    # 应用程序入口
│   ├── main.go            # 主程序入口（已重构）
│   └── test_main.go       # 单元测试入口
├── internal/              # 内部包（外部不可导入）
│   ├── config/           # 配置管理
│   │   └── config.go     # 统一的配置结构
│   ├── mysql/            # MySQL 数据库层
│   │   ├── pool.go       # 连接池管理
│   │   └── txmanager.go  # 事务管理器
│   └── tools/            # MCP 工具层（模块化）
│       ├── registry.go   # 工具注册表
│       ├── query.go      # 查询工具
│       ├── execute.go    # 执行工具
│       ├── transaction.go # 事务工具
│       ├── table.go      # 表管理工具
│       ├── index.go      # 索引管理工具
│       ├── migration.go  # 迁移工具
│       ├── columns.go    # 字段管理工具
│       └── copyrename.go # 表复制/重命名工具
├── go.mod                # Go 模块定义
├── go.sum               # 依赖校验
├── README.md           # 项目文档
├── .env.example        # 环境变量示例
├── Makefile           # 构建自动化
├── Dockerfile        # 容器化部署
├── start.sh          # Linux/macOS 启动脚本
├── start.bat         # Windows 启动脚本
├── zed-config-example.json # Zed 编辑器配置示例
├── NEW_FEATURES_EXAMPLES.md # 新功能使用示例
└── CODE_OPTIMIZATION_SUMMARY.md # 代码优化总结
```

## 🛠️ 开发指南

### 添加新工具

1. **创建参数结构体**：在 `internal/tools/` 目录下创建新的 Go 文件
2. **实现处理器**：实现 `Handler` 接口的 `Name()`, `Description()`, `Handle()` 方法
3. **注册工具**：在 `internal/tools/registry.go` 的 `RegisterAll` 方法中添加新处理器
4. **更新文档**：更新 README 和示例文档

### 模块化设计优势

- **单一职责**：每个文件只负责一个明确的功能
- **易于测试**：独立的模块便于单元测试
- **代码复用**：通用逻辑封装在基础模块中
- **扩展性强**：添加新功能不影响现有代码

### 测试

```bash
# 运行单元测试
go test ./...

# 运行集成测试（需要 MySQL 实例）
go test -tags=integration ./...

# 运行特定测试
go test ./internal/config
go test ./internal/mysql
go test ./internal/tools
```

### 构建 Docker 镜像

```bash
# 构建镜像
docker build -t mcp-mysql .

# 运行测试容器
docker run --rm -it \
  -e MYSQL_HOST=host.docker.internal \
  -e MYSQL_USER=root \
  -e MYSQL_PASSWORD=your_password \
  -e MYSQL_DATABASE=test \
  mcp-mysql

# 生产环境部署
docker run -d \
  --name mysql-mcp-server \
  --restart unless-stopped \
  -e MYSQL_HOST=mysql-server \
  -e MYSQL_USER=app_user \
  -e MYSQL_PASSWORD=secure_password \
  -e MYSQL_DATABASE=production_db \
  -p 8080:8080 \
  mcp-mysql
```

## 🔧 故障排除

### 常见问题

1. **连接失败**: 
   - 检查环境变量配置是否正确
   - 验证 MySQL 服务是否正在运行
   - 检查防火墙和网络连接

2. **权限不足**:
   - 确保数据库用户有足够的权限（SELECT, INSERT, UPDATE, DELETE, ALTER, CREATE, DROP）
   - 检查数据库用户是否可以从当前主机连接

3. **连接池耗尽**:
   - 增加 `MYSQL_MAX_OPEN_CONNS` 值
   - 优化查询性能，减少连接占用时间
   - 检查是否有连接泄漏

4. **工具调用失败**:
   - 检查 Zed 编辑器配置是否正确
   - 验证 MCP 服务器是否正在运行
   - 查看服务器日志获取详细错误信息

### 日志和监控

- **启动日志**: 服务器启动时显示配置信息和连接状态
- **错误日志**: 所有错误都会记录到标准错误输出
- **连接池状态**: 使用 `mysql_pool_status` 工具监控连接池状态
- **性能监控**: 监控查询执行时间和资源使用情况

### 调试技巧

1. **启用详细日志**:
```bash
export LOG_LEVEL=debug
./mcp-mysql.exe
```

2. **测试数据库连接**:
```bash
mysql -h localhost -u root -p
```

3. **验证工具功能**:
```json
{
  "name": "mysql_pool_status",
  "arguments": {}
}
```

4. **检查环境变量**:
```bash
# Linux/macOS
printenv | grep MYSQL_

# Windows
set | findstr MYSQL
```

## 📄 许可证

MIT License - 详见 [LICENSE](LICENSE) 文件

## 🤝 贡献指南

欢迎提交 Issue 和 Pull Request！

### 贡献流程

1. **Fork 项目**: 创建个人分支
2. **创建功能分支**: `git checkout -b feature/your-feature-name`
3. **提交更改**: 遵循 [Conventional Commits](https://www.conventionalcommits.org/) 规范
4. **推送到分支**: `git push origin feature/your-feature-name`
5. **创建 Pull Request**: 提供清晰的描述和测试用例

### 代码规范

- **Go 代码**: 遵循 [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- **命名约定**: 使用有意义的英文名称，避免缩写
- **注释规范**: 为所有导出函数和类型添加 GoDoc 注释
- **错误处理**: 使用 `fmt.Errorf` 包装错误，提供上下文信息
- **测试要求**: 新功能必须包含单元测试

### 开发环境

1. **Go 1.23+**: 确保使用正确版本的 Go
2. **MySQL 5.7+**: 本地开发数据库
3. **Zed Editor**: 用于测试 MCP 集成
4. **Git**: 版本控制工具

### 测试要求

- 所有新功能必须包含单元测试
- 数据库相关功能需要集成测试
- 确保向后兼容性
- 更新相关文档

## 📞 联系方式

- **项目仓库**: [GitHub Repository](https://github.com/yourusername/mcp-server-for-mysql)
- **问题反馈**: [GitHub Issues](https://github.com/yourusername/mcp-server-for-mysql/issues)
- **文档**: [README.md](README.md) | [NEW_FEATURES_EXAMPLES.md](NEW_FEATURES_EXAMPLES.md)

## 🙏 致谢

感谢以下开源项目的贡献：

- [Model Context Protocol](https://spec.modelcontextprotocol.io/) - MCP 协议规范
- [Go MCP SDK](https://github.com/modelcontextprotocol/go-sdk) - Go 语言 MCP SDK
- [Go MySQL Driver](https://github.com/go-sql-driver/mysql) - MySQL 数据库驱动
- [Zed Editor](https://zed.dev/) - 现代化的代码编辑器

---

**最后更新**: 2024年1月  
**版本**: 1.0.0  
**状态**: 🚀 生产就绪 | ✅ 代码优化完成 | 📚 文档完善