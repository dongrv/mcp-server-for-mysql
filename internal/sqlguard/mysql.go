package sqlguard

import (
	"fmt"
	"strings"

	"vitess.io/vitess/go/vt/sqlparser"
)

type mysqlAnalyzer struct {
	parser *sqlparser.Parser
}

const DefaultMySQLServerVersion = "8.0.0"

func NewMySQLAnalyzer(serverVersion string) (Analyzer, error) {
	if serverVersion == "" {
		serverVersion = DefaultMySQLServerVersion
	}
	parser, err := sqlparser.New(sqlparser.Options{MySQLServerVersion: serverVersion})
	if err != nil {
		return nil, fmt.Errorf("create MySQL parser: %w", err)
	}
	return mysqlAnalyzer{parser: parser}, nil
}

func (a mysqlAnalyzer) Analyze(sql string) (Plan, error) {
	pieces, err := splitSQLStatementGroups(sql)
	if err != nil || len(pieces) == 0 || len(pieces) > DefaultMaxStatements {
		return Plan{}, unsafeSQLError(err)
	}

	statements := make([]Statement, 0, len(pieces))
	for _, piece := range pieces {
		stmt, err := a.parser.ParseStrictDDL(piece)
		if err != nil {
			return Plan{}, unsafeSQLError(err)
		}
		statement, err := classifyMySQLStatement(piece, stmt)
		if err != nil {
			return Plan{}, err
		}
		statements = append(statements, statement)
	}
	return makePlan(statements), nil
}

func classifyMySQLStatement(raw string, stmt sqlparser.Statement) (Statement, error) {
	statement := Statement{SQL: raw, NormalizedSQL: sqlparser.String(stmt)}

	switch node := stmt.(type) {
	case *sqlparser.Select:
		if hasUnsafeMySQLSelect(node) {
			return Statement{}, unsafeSQLError(nil)
		}
		statement.Kind = ReadOnly
	case *sqlparser.Union:
		if hasUnsafeMySQLSelect(node) {
			return Statement{}, unsafeSQLError(nil)
		}
		statement.Kind = ReadOnly
	case *sqlparser.Insert:
		statement.Kind = Write
	case *sqlparser.Update:
		if node.With != nil {
			return Statement{}, unsafeSQLError(nil)
		}
		statement.Kind = Write
		statement.HasWhereClause = node.Where != nil
		statement.requiresWhere = true
	case *sqlparser.Delete:
		if node.With != nil {
			return Statement{}, unsafeSQLError(nil)
		}
		statement.Kind = Write
		statement.HasWhereClause = node.Where != nil
		statement.requiresWhere = true
	case sqlparser.DDLStatement:
		if !node.IsFullyParsed() {
			return Statement{}, unsafeSQLError(nil)
		}
		statement.Kind = DDL
	case sqlparser.DBDDLStatement:
		if !node.IsFullyParsed() {
			return Statement{}, unsafeSQLError(nil)
		}
		statement.Kind = DDL
	case *sqlparser.CommentOnly,
		*sqlparser.Set,
		*sqlparser.Use,
		*sqlparser.Begin,
		*sqlparser.Commit,
		*sqlparser.Rollback,
		*sqlparser.SRollback,
		*sqlparser.Savepoint,
		*sqlparser.Release:
		return Statement{}, unsafeSQLError(nil)
	default:
		return Statement{}, unsafeSQLError(nil)
	}
	return statement, nil
}

func hasUnsafeMySQLSelect(stmt sqlparser.SelectStatement) bool {
	switch node := stmt.(type) {
	case *sqlparser.Select:
		return node.Into != nil || node.Lock != sqlparser.NoLock || hasMySQLConnectionStateFunction(node)
	case *sqlparser.Union:
		return node.Into != nil || node.Lock != sqlparser.NoLock || hasUnsafeMySQLSelect(node.Left) || hasUnsafeMySQLSelect(node.Right)
	default:
		return true
	}
}

func hasMySQLConnectionStateFunction(stmt sqlparser.SelectStatement) bool {
	unsafe := false
	_ = sqlparser.Walk(func(node sqlparser.SQLNode) (bool, error) {
		switch node := node.(type) {
		case *sqlparser.AssignmentExpr:
			unsafe = true
			return false, nil
		case *sqlparser.GTIDFuncExpr:
			switch node.Type {
			case sqlparser.WaitForExecutedGTIDSetType, sqlparser.WaitUntilSQLThreadAfterGTIDSType:
				unsafe = true
				return false, nil
			}
		}
		if _, ok := node.(*sqlparser.LockingFunc); ok {
			unsafe = true
			return false, nil
		}
		function, ok := node.(*sqlparser.FuncExpr)
		if !ok {
			return true, nil
		}
		switch strings.ToUpper(function.Name.String()) {
		case "GET_LOCK", "RELEASE_LOCK", "IS_FREE_LOCK", "IS_USED_LOCK", "SLEEP", "BENCHMARK", "LAST_INSERT_ID", "LOAD_FILE", "MASTER_POS_WAIT", "WAIT_FOR_EXECUTED_GTID_SET", "WAIT_UNTIL_SQL_THREAD_AFTER_GTIDS":
			unsafe = true
			return false, nil
		default:
			return true, nil
		}
	}, stmt)
	return unsafe
}

func unsafeSQLError(cause error) error {
	if cause == nil {
		return ErrUnsafeOrUnsupportedSQL
	}
	return fmt.Errorf("%w: %v", ErrUnsafeOrUnsupportedSQL, cause)
}
