// Package tools provides MCP tool registration and management.
package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/dongrv/mcp-server-for-mysql/internal/mysql"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Handler is the interface that all tool handlers must implement.
type Handler interface {
	// Name returns the name of the tool.
	Name() string
	// Description returns a description of the tool.
	Description() string
	// Handle processes the tool request.
	Handle(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, any, error)
}

// Registry manages the registration and lookup of tool handlers.
type Registry struct {
	handlers map[string]Handler
}

// NewRegistry creates a new tool registry.
func NewRegistry() *Registry {
	return &Registry{
		handlers: make(map[string]Handler),
	}
}

// Register adds a tool handler to the registry.
func (r *Registry) Register(handler Handler) error {
	if handler == nil {
		return fmt.Errorf("handler cannot be nil")
	}

	name := handler.Name()
	if name == "" {
		return fmt.Errorf("handler name cannot be empty")
	}

	if _, exists := r.handlers[name]; exists {
		return fmt.Errorf("handler already registered: %s", name)
	}

	r.handlers[name] = handler
	return nil
}

// Get returns the handler for the given tool name.
func (r *Registry) Get(name string) (Handler, bool) {
	handler, exists := r.handlers[name]
	return handler, exists
}

// Names returns a list of all registered tool names.
func (r *Registry) Names() []string {
	names := make([]string, 0, len(r.handlers))
	for name := range r.handlers {
		names = append(names, name)
	}
	return names
}

// RegisterAll registers all available tools with the MCP server.
func (r *Registry) RegisterAll(server *mcp.Server, pool *mysql.Pool, txManager *mysql.TxManager) error {
	// Create handlers
	handlers := []Handler{
		NewQueryHandler(pool),
		NewExecuteHandler(pool),
		NewBeginTransactionHandler(txManager),
		NewCommitTransactionHandler(txManager),
		NewRollbackTransactionHandler(txManager),
		NewListTablesHandler(pool),
		NewDescribeTableHandler(pool),
		NewCreateTableHandler(pool),
		NewDropTableHandler(pool),
		NewCreateIndexHandler(pool),
		NewDropIndexHandler(pool),
		NewListIndexesHandler(pool),
		NewMigrateHandler(pool),
		NewPoolStatusHandler(pool),
		NewAddColumnsHandler(pool),
		NewDropColumnsHandler(pool),
		NewModifyColumnsHandler(pool),
		NewRenameTableHandler(pool),
		NewCopyTableHandler(pool),
		NewCopyTableStructureHandler(pool),
	}

	// Register handlers
	for _, handler := range handlers {
		if err := r.Register(handler); err != nil {
			return fmt.Errorf("failed to register handler %s: %w", handler.Name(), err)
		}

		// Register with MCP server using automatic schema inference
		// We use map[string]interface{} for input and output to allow any JSON structure
		mcp.AddTool(server, &mcp.Tool{
			Name:        handler.Name(),
			Description: handler.Description(),
		}, func(ctx context.Context, req *mcp.CallToolRequest, input map[string]interface{}) (*mcp.CallToolResult, map[string]interface{}, error) {
			result, output, err := handler.Handle(ctx, req)
			if err != nil {
				return nil, nil, err
			}
			// Convert output to map[string]interface{} if needed
			if outputMap, ok := output.(map[string]interface{}); ok {
				return result, outputMap, nil
			}
			// If output is not a map, wrap it
			return result, map[string]interface{}{"result": output}, nil
		})
	}

	return nil
}

// baseHandler provides common functionality for all tool handlers.
type baseHandler struct {
	name        string
	description string
	pool        *mysql.Pool
}

// newBaseHandler creates a new base handler.
func newBaseHandler(name, description string, pool *mysql.Pool) baseHandler {
	return baseHandler{
		name:        name,
		description: description,
		pool:        pool,
	}
}

// Name returns the handler name.
func (h *baseHandler) Name() string {
	return h.name
}

// Description returns the handler description.
func (h *baseHandler) Description() string {
	return h.description
}

// formatJSON formats data as indented JSON.
func formatJSON(data interface{}) string {
	jsonBytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Sprintf(`{"error": "Failed to marshal JSON: %v"}`, err)
	}
	return string(jsonBytes)
}

// Helper function to convert parameters to interface{} slice
func convertParams(params []string) []interface{} {
	args := make([]interface{}, len(params))
	for i, param := range params {
		args[i] = param
	}
	return args
}
