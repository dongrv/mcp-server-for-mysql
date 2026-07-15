package execution

import (
	"testing"

	"github.com/dongrv/mcp-server-for-mysql/internal/config"
	"github.com/dongrv/mcp-server-for-mysql/internal/sqlguard"
)

func highRiskIntent() Intent {
	return Intent{
		SourceID:   "orders",
		Tool:       "execute_sql",
		Parameters: []any{int64(1)},
		Atomic:     true,
		Plan: sqlguard.Plan{
			Statements: []sqlguard.Statement{{
				NormalizedSQL:  "delete from orders where id = :v1",
				Kind:           sqlguard.Write,
				HasWhereClause: true,
			}},
			Risk: sqlguard.HighRisk,
		},
	}
}

func lowRiskPlan(sql string) sqlguard.Plan {
	return sqlguard.Plan{
		Statements: []sqlguard.Statement{{NormalizedSQL: sql, Kind: sqlguard.ReadOnly}},
		Risk:       sqlguard.LowRisk,
	}
}

func TestAuthorizeQuickModeRequiresConfirmationForHighRiskIntent(t *testing.T) {
	decision, preview, err := Authorize(config.QuickMode, highRiskIntent(), Confirmation{})
	if err != nil {
		t.Fatalf("Authorize() error = %v", err)
	}
	if decision.State != PreviewRequired {
		t.Fatalf("high-risk quick decision = %q, want %q", decision.State, PreviewRequired)
	}
	if preview.PreviewHash == "" {
		t.Fatal("preview hash must bind the complete execution intent")
	}
}

func TestAuthorizeQuickModeExecutesLowRiskIntent(t *testing.T) {
	intent := highRiskIntent()
	intent.Parameters = nil
	intent.Plan = lowRiskPlan("select id from orders")

	decision, _, err := Authorize(config.QuickMode, intent, Confirmation{})
	if err != nil {
		t.Fatalf("Authorize() error = %v", err)
	}
	if decision.State != ExecuteNow {
		t.Fatalf("low-risk quick decision = %q, want %q", decision.State, ExecuteNow)
	}
}

func TestAuthorizeQuickModeRequiresConfirmationForHighRiskDestructivePlan(t *testing.T) {
	decision, _, err := Authorize(config.QuickMode, highRiskIntent(), Confirmation{})
	if err != nil {
		t.Fatalf("Authorize() error = %v", err)
	}
	if decision.State != PreviewRequired {
		t.Fatalf("high-risk destructive plan decision = %q, want %q", decision.State, PreviewRequired)
	}
}

func TestAuthorizeRejectsConfirmationForAlteredIntent(t *testing.T) {
	intent := highRiskIntent()
	_, preview, err := Authorize(config.StrictMode, intent, Confirmation{})
	if err != nil {
		t.Fatalf("Authorize() preview error = %v", err)
	}

	intent.Parameters = nil
	intent.Plan = lowRiskPlan("select 1")
	decision, _, err := Authorize(config.StrictMode, intent, Confirmation{Confirm: true, PreviewHash: preview.PreviewHash})
	if err != nil {
		t.Fatalf("Authorize() confirmation error = %v", err)
	}
	if decision.State != PreviewMismatch {
		t.Fatalf("altered intent decision = %q, want %q", decision.State, PreviewMismatch)
	}
}

func TestBuildPreviewHashBindsSourceToolRiskAtomicStatementOrderAndParameters(t *testing.T) {
	base := highRiskIntent()
	a := mustBuildPreview(t, base)

	changed := base
	changed.SourceID = "analytics"
	b := mustBuildPreview(t, changed)
	changed = base
	changed.Tool = "update_rows"
	c := mustBuildPreview(t, changed)
	changed = base
	changed.Plan.Risk = sqlguard.LowRisk
	d := mustBuildPreview(t, changed)
	changed = base
	changed.Atomic = false
	e := mustBuildPreview(t, changed)
	changed = base
	changed.Plan.Statements = []sqlguard.Statement{
		{NormalizedSQL: "select 1", Kind: sqlguard.ReadOnly},
		{NormalizedSQL: "delete from orders where id = :v1", Kind: sqlguard.Write, HasWhereClause: true},
	}
	f := mustBuildPreview(t, changed)
	changed = base
	changed.Parameters = []any{int64(2)}
	g := mustBuildPreview(t, changed)

	for label, candidate := range map[string]Preview{
		"source": b, "tool": c, "risk": d, "atomic": e, "statement order": f, "parameters": g,
	} {
		if a.PreviewHash == candidate.PreviewHash {
			t.Fatalf("preview hash did not bind %s", label)
		}
	}
}

func TestAuthorizeAcceptsMatchingConfirmationForCompleteIntent(t *testing.T) {
	intent := highRiskIntent()
	_, preview, err := Authorize(config.QuickMode, intent, Confirmation{})
	if err != nil {
		t.Fatalf("Authorize() preview error = %v", err)
	}

	decision, confirmedPreview, err := Authorize(config.QuickMode, intent, Confirmation{Confirm: true, PreviewHash: preview.PreviewHash})
	if err != nil {
		t.Fatalf("Authorize() confirmation error = %v", err)
	}
	if decision.State != ExecuteNow {
		t.Fatalf("matching confirmation decision = %q, want %q", decision.State, ExecuteNow)
	}
	if confirmedPreview.PreviewHash != preview.PreviewHash {
		t.Fatal("confirmation must be checked against the request's canonical preview")
	}
}

func TestAuthorizeStrictModeRequiresConfirmationForLowRiskIntent(t *testing.T) {
	intent := highRiskIntent()
	intent.Parameters = nil
	intent.Plan = lowRiskPlan("select id from orders")

	decision, _, err := Authorize(config.StrictMode, intent, Confirmation{})
	if err != nil {
		t.Fatalf("Authorize() error = %v", err)
	}
	if decision.State != PreviewRequired {
		t.Fatalf("strict low-risk decision = %q, want %q", decision.State, PreviewRequired)
	}
}

func TestAuthorizeFailsClosedForUnknownMode(t *testing.T) {
	decision, _, err := Authorize(config.Mode("unknown"), highRiskIntent(), Confirmation{})
	if err == nil {
		t.Fatal("Authorize() error = nil, want invalid mode error")
	}
	if decision.State != PolicyDenied {
		t.Fatalf("unknown-mode decision = %q, want %q", decision.State, PolicyDenied)
	}
}

func TestAuthorizeRejectsUnsupportedJSONParameters(t *testing.T) {
	intent := highRiskIntent()
	intent.Parameters = []any{make(chan int)}

	decision, preview, err := Authorize(config.QuickMode, intent, Confirmation{Confirm: true, PreviewHash: "anything"})
	if err == nil {
		t.Fatal("Authorize() error = nil, want canonicalization error")
	}
	if decision.State != PolicyDenied {
		t.Fatalf("unsupported-parameter decision = %q, want %q", decision.State, PolicyDenied)
	}
	if preview.PreviewHash != "" {
		t.Fatal("invalid preview must not contain an executable confirmation hash")
	}
}

func mustBuildPreview(t *testing.T, intent Intent) Preview {
	t.Helper()
	preview, err := BuildPreview(intent)
	if err != nil {
		t.Fatalf("BuildPreview() error = %v", err)
	}
	return preview
}
