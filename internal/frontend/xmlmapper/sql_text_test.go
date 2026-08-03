package xmlmapper

import (
	"strings"
	"testing"

	"github.com/superduck-ai/yourbatis/internal/ir"
)

func TestParseParameterUsesGoAwareDelimiters(t *testing.T) {
	testCases := []struct {
		name      string
		sql       string
		wantPath  string
		sensitive bool
	}{
		{
			name:     "comma in string index",
			sql:      `SELECT #{params.Values["a,b"]}`,
			wantPath: `params.Values["a,b"]`,
		},
		{
			name:     "slice expression",
			sql:      `SELECT #{items[1:3]}`,
			wantPath: `items[1:3]`,
		},
		{
			name:     "function arguments",
			sql:      `SELECT #{lookup(id, fallback)}`,
			wantPath: `lookup(id, fallback)`,
		},
		{
			name:      "option after nested commas",
			sql:       `SELECT #{lookup("a,b", fallback), sensitive=true}`,
			wantPath:  `lookup("a,b", fallback)`,
			sensitive: true,
		},
		{
			name:     "composite literal braces",
			sql:      `SELECT #{values[struct{ Key string }{Key: "a,b"}]}`,
			wantPath: `values[struct{ Key string }{Key: "a,b"}]`,
		},
		{
			name:     "brace in string",
			sql:      `SELECT #{values["}"]}`,
			wantPath: `values["}"]`,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			nodes, err := parseSQLText(testCase.sql, ir.Span{File: "params.xml", Line: 4, Column: 2})
			if err != nil {
				t.Fatal(err)
			}
			var parameter *ir.Parameter
			for _, node := range nodes {
				if current, ok := node.(*ir.Parameter); ok {
					parameter = current
				}
			}
			if parameter == nil {
				t.Fatalf("nodes = %#v", nodes)
			}
			if parameter.Path != testCase.wantPath || parameter.Sensitive != testCase.sensitive {
				t.Fatalf("parameter = %#v, want path %q sensitive=%v", parameter, testCase.wantPath, testCase.sensitive)
			}
		})
	}
}

func TestParseParameterRejectsMalformedPathsAndEmptyOptions(t *testing.T) {
	testCases := []struct {
		name    string
		content string
		message string
	}{
		{
			name:    "trailing empty option",
			content: "a,",
			message: "parameter option is empty",
		},
		{
			name:    "unmatched closing parenthesis",
			content: "a)",
			message: `invalid parameter "a)"`,
		},
		{
			name:    "malformed call with nested comma",
			content: "a(, b",
			message: `invalid parameter "a(, b"`,
		},
		{
			name:    "empty option between commas",
			content: "a,sensitive=true,",
			message: "parameter option is empty",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			assertJavaParameterRejected(t, testCase.content, testCase.message)
		})
	}
}

func TestParseSQLTextRejectsParameterExpressionComments(t *testing.T) {
	testCases := []struct {
		name    string
		sql     string
		message string
	}{
		{
			name:    "block comment in parameter path",
			sql:     `SELECT #{a/* comment */b}`,
			message: "comments are not allowed in #{} parameter expressions",
		},
		{
			name:    "line comment followed by expression",
			sql:     "SELECT #{a// comment\nb}",
			message: "comments are not allowed in #{} parameter expressions",
		},
		{
			name:    "line comment before closing delimiter",
			sql:     `SELECT #{a // comment }`,
			message: "comments are not allowed in #{} parameter expressions",
		},
		{
			name:    "comment in parameter options",
			sql:     `SELECT #{id, /* comment */ sensitive=true}`,
			message: "comments are not allowed in #{} parameter expressions",
		},
		{
			name:    "block comment in text expression",
			sql:     `SELECT ${a/* comment */b}`,
			message: "comments are not allowed in ${} text expressions",
		},
		{
			name:    "line comment before text expression delimiter",
			sql:     `SELECT ${a // comment }`,
			message: "comments are not allowed in ${} text expressions",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := parseSQLText(testCase.sql, ir.Span{File: "comments.xml", Line: 3, Column: 5})
			if err == nil || !strings.Contains(err.Error(), testCase.message) {
				t.Fatalf("error = %v, want message containing %q", err, testCase.message)
			}
			if strings.Contains(err.Error(), "unclosed parameter") {
				t.Fatalf("error = %v, want comment-specific diagnostic", err)
			}
		})
	}
}

func TestParseSQLTextAllowsCommentMarkersInsideStrings(t *testing.T) {
	for _, sql := range []string{
		`SELECT #{values["/* not a comment */"]}`,
		`SELECT #{values["// not a comment"]}`,
		"SELECT #{values[`// not a comment`]}",
		`SELECT ${values["/* not a comment */"]}`,
	} {
		t.Run(sql, func(t *testing.T) {
			if _, err := parseSQLText(sql, ir.Span{File: "comments.xml", Line: 3, Column: 5}); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestParseParameterRejectsComments(t *testing.T) {
	for _, content := range []string{
		`a/* comment */b`,
		"a// comment\nb",
		`a, /* comment */ sensitive=true`,
	} {
		t.Run(content, func(t *testing.T) {
			_, err := parseParameter(content, ir.Span{File: "comments.xml", Line: 3, Column: 5})
			if err == nil || !strings.Contains(err.Error(), "comments are not allowed in #{} parameter expressions") {
				t.Fatalf("error = %v, want comment-specific diagnostic", err)
			}
		})
	}
}

func TestParseSQLTextClosesParameterOnlyAtTopLevel(t *testing.T) {
	testCases := []struct {
		name           string
		sql            string
		wantParameter  string
		wantExpression string
	}{
		{
			name:          "parameter brace in string inside parentheses",
			sql:           `#{ f("a}b") }`,
			wantParameter: `f("a}b")`,
		},
		{
			name:          "parameter brace in string inside brackets",
			sql:           `#{ values["}"] }`,
			wantParameter: `values["}"]`,
		},
		{
			name:           "text expression brace inside parentheses",
			sql:            `${ f(a}b) }`,
			wantExpression: `f(a}b)`,
		},
		{
			name:           "text expression brace inside brackets",
			sql:            `${ [}] }`,
			wantExpression: `[}]`,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			nodes, err := parseSQLText(testCase.sql, ir.Span{File: "params.xml", Line: 4, Column: 2})
			if err != nil {
				t.Fatal(err)
			}
			if len(nodes) != 1 {
				t.Fatalf("nodes = %#v, want exactly one expression node", nodes)
			}
			if testCase.wantParameter != "" {
				parameter, ok := nodes[0].(*ir.Parameter)
				if !ok || parameter.Path != testCase.wantParameter {
					t.Fatalf("node = %#v, want parameter path %q", nodes[0], testCase.wantParameter)
				}
				return
			}
			expression, ok := nodes[0].(*ir.TextExpression)
			if !ok || expression.Expression != testCase.wantExpression {
				t.Fatalf("node = %#v, want text expression %q", nodes[0], testCase.wantExpression)
			}
		})
	}
}

func TestParseParameterStillRejectsMyBatisJDBCOptions(t *testing.T) {
	for _, sql := range []string{
		`SELECT #{id:BIGINT}`,
		`SELECT #{id,jdbcType=BIGINT}`,
	} {
		t.Run(sql, func(t *testing.T) {
			_, err := parseSQLText(sql, ir.Span{File: "params.xml", Line: 4, Column: 2})
			if err == nil {
				t.Fatal("expected unsupported JDBC option error")
			}
		})
	}
}

func TestParseSQLTextPreservesEscapedParameter(t *testing.T) {
	nodes, err := parseSQLText(`SELECT \#{literal}, #{id}`, ir.Span{File: "params.xml"})
	if err != nil {
		t.Fatal(err)
	}
	if text, ok := nodes[0].(*ir.Text); !ok || text.Value != "SELECT #{literal}, " {
		t.Fatalf("first node = %#v", nodes[0])
	}
}

// TestJavaParameterExpressionTest keeps the ParameterExpressionTest cases that
// can be decided by the Go-first XML placeholder parser under their original
// Java method names. Identifier and type validation cases live in the
// generator package, where Go type information is available.
func TestJavaParameterExpressionTest(t *testing.T) {
	t.Run("simpleProperty", func(t *testing.T) {
		parameter := mustParseJavaParameter(t, "id")
		if parameter.Path != "id" || len(parameter.Options) != 0 {
			t.Fatalf("parameter = %#v", parameter)
		}
	})

	t.Run("simplePropertyWithOldStyleJdbcType", func(t *testing.T) {
		assertJavaParameterRejected(t, "id:VARCHAR", `"jdbcType" is not supported`)
	})

	t.Run("oldStyleJdbcTypeWithExtraWhitespaces", func(t *testing.T) {
		assertJavaParameterRejected(t, " id :  VARCHAR ", `"jdbcType" is not supported`)
	})

	t.Run("expressionWithOldStyleJdbcType", func(t *testing.T) {
		assertJavaParameterRejected(t, "(id.String()):VARCHAR", `"jdbcType" is not supported`)
	})

	t.Run("simplePropertyWithOneAttribute", func(t *testing.T) {
		parameter := mustParseJavaParameter(t, "id,sensitive=true")
		if parameter.Path != "id" || !parameter.Sensitive || parameter.Options["sensitive"] != "true" {
			t.Fatalf("parameter = %#v", parameter)
		}
	})

	t.Run("expressionWithOneAttribute", func(t *testing.T) {
		parameter := mustParseJavaParameter(t, "(id.String()),sensitive=true")
		if parameter.Path != "(id.String())" || !parameter.Sensitive {
			t.Fatalf("parameter = %#v", parameter)
		}
	})

	t.Run("simplePropertyWithManyAttributes", func(t *testing.T) {
		assertJavaParameterRejected(
			t,
			"id, attr1=val1, attr2=val2, attr3=val3",
			`"attr1" is not supported`,
		)
	})

	t.Run("expressionWithManyAttributes", func(t *testing.T) {
		assertJavaParameterRejected(
			t,
			"(id.String()), attr1=val1, attr2=val2, attr3=val3",
			`"attr1" is not supported`,
		)
	})

	t.Run("simplePropertyWithOldStyleJdbcTypeAndAttributes", func(t *testing.T) {
		assertJavaParameterRejected(
			t,
			"id:VARCHAR, attr1=val1, attr2=val2",
			`"jdbcType" is not supported`,
		)
	})

	t.Run("simplePropertyWithSpaceAndManyAttributes", func(t *testing.T) {
		assertJavaParameterRejected(
			t,
			"user name, attr1=val1, attr2=val2, attr3=val3",
			`invalid parameter "user name"`,
		)
	})

	t.Run("invalidJdbcTypeOptUsingExpression", func(t *testing.T) {
		assertJavaParameterRejected(t, "(expression)+", `invalid parameter "(expression)+"`)
	})

	t.Run("shouldIgnoreLeadingAndTrailingSpaces", func(t *testing.T) {
		parameter := mustParseJavaParameter(t, " id , sensitive = true ")
		if parameter.Path != "id" || !parameter.Sensitive || parameter.Options["sensitive"] != "true" {
			t.Fatalf("parameter = %#v", parameter)
		}
	})

	t.Run("invalidOldJdbcTypeFormat", func(t *testing.T) {
		assertJavaParameterRejected(t, "id:", "non-empty names and values")
	})
}

func mustParseJavaParameter(t *testing.T, content string) *ir.Parameter {
	t.Helper()
	parameter, err := parseParameter(content, ir.Span{
		File:   "ParameterExpressionTest.xml",
		Line:   1,
		Column: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	return parameter
}

func assertJavaParameterRejected(t *testing.T, content, message string) {
	t.Helper()
	_, err := parseParameter(content, ir.Span{
		File:   "ParameterExpressionTest.xml",
		Line:   1,
		Column: 1,
	})
	if err == nil || !strings.Contains(err.Error(), message) {
		t.Fatalf("error = %v, want message containing %q", err, message)
	}
}
