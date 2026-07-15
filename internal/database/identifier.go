package database

import (
	"errors"
	"fmt"
)

// ErrInvalidIdentifier is returned when input is not a single safe SQL identifier.
var ErrInvalidIdentifier = errors.New("invalid identifier")

func validateIdentifier(identifier string) error {
	if identifier == "" {
		return ErrInvalidIdentifier
	}
	for i := 0; i < len(identifier); i++ {
		c := identifier[i]
		if c > 0x7f || (i == 0 && !isIdentifierStart(c)) || (i > 0 && !isIdentifierPart(c)) {
			return fmt.Errorf("%w: %q", ErrInvalidIdentifier, identifier)
		}
	}
	return nil
}

func quoteIdentifier(identifier string) (string, error) {
	if err := validateIdentifier(identifier); err != nil {
		return "", err
	}
	return "`" + identifier + "`", nil
}

func isIdentifierStart(c byte) bool {
	return c == '_' || (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z')
}

func isIdentifierPart(c byte) bool {
	return isIdentifierStart(c) || (c >= '0' && c <= '9')
}
