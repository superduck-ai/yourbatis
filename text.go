package yourbatis

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

// Identifier is an explicitly typed SQL identifier. Prefer NewIdentifier for
// values coming from configuration or user input.
type Identifier string

func NewIdentifier(value string) (Identifier, error) {
	if value == "" {
		return "", fmt.Errorf("yourbatis: identifier is empty")
	}
	for _, part := range strings.Split(value, ".") {
		if !validIdentifierPart(part) {
			return "", fmt.Errorf("yourbatis: invalid identifier %q", value)
		}
	}
	return Identifier(value), nil
}

func MustIdentifier(value string) Identifier {
	identifier, err := NewIdentifier(value)
	if err != nil {
		panic(err)
	}
	return identifier
}

// Fragment is SQL text that the caller has explicitly marked as trusted.
// Values originating from users should normally use #{} binding instead.
type Fragment string

func TrustedFragment(value string) Fragment {
	return Fragment(value)
}

func AppendSigned(builder *Builder, value int64) {
	builder.Text(strconv.FormatInt(value, 10))
}

func AppendUnsigned(builder *Builder, value uint64) {
	builder.Text(strconv.FormatUint(value, 10))
}

func AppendFloat(builder *Builder, value float64) {
	builder.Text(strconv.FormatFloat(value, 'g', -1, 64))
}

func AppendBool(builder *Builder, value bool) {
	builder.Text(strconv.FormatBool(value))
}

func AppendIdentifier(builder *Builder, dialect Dialect, identifier Identifier) {
	builder.Text(dialect.QuoteIdentifier(identifier))
}

func (dialect Dialect) QuoteIdentifier(identifier Identifier) string {
	quote := `"`
	if dialect == DialectMySQL {
		quote = "`"
	}

	parts := strings.Split(string(identifier), ".")
	for index, part := range parts {
		parts[index] = quote + strings.ReplaceAll(part, quote, quote+quote) + quote
	}
	return strings.Join(parts, ".")
}

func validIdentifierPart(value string) bool {
	for index, current := range value {
		if index == 0 {
			if current != '_' && !unicode.IsLetter(current) {
				return false
			}
			continue
		}
		if current != '_' && !unicode.IsLetter(current) && !unicode.IsDigit(current) {
			return false
		}
	}
	return value != ""
}
