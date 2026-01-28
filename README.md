# Zed MySQL Context Server

This extension provides a Model Context Server for MySQL, for use with the Zed AI assistant.

## Configuration

To use the MCP server, you will need to point the context server at a MySQL database by setting the `mysql-mcp-server` in your Zed `settings.json`:

```json
{
  /// The name of your MCP server
  "mysql-mcp-server": {
    /// The command which runs the MCP server
    "command": "/binary_path/mysql-mcp-server",
    /// The arguments to pass to the MCP server
    "args": [],
    /// The environment variables to set
    "env": {
      "MYSQL_HOST": "mysql_host_domain",
      "MYSQL_PORT": "3306",
      "MYSQL_USER": "mysql_user",
      "MYSQL_PASSWORD": "mysql_password",
      "MYSQL_CONN_MAX_IDLE_TIME_MINUTES": "5",
      "MYSQL_CONN_MAX_LIFETIME_MINUTES": "30",
      "MYSQL_MAX_OPEN_CONNS": "10",
      "MYSQL_MAX_IDLE_CONNS": "5",
      "MYSQL_DATABASE": "database_name"
    }
  }
}
```

## [Usage](./TOOLS_SCHEMA.md)
