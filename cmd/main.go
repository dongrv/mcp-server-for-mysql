package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/dongrv/mcp-server-for-mysql/internal/config"
	"github.com/dongrv/mcp-server-for-mysql/internal/mysql"
	"github.com/dongrv/mcp-server-for-mysql/internal/tools"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func main() {
	// Load configuration
	cfg := config.Default()
	if err := cfg.Validate(); err != nil {
		log.Fatalf("Invalid configuration: %v", err)
	}

	// Create database connection pool
	pool, err := mysql.NewPool(&cfg.MySQL)
	if err != nil {
		log.Fatalf("Failed to create database connection pool: %v", err)
	}
	defer pool.Close()

	// Create transaction manager
	txManager := mysql.NewTxManager(pool)
	defer txManager.Cleanup(0) // Cleanup all transactions on shutdown

	// Create MCP server
	server := mcp.NewServer(&mcp.Implementation{
		Name:    cfg.Server.Name,
		Title:   cfg.Server.Name,
		Version: cfg.Server.Version,
	}, nil)

	// Create tool registry and register all tools
	registry := tools.NewRegistry()
	if err := registry.RegisterAll(server, pool, txManager); err != nil {
		log.Fatalf("Failed to register tools: %v", err)
	}

	// Set up signal handling for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		fmt.Printf("\nReceived signal: %v. Shutting down...\n", sig)
		cancel()
	}()

	// Start server
	fmt.Println("Starting MySQL MCP Server...")
	fmt.Printf("Server: %s v%s\n", cfg.Server.Name, cfg.Server.Version)
	fmt.Printf("Database: %s@%s:%d/%s\n",
		cfg.MySQL.User, cfg.MySQL.Host, cfg.MySQL.Port, cfg.MySQL.Database)
	fmt.Printf("Tools registered: %d\n", len(registry.Names()))
	fmt.Println("Server is ready to accept connections")

	// Run server with stdio transport
	if err := server.Run(ctx, &mcp.StdioTransport{}); err != nil {
		log.Fatalf("Server error: %v", err)
	}

	fmt.Println("Server shutdown complete")
}
