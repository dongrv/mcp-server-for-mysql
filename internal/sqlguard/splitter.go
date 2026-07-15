package sqlguard

import (
	"errors"
	"strings"
)

var errInvalidStatementGroups = errors.New("invalid SQL statement groups")

type sqlScanState uint8

const (
	scanNormal sqlScanState = iota
	scanSingleQuote
	scanDoubleQuote
	scanBacktick
	scanLineComment
	scanBlockComment
)

// splitSQLStatementGroups validates semicolon-delimited groups before a parser
// can discard raw empty or comment-only input.
func splitSQLStatementGroups(sql string) ([]string, error) {
	groups := make([]string, 0, 1)
	start := 0
	state := scanNormal

	for i := 0; i < len(sql); i++ {
		switch state {
		case scanNormal:
			switch sql[i] {
			case '\'':
				state = scanSingleQuote
			case '"':
				state = scanDoubleQuote
			case '`':
				state = scanBacktick
			case '-':
				if startsLineComment(sql, i) {
					state = scanLineComment
				}
			case '#':
				state = scanLineComment
			case '/':
				if i+1 < len(sql) && sql[i+1] == '*' {
					state = scanBlockComment
				}
			case ';':
				groups = append(groups, sql[start:i])
				start = i + 1
			}
		case scanSingleQuote:
			if sql[i] == '\\' && i+1 < len(sql) {
				i++
			} else if sql[i] == '\'' {
				if i+1 < len(sql) && sql[i+1] == '\'' {
					i++
				} else {
					state = scanNormal
				}
			}
		case scanDoubleQuote:
			if sql[i] == '\\' && i+1 < len(sql) {
				i++
			} else if sql[i] == '"' {
				if i+1 < len(sql) && sql[i+1] == '"' {
					i++
				} else {
					state = scanNormal
				}
			}
		case scanBacktick:
			if sql[i] == '\\' && i+1 < len(sql) {
				i++
			} else if sql[i] == '`' {
				if i+1 < len(sql) && sql[i+1] == '`' {
					i++
				} else {
					state = scanNormal
				}
			}
		case scanLineComment:
			if sql[i] == '\n' || sql[i] == '\r' {
				state = scanNormal
			}
		case scanBlockComment:
			if sql[i] == '*' && i+1 < len(sql) && sql[i+1] == '/' {
				i++
				state = scanNormal
			}
		}
	}
	groups = append(groups, sql[start:])

	statements := make([]string, 0, len(groups))
	for i, group := range groups {
		if statementGroupHasCode(group) {
			statements = append(statements, strings.TrimSpace(group))
			continue
		}
		if i == len(groups)-1 && len(groups) > 1 && strings.TrimSpace(group) == "" {
			continue
		}
		return nil, errInvalidStatementGroups
	}
	if len(statements) == 0 {
		return nil, errInvalidStatementGroups
	}
	return statements, nil
}

func startsLineComment(sql string, index int) bool {
	if index+1 >= len(sql) || sql[index+1] != '-' {
		return false
	}
	if index+2 >= len(sql) {
		return true
	}
	switch sql[index+2] {
	case ' ', '\t', '\n', '\r', '\f':
		return true
	default:
		return false
	}
}

func statementGroupHasCode(sql string) bool {
	state := scanNormal
	for i := 0; i < len(sql); i++ {
		switch state {
		case scanNormal:
			switch sql[i] {
			case ' ', '\t', '\n', '\r', '\f':
				continue
			case '-':
				if startsLineComment(sql, i) {
					state = scanLineComment
					continue
				}
			case '#':
				state = scanLineComment
				continue
			case '/':
				if i+1 < len(sql) && sql[i+1] == '*' {
					state = scanBlockComment
					continue
				}
			}
			return true
		case scanLineComment:
			if sql[i] == '\n' || sql[i] == '\r' {
				state = scanNormal
			}
		case scanBlockComment:
			if sql[i] == '*' && i+1 < len(sql) && sql[i+1] == '/' {
				i++
				state = scanNormal
			}
		}
	}
	return false
}
