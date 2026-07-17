package operation

import (
	"fmt"
	"strings"

	"github.com/dongrv/mcp-server-for-mysql/internal/database"
)

const (
	maxMySQLVarcharLength = 65535
	maxMySQLDecimal       = 65
	maxClickHouseDecimal  = 76
)

func quote(dialect database.Dialect, identifier string) (string, error) {
	return dialect.QuoteIdentifier(identifier)
}

func quotedColumns(dialect database.Dialect, columns []string) ([]string, error) {
	if len(columns) == 0 {
		return nil, fmt.Errorf("at least one column is required")
	}

	seen := make(map[string]struct{}, len(columns))
	quoted := make([]string, 0, len(columns))
	for _, column := range columns {
		if _, ok := seen[column]; ok {
			return nil, fmt.Errorf("duplicate column %q", column)
		}
		seen[column] = struct{}{}
		name, err := quote(dialect, column)
		if err != nil {
			return nil, err
		}
		quoted = append(quoted, name)
	}
	return quoted, nil
}

func validateNoDimensions(spec ColumnSpec) error {
	if spec.Length != 0 || spec.Precision != 0 || spec.Scale != 0 {
		return fmt.Errorf("column kind %q does not accept dimensions", spec.Kind)
	}
	return nil
}

func mysqlColumn(dialect database.Dialect, spec ColumnSpec) (string, error) {
	name, err := quote(dialect, spec.Name)
	if err != nil {
		return "", err
	}

	var typeSQL string
	switch spec.Kind {
	case "int":
		err = validateNoDimensions(spec)
		typeSQL = "INT"
	case "bigint":
		err = validateNoDimensions(spec)
		typeSQL = "BIGINT"
	case "varchar":
		if spec.Length < 1 || spec.Length > maxMySQLVarcharLength || spec.Precision != 0 || spec.Scale != 0 {
			err = fmt.Errorf("varchar length must be between 1 and %d", maxMySQLVarcharLength)
		} else {
			typeSQL = fmt.Sprintf("VARCHAR(%d)", spec.Length)
		}
	case "text":
		err = validateNoDimensions(spec)
		typeSQL = "TEXT"
	case "decimal":
		if spec.Length != 0 || spec.Precision < 1 || spec.Precision > maxMySQLDecimal || spec.Scale < 0 || spec.Scale > spec.Precision {
			err = fmt.Errorf("decimal precision must be between 1 and %d and scale must be between 0 and precision", maxMySQLDecimal)
		} else {
			typeSQL = fmt.Sprintf("DECIMAL(%d,%d)", spec.Precision, spec.Scale)
		}
	case "boolean":
		err = validateNoDimensions(spec)
		typeSQL = "BOOLEAN"
	case "date":
		err = validateNoDimensions(spec)
		typeSQL = "DATE"
	case "datetime":
		err = validateNoDimensions(spec)
		typeSQL = "DATETIME"
	case "timestamp":
		err = validateNoDimensions(spec)
		typeSQL = "TIMESTAMP"
	default:
		return "", fmt.Errorf("unsupported MySQL column kind %q", spec.Kind)
	}
	if err != nil {
		return "", err
	}

	nullability := "NOT NULL"
	if spec.Nullable {
		nullability = "NULL"
	}
	return name + " " + typeSQL + " " + nullability, nil
}

func clickHouseColumn(dialect database.Dialect, spec ColumnSpec) (string, error) {
	name, err := quote(dialect, spec.Name)
	if err != nil {
		return "", err
	}

	var typeSQL string
	switch spec.Kind {
	case "int64":
		err = validateNoDimensions(spec)
		typeSQL = "Int64"
	case "uint64":
		err = validateNoDimensions(spec)
		typeSQL = "UInt64"
	case "string":
		err = validateNoDimensions(spec)
		typeSQL = "String"
	case "decimal":
		if spec.Length != 0 || spec.Precision < 1 || spec.Precision > maxClickHouseDecimal || spec.Scale < 0 || spec.Scale > spec.Precision {
			err = fmt.Errorf("decimal precision must be between 1 and %d and scale must be between 0 and precision", maxClickHouseDecimal)
		} else {
			typeSQL = fmt.Sprintf("Decimal(%d,%d)", spec.Precision, spec.Scale)
		}
	case "bool":
		err = validateNoDimensions(spec)
		typeSQL = "Bool"
	case "date":
		err = validateNoDimensions(spec)
		typeSQL = "Date"
	case "datetime":
		err = validateNoDimensions(spec)
		typeSQL = "DateTime"
	default:
		return "", fmt.Errorf("unsupported ClickHouse column kind %q", spec.Kind)
	}
	if err != nil {
		return "", err
	}
	if spec.Nullable {
		typeSQL = "Nullable(" + typeSQL + ")"
	}
	return name + " " + typeSQL, nil
}

func buildColumns(dialect database.Dialect, specs []ColumnSpec, column func(database.Dialect, ColumnSpec) (string, error)) ([]string, error) {
	if len(specs) == 0 {
		return nil, fmt.Errorf("at least one column is required")
	}
	seen := make(map[string]struct{}, len(specs))
	columns := make([]string, 0, len(specs))
	for _, spec := range specs {
		if _, ok := seen[spec.Name]; ok {
			return nil, fmt.Errorf("duplicate column %q", spec.Name)
		}
		seen[spec.Name] = struct{}{}
		built, err := column(dialect, spec)
		if err != nil {
			return nil, err
		}
		columns = append(columns, built)
	}
	return columns, nil
}

func join(columns []string) string {
	return strings.Join(columns, ", ")
}
