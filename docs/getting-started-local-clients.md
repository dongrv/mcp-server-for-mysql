# Claude Desktop 与 Codex 本地接入指南

本指南面向第一次把数据库 MCP Server 接入本地 AI 客户端的开发者，主流程只覆盖 Claude Desktop 与 Codex，以及两种启动方式：本地二进制和 Docker。本文不把任何真实账号、密码或 DSN 写进仓库，也不包含网页端客户端的接入步骤。

## 1. 准备环境

开始前请准备：

- Go 1.24 或更高版本，用于构建本地二进制；如果只用 Docker，则准备 Docker Desktop 或 Docker Engine。
- 最新版 Claude Desktop（Windows 或 macOS）或 Codex CLI/IDE。
- 至少一个可访问的 MySQL 或 ClickHouse 实例。
- 每个数据源一个独立的最小权限数据库账号。只查询的账号只授予元数据读取和 `SELECT` 权限；不要给 MCP 账号管理员权限，也不要让多个数据源共用高权限账号。

下面所有 `PLACEHOLDER_*`、`*.example.invalid` 和 `/ABSOLUTE/PATH/*` 都是不可直接连接的占位符。请只在本机替换它们，不要提交含真实凭据的 `config.yaml`、客户端配置或 `.env` 文件。

## 2. 配置三个数据源

在仓库根目录从 `config.example.yaml` 复制出本机专用的 `config.yaml`。配置文件只引用环境变量；orders 和 logs 是两个相互独立的 MySQL 实例，analytics 是 ClickHouse 实例：

```yaml
mode: quick
sources:
  - name: orders
    display_name: 订单库
    description: 用户充值订单、支付状态与退款记录
    aliases: [充值库, 支付订单]
    keywords: [充值, 支付, 退款, 用户]
    type: mysql
    dsn: ${ORDERS_DSN}
  - name: logs
    display_name: 日志库
    description: 服务运行日志、错误事件与审计记录
    aliases: [服务日志, 错误日志]
    keywords: [报错, 审计, 故障排查]
    type: mysql
    dsn: ${LOGS_DSN}
  - name: analytics
    display_name: 分析库
    description: 聚合指标、用户行为与经营分析数据
    aliases: [数仓, 经营分析]
    keywords: [指标, 趋势, 用户行为, 用户]
    type: clickhouse
    dsn: ${ANALYTICS_DSN}
```

`name` 是工具调用中唯一有效的 `source_id`。`display_name`、`description`、`aliases` 和 `keywords` 只帮助 AI 理解业务含义，不能代替 `source_id`。

先在本机终端设置 DSN。以下字符串只是格式示例，使用了保留域名和明显的占位内容，不能连接真实数据库。

Windows PowerShell：

```powershell
$env:ORDERS_DSN = 'PLACEHOLDER_USER:PLACEHOLDER_SECRET@tcp(orders-db.example.invalid:3306)/PLACEHOLDER_ORDERS_DB?parseTime=true'
$env:LOGS_DSN = 'PLACEHOLDER_USER:PLACEHOLDER_SECRET@tcp(logs-db.example.invalid:3306)/PLACEHOLDER_LOGS_DB?parseTime=true'
$env:ANALYTICS_DSN = 'clickhouse://PLACEHOLDER_USER:PLACEHOLDER_SECRET@analytics.example.invalid:9000/PLACEHOLDER_ANALYTICS_DB'
```

macOS、Linux 或其他 Unix shell：

```sh
export ORDERS_DSN='PLACEHOLDER_USER:PLACEHOLDER_SECRET@tcp(orders-db.example.invalid:3306)/PLACEHOLDER_ORDERS_DB?parseTime=true'
export LOGS_DSN='PLACEHOLDER_USER:PLACEHOLDER_SECRET@tcp(logs-db.example.invalid:3306)/PLACEHOLDER_LOGS_DB?parseTime=true'
export ANALYTICS_DSN='clickhouse://PLACEHOLDER_USER:PLACEHOLDER_SECRET@analytics.example.invalid:9000/PLACEHOLDER_ANALYTICS_DB'
```

## 3. 构建本地二进制

Windows PowerShell：

```powershell
New-Item -ItemType Directory -Force build | Out-Null
go build -o .\build\mcp-database.exe .\cmd
.\build\mcp-database.exe -config .\config.yaml
```

macOS、Linux 或其他 Unix shell：

```sh
mkdir -p build
go build -o ./build/mcp-database ./cmd
./build/mcp-database -config ./config.yaml
```

服务器使用 stdio，正常启动后会等待客户端输入，不会打开网页或 HTTP 端口。手工检查结束后按 `Ctrl+C` 退出，再配置客户端。

## 4. Claude Desktop 使用本地二进制

Claude Desktop 的开发者配置文件位于：

- Windows：`%APPDATA%\Claude\claude_desktop_config.json`
- macOS：`~/Library/Application Support/Claude/claude_desktop_config.json`

也可以在 Claude Desktop 的 Settings > Developer 中选择 Edit Config 打开该文件。保存后必须完全退出并重新启动 Claude Desktop。

Windows JSON 示例：

```json
{
  "mcpServers": {
    "secure-database": {
      "command": "C:\\ABSOLUTE\\PATH\\TO\\mcp-database.exe",
      "args": [
        "-config",
        "C:\\ABSOLUTE\\PATH\\TO\\config.yaml"
      ],
      "env": {
        "ORDERS_DSN": "REPLACE_LOCALLY_WITH_ORDERS_DSN",
        "LOGS_DSN": "REPLACE_LOCALLY_WITH_LOGS_DSN",
        "ANALYTICS_DSN": "REPLACE_LOCALLY_WITH_ANALYTICS_DSN"
      }
    }
  }
}
```

JSON 中的 Windows 反斜杠必须写成 `\\`。例如真实磁盘路径 `C:\Tools\mcp-database.exe` 在 JSON 中应写为 `C:\\Tools\\mcp-database.exe`。

macOS JSON 示例：

```json
{
  "mcpServers": {
    "secure-database": {
      "command": "/ABSOLUTE/PATH/TO/mcp-database",
      "args": [
        "-config",
        "/ABSOLUTE/PATH/TO/config.yaml"
      ],
      "env": {
        "ORDERS_DSN": "REPLACE_LOCALLY_WITH_ORDERS_DSN",
        "LOGS_DSN": "REPLACE_LOCALLY_WITH_LOGS_DSN",
        "ANALYTICS_DSN": "REPLACE_LOCALLY_WITH_ANALYTICS_DSN"
      }
    }
  }
}
```

这里的 `env` 是 Claude Desktop 启动 MCP 子进程时传入的环境变量。该本机文件会包含你替换后的敏感值，因此不要把它复制到仓库或聊天中，并限制其文件访问权限。

## 5. Codex 使用本地二进制

当前 Codex CLI 的 stdio 语法是 `codex mcp add <name> --env KEY=VALUE -- <command>...`。本指南同时用本机 `codex mcp add --help` 和 [OpenAI Codex MCP 官方文档](https://developers.openai.com/codex/mcp/) 校验了下列形式。

macOS、Linux 或其他 Unix shell：

```sh
codex mcp add secure-database \
  --env "ORDERS_DSN=$ORDERS_DSN" \
  --env "LOGS_DSN=$LOGS_DSN" \
  --env "ANALYTICS_DSN=$ANALYTICS_DSN" \
  -- /ABSOLUTE/PATH/TO/mcp-database -config /ABSOLUTE/PATH/TO/config.yaml
```

Windows PowerShell：

```powershell
codex mcp add secure-database `
  --env "ORDERS_DSN=$env:ORDERS_DSN" `
  --env "LOGS_DSN=$env:LOGS_DSN" `
  --env "ANALYTICS_DSN=$env:ANALYTICS_DSN" `
  -- 'C:\ABSOLUTE\PATH\TO\mcp-database.exe' -config 'C:\ABSOLUTE\PATH\TO\config.yaml'
```

执行 `codex mcp list` 确认 `secure-database` 已启用。CLI 形式会把展开后的 `--env` 值写入你的本机 Codex 配置；保护 `~/.codex/config.toml`，不要提交或分享它。

更适合长期使用的等价 `~/.codex/config.toml` 写法是只转发启动 Codex 时已有的环境变量，不把 DSN 值写入 TOML：

```toml
[mcp_servers.secure-database]
command = "/ABSOLUTE/PATH/TO/mcp-database"
args = ["-config", "/ABSOLUTE/PATH/TO/config.yaml"]
env_vars = ["ORDERS_DSN", "LOGS_DSN", "ANALYTICS_DSN"]
```

Windows 上把 `command` 和 `args` 改为绝对路径；TOML 基本字符串中的反斜杠需要转义，或者使用正斜杠，例如 `C:/ABSOLUTE/PATH/TO/mcp-database.exe`。

OpenAI 官方文档说明，同一 Codex host 上的 ChatGPT Desktop、Codex CLI 和 Codex IDE 扩展共享这份本地 MCP 配置，配置一次后可在这些本地客户端间切换。本文不扩展到其他接入方式。

## 6. 使用 Docker 启动 stdio Server

先构建本地镜像：

```sh
docker build -t secure-database-mcp:local .
```

创建一个不提交到版本库的 `mcp-database.env`：

```dotenv
ORDERS_DSN=REPLACE_LOCALLY_WITH_ORDERS_DSN
LOGS_DSN=REPLACE_LOCALLY_WITH_LOGS_DSN
ANALYTICS_DSN=REPLACE_LOCALLY_WITH_ANALYTICS_DSN
```

先在终端验证容器命令。`-i` 必须保留，因为 MCP 使用 stdin/stdout；配置文件以只读方式挂载：

macOS、Linux 或其他 Unix shell：

```sh
docker run --rm -i \
  --env-file /ABSOLUTE/PATH/TO/mcp-database.env \
  --mount type=bind,src=/ABSOLUTE/PATH/TO/config.yaml,dst=/app/config.yaml,readonly \
  secure-database-mcp:local -config /app/config.yaml
```

Windows PowerShell 终端验证：

```powershell
$envFile = (Resolve-Path 'C:\ABSOLUTE\PATH\TO\mcp-database.env').Path
$configFile = (Resolve-Path 'C:\ABSOLUTE\PATH\TO\config.yaml').Path

docker run --rm -i `
  --env-file "$envFile" `
  --mount "type=bind,source=$configFile,target=/app/config.yaml,readonly" `
  secure-database-mcp:local -config /app/config.yaml
```

先把两个 `C:\ABSOLUTE\PATH\TO\...` 替换为已存在文件的完整盘符路径，例如 `C:\Users\your-name\mcp\config.yaml`。`Resolve-Path` 会在命令启动前得到 Docker Desktop 可用的绝对宿主机路径；`source` 是 Windows 文件，`target` 是容器内的 Linux 路径。路径含空格时保留示例中的引号，路径中不要使用逗号。

### Claude Desktop 的 Docker 配置

把 Claude Desktop 的 `secure-database` 项改成：

```json
{
  "mcpServers": {
    "secure-database": {
      "command": "docker",
      "args": [
        "run",
        "--rm",
        "-i",
        "--env-file",
        "/ABSOLUTE/PATH/TO/mcp-database.env",
        "--mount",
        "type=bind,src=/ABSOLUTE/PATH/TO/config.yaml,dst=/app/config.yaml,readonly",
        "secure-database-mcp:local",
        "-config",
        "/app/config.yaml"
      ]
    }
  }
}
```

Windows 用户要把两个宿主机路径换成 Docker Desktop 可访问的绝对路径，并在 JSON 中把每个 `\` 写成 `\\`；如果 GUI 找不到 `docker`，把 `command` 改为 `docker.exe` 的绝对路径。保存后完全重启 Claude Desktop。

### Codex Docker（macOS/Linux）

```sh
codex mcp add secure-database-docker -- \
  docker run --rm -i \
  --env-file /ABSOLUTE/PATH/TO/mcp-database.env \
  --mount type=bind,src=/ABSOLUTE/PATH/TO/config.yaml,dst=/app/config.yaml,readonly \
  secure-database-mcp:local -config /app/config.yaml
```

### Codex Docker（PowerShell）

下面是可直接用于注册的 PowerShell 版本。先替换两个占位路径；变量在注册时展开为绝对路径，Codex 之后会用这些固定参数启动 `docker run`：

```powershell
$envFile = (Resolve-Path 'C:\ABSOLUTE\PATH\TO\mcp-database.env').Path
$configFile = (Resolve-Path 'C:\ABSOLUTE\PATH\TO\config.yaml').Path

codex mcp add secure-database-docker -- `
  docker run --rm -i `
  --env-file "$envFile" `
  --mount "type=bind,source=$configFile,target=/app/config.yaml,readonly" `
  secure-database-mcp:local -config /app/config.yaml
```

运行 `codex mcp list` 确认 `secure-database-docker` 已启用。本地二进制与 Docker 配置二选一即可，避免同时启用两个同名用途的 server。

## 7. 验证发现与选库

重启客户端后，新建对话并先说：

> 请调用 `list_sources`，按 `source_id` 列出每个数据源的 `display_name` 和 `description`；然后对 `orders` 调用 `list_tables`。

预期 `list_sources` 返回 `orders`、`logs` 和 `analytics`，且不返回 DSN、主机、用户名或密码。`list_tables` 必须传精确的 `source_id: "orders"`。如果工具没有出现，先不要发起查询，按文末排障。

唯一候选的自然语言示例：

> 去订单库查看最近 10 笔退款订单，只返回订单号、状态和更新时间。

AI 应先用 `list_sources` 确认“订单库”唯一对应 `orders`，必要时用 `list_tables` 或 `describe_table` 检查真实表结构，再发出类似调用：

```json
{
  "source_id": "orders",
  "sql": "SELECT id, status, updated_at FROM orders WHERE status = 'refunded' ORDER BY updated_at DESC LIMIT 10"
}
```

### 多候选时必须询问

“查一下最近的用户数据”可能同时匹配订单库的用户订单和分析库的用户行为。元数据可能重叠或过期，AI 不得猜测，也不得同时查询两个库。它应先展示候选并等待用户选择，例如：

- `订单库`：用户充值订单、支付状态与退款记录
- `分析库`：聚合指标、用户行为与经营分析数据

只有用户明确选择后，AI 才能把对应的精确 ID（`orders` 或 `analytics`）作为 `source_id` 发给工具。显示名称、别名、关键词和描述都不能直接用于执行。

## 8. 查询、严格模式与高风险确认

`query` 只接受一条只读 SQL，不能执行 `DELETE` 或其他写操作。在默认 `quick` 模式下，单条只读 `query` 可以直接执行；写操作必须调用 `execute_sql`，高风险变更和多语句变更会先返回预览。若希望所有非空 SQL（包括只读查询）都经过确认，把 `config.yaml` 第一行改为：

```yaml
mode: strict
```

重启 MCP Server 后，同一条查询会先返回 `confirmation_required`。AI 必须向用户完整展示预览中的数据源、SQL、风险、原子性和 `preview_hash`，等待明确确认，然后原样重复请求：

```json
{
  "source_id": "orders",
  "sql": "SELECT id, status, updated_at FROM orders WHERE status = 'refunded' ORDER BY updated_at DESC LIMIT 10",
  "confirm": true,
  "preview_hash": "COPY_THE_RETURNED_HASH_EXACTLY"
}
```

高风险示例是通过 `execute_sql` 执行不带 `WHERE` 的删除。第一次调用 `execute_sql` 获取预览，输入为：

```json
{
  "source_id": "logs",
  "sql": "DELETE FROM audit_events"
}
```

此时无论 quick 还是 strict 模式，都必须先展示完整高风险预览并等待用户确认。用户没有明确确认时不得补发 `confirm: true`。

用户确认的内容与预览完全一致时，才使用刚返回的哈希再次调用同一个 `execute_sql` 工具，输入为：

```json
{
  "source_id": "logs",
  "sql": "DELETE FROM audit_events",
  "confirm": true,
  "preview_hash": "COPY_THE_RETURNED_HIGH_RISK_HASH_EXACTLY"
}
```

如果确认前 SQL、参数、工具、风险、原子性或 `source_id` 发生变化，旧哈希不再匹配，服务器返回 `preview_mismatch` 和新的预览。AI 应放弃旧 `preview_hash`，展示新的完整预览，重新等待用户确认；不能自动重试确认，也不能把旧哈希套到新请求上。`display_name` 只是展示信息，不参与哈希绑定，执行边界由精确 `source_id` 控制。

## 9. 常见问题排查

- **找不到本地二进制**：客户端配置必须使用绝对路径；确认 Windows 文件名带 `.exe`，Unix 文件有执行权限，并在终端直接运行同一命令。
- **GUI 没有继承环境变量**：从桌面启动的 Claude Desktop 或 Codex IDE 通常不会继承你刚在终端设置的变量。Claude Desktop 请在 server 的 `env` 中传入；Codex 可用 `env_vars` 转发启动客户端时已有的变量，修改后完全重启客户端。
- **YAML 无法解析**：只用空格缩进，不用 Tab；检查 `sources` 列表、冒号和引号。未知字段会使启动失败。
- **`.env` 不生效**：`.env` 只由 `docker run --env-file` 读取，不会被本地二进制自动读取。确认每行都是 `KEY=VALUE`、没有多余引号或空格，并使用绝对路径。
- **Docker 挂载失败**：确认宿主机文件存在、Docker Desktop 已共享对应磁盘，并保持目标为 `/app/config.yaml`、挂载为 `readonly`。Windows JSON 路径需要双反斜杠。
- **`unknown_source`**：重新调用 `list_sources`，把返回项的精确 `id` 作为 `source_id`；不要传显示名称或别名。
- **自然语言有歧义**：让 AI 列出所有合理候选的 `display_name` 和 `description`，等待你选择；不要允许它猜测或查询多个候选。
- **数据库连接失败**：先在同一终端或容器网络中使用数据库原生客户端验证 DSN、DNS、端口、TLS 和账号权限；容器里的 `localhost` 指向容器自身。服务器只返回脱敏的 `connection_failure`，不会回显 DSN。

## 10. 官方参考与其他客户端

- [OpenAI：Codex MCP](https://developers.openai.com/codex/mcp/)
- [Anthropic / Model Context Protocol：连接本地 MCP Server](https://modelcontextprotocol.io/docs/develop/connect-local-servers)
- [Claude Help：Claude Desktop 本地 MCP Server](https://support.claude.com/en/articles/10949351-getting-started-with-local-mcp-servers-on-claude-desktop)

Zed 不是本指南的主支持流程。仓库仅单独提供一个不含凭据、符合当前 `context_servers` 格式的 [`zed-config-example.json`](../zed-config-example.json) 作为可选参考。
