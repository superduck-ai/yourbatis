package generator

import (
	"strings"
	"testing"
)

func TestTranslateCommonMyBatisExpressions(t *testing.T) {
	testCases := map[string]string{
		"params.Status != null":                        "params.Status != nil",
		"params.Name != ''":                            `params.Name != ""`,
		"not params.Disabled":                          "!params.Disabled",
		"params.IDs != null and params.IDs.size > 0":   "params.IDs != nil && len(params.IDs) > 0",
		"params.IDs == null or params.IDs.isEmpty()":   "params.IDs == nil || len(params.IDs) == 0",
		`params.Name == "null and params.IDs.size"`:    `params.Name == "null and params.IDs.size"`,
		"params.Name == `null and params.IDs.size`":    "params.Name == `null and params.IDs.size`",
		`params.Name == "or" and params.IDs.size > 0`:  `params.Name == "or" && len(params.IDs) > 0`,
		"params.Initial == 'A'":                        "params.Initial == 'A'",
		"params.Name == 'a\\tb'":                       `params.Name == "a\tb"`,
		"params.Check ? params.A == 1 : params.B == 2": "func() bool {\n\tif params.Check {\n\t\treturn params.A == 1\n\t}\n\treturn params.B == 2\n}()",
	}
	for input, expected := range testCases {
		actual, err := translateExpression(input)
		if err != nil {
			t.Fatalf("translateExpression(%q): %v", input, err)
		}
		if actual != expected {
			t.Fatalf("translateExpression(%q) = %q, want %q", input, actual, expected)
		}
	}
}

func TestCompileExpressionRetainsItsValidationContext(t *testing.T) {
	compiled, err := compileExpressionForMode("index + 1", expressionText)
	if err != nil {
		t.Fatalf("compileExpressionForMode() error = %v", err)
	}
	if compiled.Mode != expressionText {
		t.Fatalf("mode = %v, want expressionText", compiled.Mode)
	}
}

func TestTranslateExpressionRejectsUnsupportedSyntax(t *testing.T) {
	if _, err := translateExpression("params.Items.{ name }"); err == nil {
		t.Fatal("expected projection syntax to be rejected")
	}
}

func TestTranslateMyBatisMembershipExpression(t *testing.T) {
	actual, err := translateExpression(`name in {'Bob', 'Steve'}`)
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{`name == "Bob"`, `name == "Steve"`, "||"} {
		if !strings.Contains(actual, expected) {
			t.Fatalf("translated expression %q does not contain %q", actual, expected)
		}
	}
}

func TestTranslateMyBatisNotInExpression(t *testing.T) {
	actual, err := translateExpression(`name not in {'', 'Bob'}`)
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{`name != ""`, `name != "Bob"`, "&&"} {
		if !strings.Contains(actual, expected) {
			t.Fatalf("translated expression %q does not contain %q", actual, expected)
		}
	}
}

func TestTranslateMyBatisTernaryExpressionLazily(t *testing.T) {
	actual, err := translateExpression(`name in {'Bob', 'Steve'} ? 'yes' : 'no'`)
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{"func() string", "if", `return "yes"`, `return "no"`} {
		if !strings.Contains(actual, expected) {
			t.Fatalf("translated expression %q does not contain %q", actual, expected)
		}
	}
}

func TestTranslateMyBatisIndexOfExpression(t *testing.T) {
	actual, err := translateExpression(`name.indexOf('v')`)
	if err != nil {
		t.Fatal(err)
	}
	if actual != `yourbatis.StringIndexRune(name, 'v')` {
		t.Fatalf("translated expression = %q", actual)
	}
}

func TestTranslateExpressionRejectsArbitraryLowercaseMethod(t *testing.T) {
	if _, err := translateExpression(`value.secret()`); err == nil {
		t.Fatal("expected arbitrary lowercase method call to be rejected")
	}
}
