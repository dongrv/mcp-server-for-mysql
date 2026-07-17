package tools

import (
	"errors"
	"fmt"

	"github.com/dongrv/mcp-server-for-mysql/internal/execution"
)

const (
	StateExecuted             = "executed"
	StateConfirmationRequired = "confirmation_required"
	StatePreviewMismatch      = "preview_mismatch"
	StateError                = "error"
)

var (
	ErrInvalidInput        = errors.New("invalid input")
	ErrReadOnlySQLRequired = errors.New("query requires read-only SQL")
	ErrMutationSQLRequired = errors.New("execute_sql requires non-read SQL")
	ErrUnsafeSQL           = errors.New("unsafe or unsupported SQL")
	ErrConfirmation        = errors.New("confirmation required")
	ErrPreviewMismatch     = errors.New("preview hash does not match request")
	ErrUnsupported         = errors.New("unsupported capability")
	ErrExecution           = errors.New("execution failed")
)

// ErrorCode is a stable, machine-readable error classification for MCP callers.
type ErrorCode string

const (
	CodeInvalidInput    ErrorCode = "invalid_input"
	CodeUnknownSource   ErrorCode = "unknown_source"
	CodeUnsafeSQL       ErrorCode = "unsafe_sql"
	CodeConfirmation    ErrorCode = "confirmation_required"
	CodePreviewMismatch ErrorCode = "preview_mismatch"
	CodeUnsupported     ErrorCode = "unsupported_capability"
	CodeTimeout         ErrorCode = "timeout"
	CodeConnection      ErrorCode = "connection_failure"
	CodeExecution       ErrorCode = "execution_failure"
)

type ToolError struct {
	Code ErrorCode `json:"code"`
	Err  error
}

func (e *ToolError) Error() string { return string(e.Code) }
func (e *ToolError) Unwrap() error { return e.Err }

// Response is the common envelope returned by all state-changing and query tools.
type Response struct {
	RequestID string                   `json:"request_id"`
	State     string                   `json:"state"`
	Preview   *execution.Preview       `json:"preview,omitempty"`
	Query     *execution.QueryResult   `json:"query,omitempty"`
	Execution *execution.ExecuteResult `json:"execution,omitempty"`
	Data      any                      `json:"data,omitempty"`
	Error     *ErrorCode               `json:"error_code,omitempty"`
}

func responseForError(requestID string, err error) Response {
	code := ErrorCodeFor(err)
	return Response{RequestID: requestID, State: StateError, Error: &code}
}

func ErrorCodeFor(err error) ErrorCode {
	var typed *ToolError
	if errors.As(err, &typed) {
		return typed.Code
	}
	return CodeExecution
}

func newToolError(code ErrorCode, cause error) error {
	if cause == nil {
		cause = fmt.Errorf("%s", code)
	}
	return &ToolError{Code: code, Err: cause}
}
