package generator

import (
	"fmt"
	"go/ast"
	goparser "go/parser"
	"go/token"
	"strconv"
	"strings"
	"unicode"
)

type compiledExpression struct {
	AST    ast.Expr
	Source string
	Mode   expressionMode
}

type expressionMode uint8

const (
	expressionValue expressionMode = iota
	expressionCondition
	expressionCollection
	expressionText
)

func translateExpression(expression string) (string, error) {
	compiled, err := compileExpression(expression)
	if err != nil {
		return "", err
	}
	return compiled.Source, nil
}

func compileExpression(expression string) (compiledExpression, error) {
	return compileExpressionForMode(expression, expressionValue)
}

// compileExpression keeps Go as the primary expression language. MyBatis
// compatibility syntax is only parsed when the normalized expression contains
// one of the explicitly supported legacy constructs.
func compileExpressionForMode(
	expression string,
	mode expressionMode,
) (compiledExpression, error) {
	normalized, err := normalizeQuotedStrings(expression)
	if err != nil {
		return compiledExpression{}, err
	}
	normalized = strings.TrimSpace(normalized)
	if normalized == "" {
		return compiledExpression{}, fmt.Errorf("empty dynamic SQL expression")
	}

	if !needsMyBatisCompatibility(normalized) {
		parsed, parseErr := goparser.ParseExpr(normalized)
		if parseErr == nil {
			if err := validateNativeExpressionCalls(parsed); err != nil {
				return compiledExpression{}, err
			}
			source, err := expressionString(parsed)
			if err != nil {
				return compiledExpression{}, err
			}
			return compiledExpression{AST: parsed, Source: source, Mode: mode}, nil
		}
	}

	parsed, err := parseMyBatisExpression(normalized)
	if err != nil {
		return compiledExpression{}, fmt.Errorf(
			"cannot translate expression %q to Go: %w",
			expression,
			err,
		)
	}
	lowered, err := lowerMyBatisExpression(parsed)
	if err != nil {
		return compiledExpression{}, fmt.Errorf(
			"cannot translate expression %q to Go: %w",
			expression,
			err,
		)
	}
	source, err := expressionString(lowered)
	if err != nil {
		return compiledExpression{}, err
	}
	if _, err := goparser.ParseExpr(source); err != nil {
		return compiledExpression{}, fmt.Errorf(
			"cannot translate expression %q to Go (%q): %w",
			expression,
			source,
			err,
		)
	}
	return compiledExpression{AST: lowered, Source: source, Mode: mode}, nil
}

func validateNativeExpressionCalls(expression ast.Expr) error {
	var validationErr error
	ast.Inspect(expression, func(node ast.Node) bool {
		if validationErr != nil {
			return false
		}
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		selector, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		name := selector.Sel.Name
		if name != "" && unicode.IsLower(rune(name[0])) {
			validationErr = fmt.Errorf(
				"method %q is not an exported Go method or a supported MyBatis adapter",
				name,
			)
			return false
		}
		return true
	})
	return validationErr
}

func normalizeQuotedStrings(expression string) (string, error) {
	var result strings.Builder
	for offset := 0; offset < len(expression); {
		switch expression[offset] {
		case '"':
			end, err := scanQuoted(expression, offset, '"')
			if err != nil {
				return "", err
			}
			result.WriteString(expression[offset:end])
			offset = end
		case '\'':
			end, err := scanQuoted(expression, offset, '\'')
			if err != nil {
				return "", err
			}
			if decoded, unquoteErr := strconv.Unquote(expression[offset:end]); unquoteErr == nil &&
				len([]rune(decoded)) == 1 {
				result.WriteString(expression[offset:end])
				offset = end
				continue
			}
			content := expression[offset+1 : end-1]
			unescaped, err := unescapeSingleQuoted(content)
			if err != nil {
				return "", err
			}
			result.WriteString(strconv.Quote(unescaped))
			offset = end
		default:
			result.WriteByte(expression[offset])
			offset++
		}
	}
	return result.String(), nil
}

func unescapeSingleQuoted(s string) (string, error) {
	var builder strings.Builder
	for i := 0; i < len(s); {
		if s[i] == '\\' {
			if i+1 >= len(s) {
				return "", fmt.Errorf("bad escape sequence at the end of string")
			}
			switch s[i+1] {
			case 'n':
				builder.WriteByte('\n')
			case 't':
				builder.WriteByte('\t')
			case 'r':
				builder.WriteByte('\r')
			case 'b':
				builder.WriteByte('\b')
			case 'f':
				builder.WriteByte('\f')
			case 'v':
				builder.WriteByte('\v')
			case 'a':
				builder.WriteByte('\a')
			case '\\':
				builder.WriteByte('\\')
			case '\'':
				builder.WriteByte('\'')
			case '"':
				builder.WriteByte('"')
			default:
				builder.WriteByte(s[i+1])
			}
			i += 2
		} else {
			builder.WriteByte(s[i])
			i++
		}
	}
	return builder.String(), nil
}

func scanQuoted(value string, offset int, quote byte) (int, error) {
	escaped := false
	for index := offset + 1; index < len(value); index++ {
		if escaped {
			escaped = false
			continue
		}
		if value[index] == '\\' {
			escaped = true
			continue
		}
		if value[index] == quote {
			return index + 1, nil
		}
	}
	return 0, fmt.Errorf("unterminated quoted string in expression %q", value)
}

func rewriteExpressionIdentifiers(expression ast.Expr, aliases map[string]string) {
	if len(aliases) == 0 {
		return
	}
	ast.Inspect(expression, func(node ast.Node) bool {
		identifier, ok := node.(*ast.Ident)
		if !ok {
			return true
		}
		if replacement := aliases[identifier.Name]; replacement != "" {
			identifier.Name = replacement
		}
		return true
	})
}

func tokenForBinaryOperator(operator string) (token.Token, bool) {
	switch operator {
	case "==", "eq":
		return token.EQL, true
	case "!=", "ne", "neq":
		return token.NEQ, true
	case "<", "lt":
		return token.LSS, true
	case "<=", "le", "lte":
		return token.LEQ, true
	case ">", "gt":
		return token.GTR, true
	case ">=", "ge", "gte":
		return token.GEQ, true
	case "&&", "and":
		return token.LAND, true
	case "||", "or":
		return token.LOR, true
	case "+":
		return token.ADD, true
	case "-":
		return token.SUB, true
	case "*":
		return token.MUL, true
	case "/":
		return token.QUO, true
	case "%":
		return token.REM, true
	default:
		return token.ILLEGAL, false
	}
}
