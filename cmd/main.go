package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/dongrv/mcp-mysql/pkg/database"
	"github.com/dongrv/mcp-mysql/pkg/tools"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func main() {
	// 创建数据库连接池
	dbPool, err := database.NewConnectionPool()
	if err != nil {
		log.Fatalf("Failed to create database connection pool: %v", err)
	}
	defer dbPool.Close()

	// 创建 MCP 服务器
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "mysql-mcp-server",
		Title:   "MySQL MCP Server",
		Version: "1.0.0",
	}, nil)

	// 注册工具
	toolHandler := tools.NewToolHandler(dbPool)
	toolHandler.RegisterTools(server)

	// 设置信号处理
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		fmt.Printf("\nReceived signal: %v. Shutting down...\n", sig)
		cancel()
	}()

	// 启动服务器
	fmt.Println("Starting MySQL MCP Server...")
	fmt.Println("Server is ready to accept connections")

	// 使用 StdioTransport 运行服务器
	if err := server.Run(ctx, &mcp.StdioTransport{}); err != nil {
		log.Fatalf("Server error: %v", err)
	}

	fmt.Println("Server shutdown complete")
}
