package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"reflect"
	"strings"
	"time"

	"github.com/dongrv/mcp-server-for-mysql/internal/observability"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// RegisterAll attaches the source-aware, stateless tools to an MCP server.
// It deliberately does not register transaction tools: a transaction shared
// across calls would introduce server-side state and bypass the preview model.
func RegisterAll(server *mcp.Server, service *Service) error {
	if server == nil || service == nil {
		return fmt.Errorf("MCP server and service are required")
	}
	register := func(name, description string, input any, handle func(context.Context, any) (Response, error)) {
		registerUntypedTool(server, name, description, input, handle)
	}

	register("list_sources", "List configured database sources.", ListSourcesInput{}, func(ctx context.Context, input any) (Response, error) {
		value := input.(ListSourcesInput)
		sources, err := service.ListSources(ctx, RequestMeta{RequestID: value.RequestID})
		return Response{RequestID: requestID(value.RequestID), State: StateExecuted, Data: sources}, err
	})
	register("list_tables", "List tables in a configured source.", RequestMeta{}, func(ctx context.Context, input any) (Response, error) {
		return service.ListTables(ctx, input.(RequestMeta))
	})
	register("describe_table", "Describe one table in a configured source.", TableInput{}, func(ctx context.Context, input any) (Response, error) {
		return service.DescribeTable(ctx, input.(TableInput))
	})
	register("query", "Run one read-only SQL query.", QueryInput{}, func(ctx context.Context, input any) (Response, error) {
		return service.Query(ctx, input.(QueryInput))
	})
	register("execute_sql", "Execute non-read SQL after applying the configured confirmation policy.", ExecuteSQLInput{}, func(ctx context.Context, input any) (Response, error) {
		return service.ExecuteSQL(ctx, input.(ExecuteSQLInput))
	})
	register("create_table", "Create a table using typed column definitions.", CreateTableInput{}, func(ctx context.Context, input any) (Response, error) {
		return service.CreateTable(ctx, input.(CreateTableInput))
	})
	register("drop_table", "Drop a table after confirmation.", DropTableInput{}, func(ctx context.Context, input any) (Response, error) {
		return service.DropTable(ctx, input.(DropTableInput))
	})
	register("add_columns", "Add typed columns after confirmation.", AddColumnsInput{}, func(ctx context.Context, input any) (Response, error) {
		return service.AddColumns(ctx, input.(AddColumnsInput))
	})
	register("drop_columns", "Drop columns after confirmation.", DropColumnsInput{}, func(ctx context.Context, input any) (Response, error) {
		return service.DropColumns(ctx, input.(DropColumnsInput))
	})
	register("modify_columns", "Modify typed columns after confirmation.", ModifyColumnsInput{}, func(ctx context.Context, input any) (Response, error) {
		return service.ModifyColumns(ctx, input.(ModifyColumnsInput))
	})
	register("create_index", "Create an index after confirmation.", CreateIndexInput{}, func(ctx context.Context, input any) (Response, error) {
		return service.CreateIndex(ctx, input.(CreateIndexInput))
	})
	register("drop_index", "Drop an index after confirmation.", DropIndexInput{}, func(ctx context.Context, input any) (Response, error) {
		return service.DropIndex(ctx, input.(DropIndexInput))
	})
	register("list_indexes", "List normalized indexes for a table.", TableInput{}, func(ctx context.Context, input any) (Response, error) {
		return service.ListIndexes(ctx, input.(TableInput))
	})
	register("rename_table", "Rename a table after confirmation.", RenameTableInput{}, func(ctx context.Context, input any) (Response, error) {
		return service.RenameTable(ctx, input.(RenameTableInput))
	})
	register("copy_table", "Copy table structure and optionally data after confirmation.", CopyTableInput{}, func(ctx context.Context, input any) (Response, error) {
		return service.CopyTable(ctx, input.(CopyTableInput))
	})
	register("copy_table_structure", "Copy table structure after confirmation.", CopyTableInput{}, func(ctx context.Context, input any) (Response, error) {
		return service.CopyTableStructure(ctx, input.(CopyTableInput))
	})
	register("migrate", "Run a migration after confirmation.", MigrateInput{}, func(ctx context.Context, input any) (Response, error) {
		return service.Migrate(ctx, input.(MigrateInput))
	})
	register("pool_status", "Return connection-pool status for a source.", RequestMeta{}, func(ctx context.Context, input any) (Response, error) {
		return service.PoolStatus(ctx, input.(RequestMeta))
	})
	return nil
}

// RegisteredToolNames returns the source-aware, stateless MCP tool surface.
// A copy is returned so callers cannot alter process behavior.
func RegisteredToolNames() []string {
	return []string{
		"list_sources", "list_tables", "describe_table", "query", "execute_sql",
		"create_table", "drop_table", "add_columns", "drop_columns", "modify_columns",
		"create_index", "drop_index", "list_indexes", "rename_table", "copy_table",
		"copy_table_structure", "migrate", "pool_status",
	}
}

type ListSourcesInput struct {
	RequestID string `json:"request_id,omitempty"`
}

func registerUntypedTool(server *mcp.Server, name, description string, input any, handle func(context.Context, any) (Response, error)) {
	mcp.AddTool(server, &mcp.Tool{Name: name, Description: description, InputSchema: JSONSchema(input)}, func(ctx context.Context, request *mcp.CallToolRequest, _ map[string]any) (*mcp.CallToolResult, map[string]any, error) {
		started := time.Now()
		decoded, requestID, err := decodeToolInput(request.Params.Arguments, input)
		if err != nil {
			response := responseForError(requestID, newToolError(CodeInvalidInput, ErrInvalidInput))
			observability.LogEvent(slog.Default(), observability.Event{RequestID: response.RequestID, Tool: name, State: response.State, Duration: time.Since(started), ErrorCode: string(CodeInvalidInput)})
			return mcpResponse(response), responseMap(response), nil
		}
		response, err := handle(ctx, decoded)
		if err != nil {
			response = applyToolError(response, requestID, err)
		}
		errorCode := ""
		if response.Error != nil {
			errorCode = string(*response.Error)
		}
		observability.LogEvent(slog.Default(), observability.Event{RequestID: response.RequestID, Tool: name, SourceID: sourceIDFromValue(reflect.ValueOf(decoded)), State: response.State, Duration: time.Since(started), ErrorCode: errorCode})
		return mcpResponse(response), responseMap(response), nil
	})
}

func sourceIDFromValue(value reflect.Value) string {
	for value.Kind() == reflect.Pointer {
		value = value.Elem()
	}
	if value.Kind() != reflect.Struct {
		return ""
	}
	if field := value.FieldByName("SourceID"); field.IsValid() && field.Kind() == reflect.String {
		return field.String()
	}
	for index := 0; index < value.NumField(); index++ {
		if sourceID := sourceIDFromValue(value.Field(index)); sourceID != "" {
			return sourceID
		}
	}
	return ""
}

func applyToolError(response Response, fallbackRequestID string, err error) Response {
	code := ErrorCodeFor(err)
	response.Error = &code
	if response.RequestID == "" {
		response.RequestID = fallbackRequestID
	}
	if response.State == "" {
		response.State = StateError
	}
	return response
}

func decodeToolInput(raw json.RawMessage, prototype any) (any, string, error) {
	typeOf := reflect.TypeOf(prototype)
	if typeOf.Kind() != reflect.Struct {
		return nil, "", fmt.Errorf("tool input must be a struct")
	}
	value := reflect.New(typeOf)
	if len(raw) == 0 {
		raw = []byte("{}")
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if err := decoder.Decode(value.Interface()); err != nil {
		return nil, "", err
	}
	requestID := requestIDFromValue(value.Elem())
	return value.Elem().Interface(), requestID, nil
}

func requestIDFromValue(value reflect.Value) string {
	if field := value.FieldByName("RequestID"); field.IsValid() && field.Kind() == reflect.String {
		return field.String()
	}
	for index := 0; index < value.NumField(); index++ {
		field := value.Field(index)
		if field.Kind() == reflect.Struct {
			if requestID := requestIDFromValue(field); requestID != "" {
				return requestID
			}
		}
	}
	return ""
}

func mcpResponse(response Response) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: formatJSON(response)}},
		IsError: response.Error != nil,
	}
}

func responseMap(response Response) map[string]any {
	encoded, err := json.Marshal(response)
	if err != nil {
		return map[string]any{"state": StateError, "error_code": string(CodeExecution)}
	}
	var result map[string]any
	if err := json.Unmarshal(encoded, &result); err != nil {
		return map[string]any{"state": StateError, "error_code": string(CodeExecution)}
	}
	return result
}

// JSONSchema derives the basic JSON schema from the typed input struct. This
// keeps exposed MCP arguments aligned with the service model without a second,
// hand-maintained schema registry.
func JSONSchema(input any) map[string]any {
	return objectSchema(reflect.TypeOf(input))
}

func objectSchema(typeOf reflect.Type) map[string]any {
	for typeOf.Kind() == reflect.Pointer {
		typeOf = typeOf.Elem()
	}
	properties := make(map[string]any)
	required := make([]string, 0)
	if typeOf.Kind() != reflect.Struct {
		return map[string]any{"type": "object", "additionalProperties": false}
	}
	for index := 0; index < typeOf.NumField(); index++ {
		field := typeOf.Field(index)
		if field.PkgPath != "" {
			continue
		}
		if field.Anonymous {
			for name, value := range objectSchema(field.Type)["properties"].(map[string]any) {
				properties[name] = value
			}
			for _, name := range objectSchema(field.Type)["required"].([]string) {
				required = append(required, name)
			}
			continue
		}
		name, optional := jsonFieldName(field)
		if name == "" {
			continue
		}
		properties[name] = valueSchema(field.Type)
		if !optional {
			required = append(required, name)
		}
	}
	schema := map[string]any{"type": "object", "properties": properties, "additionalProperties": false}
	if len(required) > 0 {
		schema["required"] = required
	}
	return schema
}

func jsonFieldName(field reflect.StructField) (string, bool) {
	tag := field.Tag.Get("json")
	parts := strings.Split(tag, ",")
	if len(parts) == 0 || parts[0] == "" || parts[0] == "-" {
		return "", true
	}
	for _, option := range parts[1:] {
		if option == "omitempty" {
			return parts[0], true
		}
	}
	return parts[0], false
}

func valueSchema(typeOf reflect.Type) map[string]any {
	for typeOf.Kind() == reflect.Pointer {
		typeOf = typeOf.Elem()
	}
	switch typeOf.Kind() {
	case reflect.String:
		return map[string]any{"type": "string"}
	case reflect.Bool:
		return map[string]any{"type": "boolean"}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return map[string]any{"type": "integer"}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return map[string]any{"type": "integer", "minimum": 0}
	case reflect.Float32, reflect.Float64:
		return map[string]any{"type": "number"}
	case reflect.Slice, reflect.Array:
		return map[string]any{"type": "array", "items": valueSchema(typeOf.Elem())}
	case reflect.Struct:
		return objectSchema(typeOf)
	default:
		return map[string]any{}
	}
}
