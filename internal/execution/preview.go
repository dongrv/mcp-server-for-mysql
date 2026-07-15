package execution

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/dongrv/mcp-server-for-mysql/internal/sqlguard"
)

type previewEnvelope struct {
	SourceID   string   `json:"source_id"`
	Tool       string   `json:"tool"`
	SQL        []string `json:"sql"`
	Parameters []any    `json:"parameters"`
	Risk       string   `json:"risk"`
	Atomic     bool     `json:"atomic"`
}

// BuildPreview creates a deterministic, stateless confirmation artifact from
// a dialect-analyzed SQL plan. SQL and risk are derived from the plan instead
// of caller-supplied fields. Invalid intents produce no hash, so they cannot
// authorize execution.
func BuildPreview(intent Intent) (Preview, error) {
	canonicalSQL, err := canonicalPlanSQL(intent.Plan)
	if err != nil {
		return invalidPreview(intent), err
	}
	if strings.TrimSpace(intent.SourceID) == "" {
		return invalidPreview(intent), fmt.Errorf("source ID is required")
	}
	if strings.TrimSpace(intent.Tool) == "" {
		return invalidPreview(intent), fmt.Errorf("tool is required")
	}
	effectiveRisk := intent.Plan.RiskForAtomicBatches(intent.Atomic)
	if effectiveRisk != sqlguard.LowRisk && effectiveRisk != sqlguard.HighRisk {
		return invalidPreview(intent), fmt.Errorf("invalid SQL risk")
	}

	parameters := intent.Parameters
	if parameters == nil {
		parameters = []any{}
	}

	envelope := previewEnvelope{
		SourceID:   intent.SourceID,
		Tool:       intent.Tool,
		SQL:        canonicalSQL,
		Parameters: parameters,
		Risk:       string(effectiveRisk),
		Atomic:     intent.Atomic,
	}
	encoded, err := json.Marshal(envelope)
	if err != nil {
		return invalidPreview(intent), fmt.Errorf("canonicalize execution parameters: %w", err)
	}
	sum := sha256.Sum256(encoded)

	return Preview{
		State:       string(PreviewRequired),
		SQL:         canonicalSQL,
		Risk:        string(effectiveRisk),
		Atomic:      intent.Atomic,
		PreviewHash: hex.EncodeToString(sum[:]),
	}, nil
}

func canonicalPlanSQL(plan sqlguard.Plan) ([]string, error) {
	if len(plan.Statements) == 0 {
		return nil, fmt.Errorf("at least one SQL statement is required")
	}

	canonical := make([]string, len(plan.Statements))
	for i, statement := range plan.Statements {
		normalized := strings.TrimSpace(statement.NormalizedSQL)
		if normalized == "" {
			return nil, fmt.Errorf("SQL statement %d is empty", i+1)
		}
		canonical[i] = normalized
	}
	return canonical, nil
}

func invalidPreview(intent Intent) Preview {
	canonicalSQL := make([]string, len(intent.Plan.Statements))
	for i, statement := range intent.Plan.Statements {
		canonicalSQL[i] = strings.TrimSpace(statement.NormalizedSQL)
	}
	return Preview{
		State:  string(PolicyDenied),
		SQL:    canonicalSQL,
		Risk:   string(intent.Plan.RiskForAtomicBatches(intent.Atomic)),
		Atomic: intent.Atomic,
	}
}
