package sqlguard

import "errors"

type StatementKind string

const (
	ReadOnly StatementKind = "read_only"
	Write    StatementKind = "write"
	DDL      StatementKind = "ddl"
	Session  StatementKind = "session"
)

type Risk string

const (
	LowRisk  Risk = "low"
	HighRisk Risk = "high"
)

const DefaultMaxStatements = 50

var ErrUnsafeOrUnsupportedSQL = errors.New("unsafe or unsupported SQL")

type Statement struct {
	SQL            string
	NormalizedSQL  string
	Kind           StatementKind
	HasWhereClause bool

	requiresWhere bool
}

type Plan struct {
	Statements []Statement
	Risk       Risk
	ReadOnly   bool
}

// RiskForAtomicBatches accounts for the execution source's atomicity guarantee.
func (p Plan) RiskForAtomicBatches(atomicBatches bool) Risk {
	if !atomicBatches && len(p.Statements) > 1 {
		return HighRisk
	}
	return p.Risk
}
