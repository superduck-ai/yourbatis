package generator

import (
	"go/ast"
	"go/parser"
	"strings"
	"testing"

	"github.com/superduck-ai/yourbatis/internal/ir"
)

func TestRejectsNullComparisonForNonNilableTypes(t *testing.T) {
	testCases := []struct {
		name           string
		valueType      string
		underlyingType string
	}{
		{name: "string", valueType: "string"},
		{name: "integer", valueType: "int64"},
		{name: "boolean", valueType: "bool"},
		{name: "struct", valueType: "struct{}"},
		{name: "array", valueType: "[1]int64"},
		{name: "named string", valueType: "Name", underlyingType: "string"},
		{name: "named struct", valueType: "Record", underlyingType: "struct{}"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			for _, expression := range []string{
				"value != null",
				"value == null",
				"null != value",
				"null == value",
			} {
				t.Run(strings.ReplaceAll(expression, " ", "_"), func(t *testing.T) {
					err := validateNullComparison(
						t,
						testCase.valueType,
						testCase.underlyingType,
						expression,
					)
					expected := "type " + testCase.valueType + " cannot be compared with nil"
					if err == nil || !strings.Contains(err.Error(), expected) {
						t.Fatalf("error = %v, want message containing %q", err, expected)
					}
				})
			}
		})
	}
}

func TestAcceptsNullComparisonForNilableTypes(t *testing.T) {
	testCases := []struct {
		name           string
		valueType      string
		underlyingType string
	}{
		{name: "pointer", valueType: "*string"},
		{name: "slice", valueType: "[]int64"},
		{name: "map", valueType: "map[string]int64"},
		{name: "channel", valueType: "chan int64"},
		{name: "function", valueType: "func()"},
		{name: "any", valueType: "any"},
		{name: "error", valueType: "error"},
		{name: "anonymous interface", valueType: "interface{}"},
		{name: "named pointer", valueType: "OptionalName", underlyingType: "*string"},
		{name: "named slice", valueType: "IDs", underlyingType: "[]int64"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			for _, expression := range []string{"value != null", "value == null"} {
				t.Run(strings.ReplaceAll(expression, " ", "_"), func(t *testing.T) {
					if err := validateNullComparison(
						t,
						testCase.valueType,
						testCase.underlyingType,
						expression,
					); err != nil {
						t.Fatalf("validate %q for %s: %v", expression, testCase.valueType, err)
					}
				})
			}
		})
	}
}

func validateNullComparison(
	t *testing.T,
	valueType string,
	underlyingType string,
	expression string,
) error {
	t.Helper()
	parsedType, err := parser.ParseExpr(valueType)
	if err != nil {
		t.Fatal(err)
	}
	goPackage := &GoPackage{
		Structs: map[string]*StructInfo{},
		Types:   map[string]ast.Expr{},
	}
	if underlyingType != "" {
		goPackage.Types[valueType] = mustTypeExpr2(t, underlyingType)
	}
	method := &MapperMethod{
		Name: "Run",
		Params: []GoValue{
			{Name: "ctx", Type: "context.Context", Expr: mustTypeExpr2(t, "context.Context")},
			{Name: "value", Type: valueType, Expr: parsedType},
		},
	}
	return validateStatementExpressions(
		goPackage,
		method,
		&ir.Statement{Body: []ir.Node{
			&ir.If{Test: expression},
		}},
	)
}

func mustTypeExpr2(t *testing.T, value string) ast.Expr {
	t.Helper()
	parsed, err := parser.ParseExpr(value)
	if err != nil {
		t.Fatal(err)
	}
	return parsed
}
