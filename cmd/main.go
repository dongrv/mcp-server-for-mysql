package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/dongrv/mcp-server-for-mysql/internal/config"
	"github.com/dongrv/mcp-server-for-mysql/internal/database"
	"github.com/dongrv/mcp-server-for-mysql/internal/tools"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	applicationName    = "database-mcp-server"
	applicationVersion = "1.0.0"
	startupTimeout     = 10 * time.Second
)

type registryOpener func(context.Context, config.Config) (*database.Registry, error)

// application owns the process-lifetime resources for one stateless MCP
// server. Request state remains solely in MCP tool inputs and outputs.
type application struct {
	server    *mcp.Server
	registry  *database.Registry
	toolNames []string
}

func (a *application) Server() *mcp.Server { return a.server }

func (a *application) ToolNames() []string {
	return append([]string(nil), a.toolNames...)
}

func (a *application) Close() error {
	if a == nil || a.registry == nil {
		return nil
	}
	return a.registry.Close()
}

func buildApplication(ctx context.Context, cfg config.Config, open registryOpener) (*application, error) {
	if open == nil {
		return nil, fmt.Errorf("database registry opener is required")
	}
	registry, err := open(ctx, cfg)
	if err != nil {
		return nil, err
	}
	if registry == nil {
		return nil, fmt.Errorf("database registry opener returned nil registry")
	}

	server := mcp.NewServer(&mcp.Implementation{
		Name:    applicationName,
		Title:   applicationName,
		Version: applicationVersion,
	}, nil)
	service := tools.NewService(registry, cfg.Mode, nil)
	if err := tools.RegisterAll(server, service); err != nil {
		_ = registry.Close()
		return nil, fmt.Errorf("register MCP tools: %w", err)
	}
	return &application{server: server, registry: registry, toolNames: tools.RegisteredToolNames()}, nil
}

func openApplication(ctx context.Context, cfg config.Config) (*application, error) {
	startup, cancel := context.WithTimeout(ctx, startupTimeout)
	defer cancel()
	return buildApplication(startup, cfg, database.OpenRegistry)
}

func main() {
	configPath := flag.String("config", "config.yaml", "path to MCP configuration")
	flag.Parse()

	cfg, err := config.Load(*configPath, os.LookupEnv)
	if err != nil {
		log.Fatalf("load configuration: %v", err)
	}
	app, err := openApplication(context.Background(), cfg)
	if err != nil {
		log.Fatalf("open configured database sources: %v", err)
	}
	defer func() {
		if err := app.Close(); err != nil {
			log.Printf("close database sources: %v", err)
		}
	}()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	log.Printf("starting %s v%s with %d configured source(s) and %d tool(s)", applicationName, applicationVersion, len(cfg.Sources), len(app.ToolNames()))
	if err := app.Server().Run(ctx, &mcp.StdioTransport{}); err != nil {
		log.Fatalf("MCP server: %v", err)
	}
}
