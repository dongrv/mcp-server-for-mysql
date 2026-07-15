// Package execution applies the stateless execution policy and database limits.
package execution

import "github.com/dongrv/mcp-server-for-mysql/internal/sqlguard"

// Confirmation is supplied by the caller when it asks to execute a previewed
// operation. It intentionally contains no user identity or server-side state.
type Confirmation struct {
	Confirm     bool   `json:"confirm"`
	PreviewHash string `json:"preview_hash,omitempty"`
}

// DecisionState describes whether an operation may proceed.
type DecisionState string

const (
	// ExecuteNow permits execution of the already analyzed operation.
	ExecuteNow DecisionState = "execute"
	// PreviewRequired tells the caller to obtain and present a preview first.
	PreviewRequired DecisionState = "confirmation_required"
	// PreviewMismatch denies an attempted confirmation with a different hash.
	PreviewMismatch DecisionState = "preview_mismatch"
	// PolicyDenied denies an operation which cannot be authorized safely.
	PolicyDenied DecisionState = "policy_denied"
)

// Decision is the result of applying the confirmation policy.
type Decision struct {
	State DecisionState `json:"state"`
}

// Intent contains every execution-relevant field that is bound into a
// stateless confirmation hash. Callers must authorize the same intent they
// intend to execute.
type Intent struct {
	SourceID   string
	Tool       string
	Parameters []any
	Atomic     bool
	Plan       sqlguard.Plan
}

// Preview is the complete, caller-visible description of an operation that
// requires confirmation. PreviewHash binds every execution-relevant field.
type Preview struct {
	State       string   `json:"state"`
	SQL         []string `json:"sql"`
	Risk        string   `json:"risk"`
	Atomic      bool     `json:"atomic"`
	PreviewHash string   `json:"preview_hash"`
}
