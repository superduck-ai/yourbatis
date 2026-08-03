package generator

import (
	"go/ast"
	goparser "go/parser"
	"strings"
	"testing"

	"github.com/superduck-ai/yourbatis/internal/ir"
)

func TestValidateStatementExpressionsAcceptsScopedGoExpressions(t *testing.T) {
	goPackage, method := validationFixture(t)
	statement := &ir.Statement{Body: []ir.Node{
		&ir.Bind{
			Name:       "keyword",
			Expression: "params.Keyword",
		},
		&ir.If{
			Test: "params.IDs != null and params.IDs.size > 0",
			Body: []ir.Node{
				&ir.Foreach{
					Collection: "params.IDs",
					Index:      "index",
					Item:       "id",
					Body: []ir.Node{
						&ir.If{
							Test: "index % 2 == 0",
							Body: []ir.Node{
								&ir.Parameter{Path: "id"},
							},
						},
					},
				},
			},
		},
		&ir.Foreach{
			Collection: "params.Groups",
			Index:      "group",
			Item:       "ids",
			Body: []ir.Node{
				&ir.Foreach{
					Collection: "ids",
					Index:      "index",
					Item:       "id",
					Body: []ir.Node{
						&ir.Parameter{Path: "id"},
					},
				},
			},
		},
		&ir.Parameter{Path: "keyword"},
		&ir.Parameter{Path: "limit"},
		&ir.Parameter{Path: "*pointer"},
	}}

	if err := validateStatementExpressions(goPackage, method, statement); err != nil {
		t.Fatal(err)
	}
}

func TestValidateStatementExpressionsRejectsInvalidPathsAndScopes(t *testing.T) {
	testCases := []struct {
		name    string
		body    []ir.Node
		message string
	}{
		{
			name: "unknown root parameter",
			body: []ir.Node{
				&ir.Parameter{Path: "missing"},
			},
			message: `unknown identifier "missing"`,
		},
		{
			name: "unknown struct field",
			body: []ir.Node{
				&ir.Parameter{Path: "params.Missing"},
			},
			message: `type SearchParams has no exported field "Missing"`,
		},
		{
			name: "parameter field on scalar",
			body: []ir.Node{
				&ir.Parameter{Path: "params.Keyword.Length"},
			},
			message: `cannot select field "Length" from scalar type string`,
		},
		{
			name: "bind field on scalar",
			body: []ir.Node{
				&ir.Bind{Name: "length", Expression: "params.Keyword.Length"},
			},
			message: `cannot select field "Length" from scalar type string`,
		},
		{
			name: "foreach collection field on scalar",
			body: []ir.Node{
				&ir.Foreach{
					Collection: "params.Keyword.Length",
					Index:      "index",
					Item:       "value",
				},
			},
			message: `cannot select field "Length" from scalar type string`,
		},
		{
			name: "foreach item outside loop",
			body: []ir.Node{
				&ir.Foreach{
					Collection: "params.IDs",
					Index:      "index",
					Item:       "id",
					Body:       []ir.Node{&ir.Parameter{Path: "id"}},
				},
				&ir.Parameter{Path: "id"},
			},
			message: `unknown identifier "id"`,
		},
		{
			name: "conditional bind outside branch",
			body: []ir.Node{
				&ir.If{
					Test: "params.Keyword != ''",
					Body: []ir.Node{
						&ir.Bind{Name: "pattern", Expression: "params.Keyword"},
					},
				},
				&ir.Parameter{Path: "pattern"},
			},
			message: `unknown identifier "pattern"`,
		},
		{
			name: "duplicate bind",
			body: []ir.Node{
				&ir.Bind{Name: "pattern", Expression: "params.Keyword"},
				&ir.Bind{Name: "pattern", Expression: "params.Keyword"},
			},
			message: `bind name "pattern" is already declared`,
		},
		{
			name: "scalar foreach collection",
			body: []ir.Node{
				&ir.Foreach{
					Collection: "limit",
					Index:      "index",
					Item:       "value",
				},
			},
			message: "foreach collection must be a slice",
		},
		{
			name: "method call on unknown field",
			body: []ir.Node{
				&ir.Bind{Name: "value", Expression: "params.Missing.String()"},
			},
			message: `type SearchParams has no exported field "Missing"`,
		},
		{
			name: "runtime import bind",
			body: []ir.Node{
				&ir.Bind{Name: "yourbatis", Expression: "params.Keyword"},
			},
			message: `bind name "yourbatis" is reserved`,
		},
		{
			name: "parameter alias with multiple business arguments",
			body: []ir.Node{
				&ir.Bind{Name: "value", Expression: "_parameter"},
			},
			message: `unknown identifier "_parameter"`,
		},
		{
			name: "plain string text expression",
			body: []ir.Node{
				&ir.TextExpression{Expression: "params.Keyword"},
			},
			message: "plain string interpolation is disabled",
		},
		{
			name: "interface text expression",
			body: []ir.Node{
				&ir.TextExpression{Expression: "opaque"},
			},
			message: "interface text interpolation would require runtime reflection",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			goPackage, method := validationFixture(t)
			err := validateStatementExpressions(goPackage, method, &ir.Statement{Body: testCase.body})
			if err == nil || !strings.Contains(err.Error(), testCase.message) {
				t.Fatalf("error = %v, want message containing %q", err, testCase.message)
			}
		})
	}
}

func TestJavaParameterExpressionGoFirstValidation(t *testing.T) {
	testCases := []struct {
		name string
		path string
	}{
		{
			name: "propertyWithSpacesInside",
			path: "with spaces",
		},
		{
			name: "simplePropertyWithSpaceAndManyAttributes",
			path: "user name",
		},
		{
			name: "invalidJdbcTypeOptUsingExpression",
			path: "(expression)+",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			goPackage, method := validationFixture(t)
			err := validateStatementExpressions(
				goPackage,
				method,
				&ir.Statement{Body: []ir.Node{
					&ir.Parameter{Path: testCase.path},
				}},
			)
			if err == nil || !strings.Contains(err.Error(), "invalid parameter") {
				t.Fatalf("error = %v, want invalid parameter error", err)
			}
		})
	}
}

func validationFixture(t *testing.T) (*GoPackage, *MapperMethod) {
	t.Helper()
	searchParams := &StructInfo{
		Name: "SearchParams",
		Fields: []StructField{
			{Name: "Keyword", Type: mustTypeExpression(t, "string")},
			{Name: "IDs", Type: mustTypeExpression(t, "[]int64")},
			{Name: "Groups", Type: mustTypeExpression(t, "map[string][]int64")},
		},
	}
	return &GoPackage{
			Structs: map[string]*StructInfo{
				"SearchParams": searchParams,
			},
			Types: map[string]ast.Expr{
				"SearchParams": mustTypeExpression(t, "struct{}"),
			},
		}, &MapperMethod{
			Name: "Search",
			Params: []GoValue{
				{Name: "ctx", Type: "context.Context", Expr: mustTypeExpression(t, "context.Context")},
				{Name: "params", Type: "SearchParams", Expr: mustTypeExpression(t, "SearchParams")},
				{Name: "limit", Type: "int", Expr: mustTypeExpression(t, "int")},
				{Name: "pointer", Type: "*int64", Expr: mustTypeExpression(t, "*int64")},
				{Name: "opaque", Type: "any", Expr: mustTypeExpression(t, "any")},
			},
		}
}

func mustTypeExpression(t *testing.T, value string) ast.Expr {
	t.Helper()
	expression, err := goparser.ParseExpr(value)
	if err != nil {
		t.Fatal(err)
	}
	return expression
}
