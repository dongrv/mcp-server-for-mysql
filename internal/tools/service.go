package tools

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/dongrv/mcp-server-for-mysql/internal/config"
	"github.com/dongrv/mcp-server-for-mysql/internal/database"
	"github.com/dongrv/mcp-server-for-mysql/internal/execution"
	"github.com/dongrv/mcp-server-for-mysql/internal/operation"
	"github.com/dongrv/mcp-server-for-mysql/internal/sqlguard"
)

type Service struct {
	registry *database.Registry
	mode     config.Mode
	executor execution.Executor
}

func NewService(registry *database.Registry, mode config.Mode, executor *execution.Executor) *Service {
	if mode == "" {
		mode = config.QuickMode
	}
	value := execution.NewDefaultExecutor()
	if executor != nil {
		value = *executor
	}
	return &Service{registry: registry, mode: mode, executor: value}
}

type SourceInfo struct {
	ID          string   `json:"id"`
	Engine      string   `json:"engine"`
	DisplayName string   `json:"display_name"`
	Description string   `json:"description"`
	Aliases     []string `json:"aliases"`
	Keywords    []string `json:"keywords"`
}

func (s *Service) ListSources(_ context.Context, _ RequestMeta) ([]SourceInfo, error) {
	if s == nil || s.registry == nil {
		return nil, newToolError(CodeExecution, ErrExecution)
	}
	sources := s.registry.Sources()
	result := make([]SourceInfo, len(sources))
	for i, source := range sources {
		profile := source.Profile()
		if profile.Aliases != nil {
			profile.Aliases = append([]string{}, profile.Aliases...)
		}
		if profile.Keywords != nil {
			profile.Keywords = append([]string{}, profile.Keywords...)
		}
		result[i] = SourceInfo{
			ID:          source.ID(),
			Engine:      source.Engine(),
			DisplayName: profile.DisplayName,
			Description: profile.Description,
			Aliases:     profile.Aliases,
			Keywords:    profile.Keywords,
		}
	}
	return result, nil
}

func (s *Service) ListTables(ctx context.Context, input RequestMeta) (Response, error) {
	requestID := requestID(input.RequestID)
	source, err := s.source(input.SourceID)
	if err != nil {
		return Response{RequestID: requestID}, err
	}
	ctx, cancel := metadataContext(ctx)
	defer cancel()
	tables, err := source.Dialect().ListTables(ctx, source.DB())
	if err != nil {
		return Response{RequestID: requestID}, databaseError(err)
	}
	return Response{RequestID: requestID, State: StateExecuted, Data: tables}, nil
}

func (s *Service) DescribeTable(ctx context.Context, input TableInput) (Response, error) {
	requestID := requestID(input.RequestID)
	source, err := s.source(input.SourceID)
	if err != nil {
		return Response{RequestID: requestID}, err
	}
	if strings.TrimSpace(input.Table) == "" {
		return Response{RequestID: requestID}, newToolError(CodeInvalidInput, ErrInvalidInput)
	}
	ctx, cancel := metadataContext(ctx)
	defer cancel()
	description, err := source.Dialect().DescribeTable(ctx, source.DB(), input.Table)
	if err != nil {
		return Response{RequestID: requestID}, databaseError(err)
	}
	return Response{RequestID: requestID, State: StateExecuted, Data: description}, nil
}

func (s *Service) ListIndexes(ctx context.Context, input TableInput) (Response, error) {
	response, err := s.DescribeTable(ctx, input)
	if err != nil {
		return response, err
	}
	description := response.Data.(database.TableDescription)
	response.Data = description.Indexes
	return response, nil
}

func (s *Service) PoolStatus(_ context.Context, input RequestMeta) (Response, error) {
	requestID := requestID(input.RequestID)
	source, err := s.source(input.SourceID)
	if err != nil {
		return Response{RequestID: requestID}, err
	}
	if source.DB() == nil {
		return Response{RequestID: requestID}, newToolError(CodeConnection, execution.ErrNilSourceDatabase)
	}
	stats := source.DB().Stats()
	return Response{RequestID: requestID, State: StateExecuted, Data: stats}, nil
}

func (s *Service) Query(ctx context.Context, input QueryInput) (Response, error) {
	requestID := requestID(input.RequestID)
	source, plan, err := s.analyze(input.SourceID, input.SQL)
	if err != nil {
		return Response{RequestID: requestID}, err
	}
	if !plan.ReadOnly || len(plan.Statements) != 1 {
		return Response{RequestID: requestID}, newToolError(CodeUnsafeSQL, ErrReadOnlySQLRequired)
	}
	parameters, err := normalizeParameters(input.Parameters)
	if err != nil {
		return Response{RequestID: requestID}, err
	}
	response, allowed, err := s.authorize(requestID, input.RequestMeta, source, "query", plan, parameters, false)
	if err != nil || !allowed {
		return response, err
	}
	query, err := s.executor.Query(ctx, source.DB(), plan.Statements[0].NormalizedSQL, parameters)
	if err != nil {
		return Response{RequestID: requestID}, databaseError(err)
	}
	response.Query = &query
	response.State = StateExecuted
	return response, nil
}

func (s *Service) ExecuteSQL(ctx context.Context, input ExecuteSQLInput) (Response, error) {
	return s.executeSQL(ctx, input.RequestMeta, "execute_sql", input.SQL, input.Parameters, false)
}

func (s *Service) Migrate(ctx context.Context, input MigrateInput) (Response, error) {
	return s.executeSQL(ctx, input.RequestMeta, "migrate", input.SQL, input.Parameters, true)
}

func (s *Service) executeSQL(ctx context.Context, meta RequestMeta, tool, text string, parameters []any, forceHighRisk bool) (Response, error) {
	requestID := requestID(meta.RequestID)
	source, plan, err := s.analyze(meta.SourceID, text)
	if err != nil {
		return Response{RequestID: requestID}, err
	}
	if plan.ReadOnly {
		return Response{RequestID: requestID}, newToolError(CodeUnsafeSQL, ErrMutationSQLRequired)
	}
	if forceHighRisk || len(plan.Statements) > 1 {
		plan.Risk = sqlguard.HighRisk
	}
	normalized, err := normalizeParameters(parameters)
	if err != nil {
		return Response{RequestID: requestID}, err
	}
	response, allowed, err := s.authorize(requestID, meta, source, tool, plan, normalized, false)
	if err != nil || !allowed {
		return response, err
	}
	result, err := s.executor.ExecutePlan(ctx, source, plan, normalized)
	if err != nil {
		return Response{RequestID: requestID}, databaseError(err)
	}
	response.Execution = &result
	response.State = StateExecuted
	return response, nil
}

func (s *Service) CreateTable(ctx context.Context, input CreateTableInput) (Response, error) {
	return s.executeOperation(ctx, input.RequestMeta, "create_table", func(builder operation.Builder) ([]string, error) {
		return builder.CreateTable(operation.CreateTableRequest{Table: input.Table, Columns: input.Columns})
	})
}

func (s *Service) DropTable(ctx context.Context, input DropTableInput) (Response, error) {
	return s.executeOperation(ctx, input.RequestMeta, "drop_table", func(builder operation.Builder) ([]string, error) {
		return builder.DropTable(operation.DropTableRequest{Table: input.Table})
	})
}

func (s *Service) AddColumns(ctx context.Context, input AddColumnsInput) (Response, error) {
	return s.executeOperation(ctx, input.RequestMeta, "add_columns", func(builder operation.Builder) ([]string, error) {
		return builder.AddColumns(operation.AddColumnsRequest{Table: input.Table, Columns: input.Columns})
	})
}

func (s *Service) DropColumns(ctx context.Context, input DropColumnsInput) (Response, error) {
	return s.executeOperation(ctx, input.RequestMeta, "drop_columns", func(builder operation.Builder) ([]string, error) {
		return builder.DropColumns(operation.DropColumnsRequest{Table: input.Table, Columns: input.Columns})
	})
}

func (s *Service) ModifyColumns(ctx context.Context, input ModifyColumnsInput) (Response, error) {
	return s.executeOperation(ctx, input.RequestMeta, "modify_columns", func(builder operation.Builder) ([]string, error) {
		return builder.ModifyColumns(operation.ModifyColumnsRequest{Table: input.Table, Columns: input.Columns})
	})
}

func (s *Service) CreateIndex(ctx context.Context, input CreateIndexInput) (Response, error) {
	return s.executeOperation(ctx, input.RequestMeta, "create_index", func(builder operation.Builder) ([]string, error) {
		return builder.CreateIndex(operation.CreateIndexRequest{Table: input.Table, Index: input.Index, Columns: input.Columns, Unique: input.Unique})
	})
}

func (s *Service) DropIndex(ctx context.Context, input DropIndexInput) (Response, error) {
	return s.executeOperation(ctx, input.RequestMeta, "drop_index", func(builder operation.Builder) ([]string, error) {
		return builder.DropIndex(operation.DropIndexRequest{Table: input.Table, Index: input.Index})
	})
}

func (s *Service) RenameTable(ctx context.Context, input RenameTableInput) (Response, error) {
	return s.executeOperation(ctx, input.RequestMeta, "rename_table", func(builder operation.Builder) ([]string, error) {
		return builder.RenameTable(operation.RenameTableRequest{From: input.From, To: input.To})
	})
}

func (s *Service) CopyTable(ctx context.Context, input CopyTableInput) (Response, error) {
	return s.copyTable(ctx, input, "copy_table", input.WithData)
}

func (s *Service) CopyTableStructure(ctx context.Context, input CopyTableInput) (Response, error) {
	return s.copyTable(ctx, input, "copy_table_structure", false)
}

func (s *Service) copyTable(ctx context.Context, input CopyTableInput, tool string, withData bool) (Response, error) {
	return s.executeOperation(ctx, input.RequestMeta, tool, func(builder operation.Builder) ([]string, error) {
		return builder.CopyTable(operation.CopyTableRequest{Source: input.Source, Destination: input.Destination, WithData: withData})
	})
}

func (s *Service) executeOperation(ctx context.Context, meta RequestMeta, tool string, build func(operation.Builder) ([]string, error)) (Response, error) {
	requestID := requestID(meta.RequestID)
	source, err := s.source(meta.SourceID)
	if err != nil {
		return Response{RequestID: requestID}, err
	}
	statements, err := build(builderFor(source))
	if err != nil {
		return Response{RequestID: requestID}, operationError(err)
	}
	analyzer, err := analyzerFor(source)
	if err != nil {
		return Response{RequestID: requestID}, err
	}
	plan, err := analyzer.Analyze(strings.Join(statements, "; "))
	if err != nil || plan.ReadOnly {
		return Response{RequestID: requestID}, newToolError(CodeUnsafeSQL, errors.Join(ErrUnsafeSQL, err))
	}
	response, allowed, err := s.authorize(requestID, meta, source, tool, plan, nil, false)
	if err != nil || !allowed {
		return response, err
	}
	result, err := s.executor.ExecutePlan(ctx, source, plan, nil)
	if err != nil {
		return Response{RequestID: requestID}, databaseError(err)
	}
	response.State = StateExecuted
	response.Execution = &result
	return response, nil
}

func (s *Service) authorize(requestID string, meta RequestMeta, source database.Source, tool string, plan sqlguard.Plan, parameters []any, forceHighRisk bool) (Response, bool, error) {
	if forceHighRisk {
		plan.Risk = sqlguard.HighRisk
	}
	decision, preview, err := execution.Authorize(s.mode, execution.Intent{
		SourceID: source.ID(), Tool: tool, Parameters: parameters,
		Atomic: execution.IsAtomicBatch(source, plan), Plan: plan,
	}, execution.Confirmation{Confirm: meta.Confirm, PreviewHash: meta.PreviewHash})
	if err != nil {
		return Response{RequestID: requestID}, false, newToolError(CodeUnsafeSQL, errors.Join(ErrUnsafeSQL, err))
	}
	profile := source.Profile()
	preview.Source = &execution.SourceReference{ID: source.ID(), DisplayName: profile.DisplayName}
	switch decision.State {
	case execution.ExecuteNow:
		return Response{RequestID: requestID, State: StateExecuted}, true, nil
	case execution.PreviewRequired:
		return Response{RequestID: requestID, State: StateConfirmationRequired, Preview: &preview}, false, nil
	case execution.PreviewMismatch:
		return Response{RequestID: requestID, State: StatePreviewMismatch, Preview: &preview}, false, newToolError(CodePreviewMismatch, ErrPreviewMismatch)
	default:
		return Response{RequestID: requestID}, false, newToolError(CodeUnsafeSQL, ErrUnsafeSQL)
	}
}

func (s *Service) source(id string) (database.Source, error) {
	if s == nil || s.registry == nil || strings.TrimSpace(id) == "" {
		return nil, newToolError(CodeInvalidInput, ErrInvalidInput)
	}
	source, err := s.registry.Source(id)
	if errors.Is(err, database.ErrUnknownSource) {
		return nil, newToolError(CodeUnknownSource, err)
	}
	if err != nil {
		return nil, newToolError(CodeExecution, err)
	}
	return source, nil
}

func (s *Service) analyze(sourceID, text string) (database.Source, sqlguard.Plan, error) {
	source, err := s.source(sourceID)
	if err != nil {
		return nil, sqlguard.Plan{}, err
	}
	if strings.TrimSpace(text) == "" {
		return nil, sqlguard.Plan{}, newToolError(CodeInvalidInput, ErrInvalidInput)
	}
	analyzer, err := analyzerFor(source)
	if err != nil {
		return nil, sqlguard.Plan{}, err
	}
	plan, err := analyzer.Analyze(text)
	if err != nil {
		return nil, sqlguard.Plan{}, newToolError(CodeUnsafeSQL, errors.Join(ErrUnsafeSQL, err))
	}
	return source, plan, nil
}

func analyzerFor(source database.Source) (sqlguard.Analyzer, error) {
	switch source.Engine() {
	case "mysql":
		analyzer, err := sqlguard.NewMySQLAnalyzer("")
		if err != nil {
			return nil, newToolError(CodeExecution, err)
		}
		return analyzer, nil
	case "clickhouse":
		return sqlguard.NewClickHouseAnalyzer(), nil
	default:
		return nil, newToolError(CodeUnsupported, database.ErrUnsupportedCapability)
	}
}

func builderFor(source database.Source) operation.Builder {
	switch source.Engine() {
	case "clickhouse":
		return operation.ClickHouseBuilder{}
	default:
		return operation.MySQLBuilder{}
	}
}

func normalizeParameters(values []any) ([]any, error) {
	if values == nil {
		return nil, nil
	}
	result := make([]any, len(values))
	for index, value := range values {
		normalized, err := normalizeJSONValue(value)
		if err != nil {
			return nil, newToolError(CodeInvalidInput, errors.Join(ErrInvalidInput, err))
		}
		result[index] = normalized
	}
	return result, nil
}

func normalizeJSONValue(value any) (any, error) {
	switch value := value.(type) {
	case json.Number:
		if integer, err := value.Int64(); err == nil {
			return integer, nil
		}
		floating, err := value.Float64()
		if err != nil {
			return nil, fmt.Errorf("invalid JSON number")
		}
		return floating, nil
	case []any:
		result := make([]any, len(value))
		for index := range value {
			normalized, err := normalizeJSONValue(value[index])
			if err != nil {
				return nil, err
			}
			result[index] = normalized
		}
		return result, nil
	case map[string]any:
		result := make(map[string]any, len(value))
		for key, item := range value {
			normalized, err := normalizeJSONValue(item)
			if err != nil {
				return nil, err
			}
			result[key] = normalized
		}
		return result, nil
	default:
		return value, nil
	}
}

func requestID(value string) string {
	if strings.TrimSpace(value) != "" {
		return value
	}
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "request-unavailable"
	}
	return hex.EncodeToString(bytes)
}

func metadataContext(ctx context.Context) (context.Context, context.CancelFunc) {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithTimeout(ctx, execution.DefaultQueryTimeout)
}

func operationError(err error) error {
	if errors.Is(err, database.ErrUnsupportedCapability) {
		return newToolError(CodeUnsupported, errors.Join(ErrUnsupported, err))
	}
	return newToolError(CodeInvalidInput, errors.Join(ErrInvalidInput, err))
}

func databaseError(err error) error {
	if errors.Is(err, context.DeadlineExceeded) {
		return newToolError(CodeTimeout, err)
	}
	if errors.Is(err, sql.ErrConnDone) || errors.Is(err, execution.ErrNilDatabase) || errors.Is(err, execution.ErrNilSourceDatabase) {
		return newToolError(CodeConnection, err)
	}
	return newToolError(CodeExecution, errors.Join(ErrExecution, err))
}
