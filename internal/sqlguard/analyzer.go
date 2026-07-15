package sqlguard

type Analyzer interface {
	Analyze(sql string) (Plan, error)
}

func makePlan(statements []Statement) Plan {
	plan := Plan{Statements: statements, Risk: LowRisk, ReadOnly: true}
	for _, statement := range statements {
		if statement.Kind != ReadOnly {
			plan.ReadOnly = false
		}
		if statement.Kind == DDL || (statement.requiresWhere && !statement.HasWhereClause) {
			plan.Risk = HighRisk
		}
	}
	return plan
}
