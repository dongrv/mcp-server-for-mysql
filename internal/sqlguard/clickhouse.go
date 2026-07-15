package sqlguard

import (
	"strings"

	clickhouse "github.com/AfterShip/clickhouse-sql-parser/parser"
)

type clickHouseAnalyzer struct{}

func NewClickHouseAnalyzer() Analyzer {
	return clickHouseAnalyzer{}
}

func (clickHouseAnalyzer) Analyze(sql string) (Plan, error) {
	pieces, err := splitSQLStatementGroups(sql)
	if err != nil || len(pieces) == 0 || len(pieces) > DefaultMaxStatements {
		return Plan{}, unsafeSQLError(err)
	}

	statements := make([]Statement, 0, len(pieces))
	for _, piece := range pieces {
		parsed, err := clickhouse.NewParser(piece).ParseStmts()
		if err != nil || len(parsed) != 1 {
			return Plan{}, unsafeSQLError(err)
		}
		statement, err := classifyClickHouseStatement(piece, parsed[0])
		if err != nil {
			return Plan{}, err
		}
		statements = append(statements, statement)
	}
	return makePlan(statements), nil
}

func classifyClickHouseStatement(raw string, stmt clickhouse.Expr) (Statement, error) {
	statement := Statement{SQL: raw, NormalizedSQL: clickhouse.Format(stmt)}

	switch node := stmt.(type) {
	case *clickhouse.SelectQuery:
		if !validClickHouseSelect(node) {
			return Statement{}, unsafeSQLError(nil)
		}
		statement.Kind = ReadOnly
	case *clickhouse.InsertStmt:
		statement.Kind = Write
	case *clickhouse.DeleteClause:
		statement.Kind = Write
		statement.HasWhereClause = node.WhereExpr != nil
		statement.requiresWhere = true
	case *clickhouse.AlterTable:
		statement.Kind = DDL
	case *clickhouse.CreateDatabase,
		*clickhouse.CreateTable,
		*clickhouse.CreateMaterializedView,
		*clickhouse.CreateView,
		*clickhouse.CreateFunction,
		*clickhouse.CreateRole,
		*clickhouse.CreateUser,
		*clickhouse.CreateLiveView,
		*clickhouse.CreateDictionary,
		*clickhouse.CreateNamedCollection,
		*clickhouse.AlterRole,
		*clickhouse.DropDatabase,
		*clickhouse.DropStmt,
		*clickhouse.DropUserOrRole,
		*clickhouse.TruncateTable,
		*clickhouse.RenameStmt:
		statement.Kind = DDL
	case *clickhouse.SetStmt, *clickhouse.UseStmt:
		return Statement{}, unsafeSQLError(nil)
	default:
		return Statement{}, unsafeSQLError(nil)
	}
	return statement, nil
}

func validClickHouseSelect(query *clickhouse.SelectQuery) bool {
	if len(query.SelectItems) == 0 {
		return false
	}
	if hasClickHouseTableFunction(query) {
		return false
	}
	for _, item := range query.SelectItems {
		ident, ok := item.Expr.(*clickhouse.Ident)
		if !ok || ident.QuoteType != clickhouse.Unquoted {
			continue
		}
		switch strings.ToUpper(ident.Name) {
		case "FROM", "WHERE", "PREWHERE", "GROUP", "HAVING", "ORDER", "LIMIT", "SETTINGS", "FORMAT", "UNION", "EXCEPT", "INTERSECT", "INTO":
			return false
		}
	}
	return true
}

func hasClickHouseTableFunction(query *clickhouse.SelectQuery) bool {
	return !clickhouse.Walk(query, func(node clickhouse.Expr) bool {
		_, isTableFunction := node.(*clickhouse.TableFunctionExpr)
		return !isTableFunction
	})
}
