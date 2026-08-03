package parsing

import (
	"strings"
	"testing"
)

type testVariableTokenHandler struct {
	variables map[string]string
}

func (handler testVariableTokenHandler) HandleToken(content string) *string {
	value, ok := handler.variables[content]
	if !ok {
		return nil
	}
	return &value
}

func TestGenericTokenParserShouldDemonstrateGenericTokenReplacement(t *testing.T) {
	parser := NewGenericTokenParser("${", "}", testVariableTokenHandler{
		variables: map[string]string{
			"first_name":     "James",
			"initial":        "T",
			"last_name":      "Kirk",
			"var{with}brace": "Hiya",
			"":               "",
		},
	})

	testCases := []struct {
		expected string
		text     string
	}{
		{expected: "James T Kirk reporting.", text: "${first_name} ${initial} ${last_name} reporting."},
		{expected: "Hello captain James T Kirk", text: "Hello captain ${first_name} ${initial} ${last_name}"},
		{expected: "James T Kirk", text: "${first_name} ${initial} ${last_name}"},
		{expected: "JamesTKirk", text: "${first_name}${initial}${last_name}"},
		{expected: "{}JamesTKirk", text: "{}${first_name}${initial}${last_name}"},
		{expected: "}JamesTKirk", text: "}${first_name}${initial}${last_name}"},
		{expected: "}James{{T}}Kirk", text: "}${first_name}{{${initial}}}${last_name}"},
		{expected: "}James}T{Kirk", text: "}${first_name}}${initial}{${last_name}"},
		{expected: "}James}T{Kirk", text: "}${first_name}}${initial}{${last_name}"},
		{expected: "}James}T{Kirk{{}}", text: "}${first_name}}${initial}{${last_name}{{}}"},
		{expected: "}James}T{Kirk{{}}", text: "}${first_name}}${initial}{${last_name}{{}}${}"},
		{expected: "{$$something}JamesTKirk", text: "{$$something}${first_name}${initial}${last_name}"},
		{expected: "${", text: "${"},
		{expected: "${\\}", text: "${\\}"},
		{expected: "Hiya", text: "${var{with\\}brace}"},
		{expected: "", text: "${}"},
		{expected: "}", text: "}"},
		{expected: "Hello ${ this is a test.", text: "Hello ${ this is a test."},
		{expected: "Hello } this is a test.", text: "Hello } this is a test."},
		{expected: "Hello } ${ this is a test.", text: "Hello } ${ this is a test."},
	}

	for _, testCase := range testCases {
		if actual := parser.Parse(testCase.text); actual != testCase.expected {
			t.Fatalf("parse(%q) = %q, want %q", testCase.text, actual, testCase.expected)
		}
	}
}

func TestGenericTokenParserShallNotInterpolateSkippedVariables(t *testing.T) {
	parser := NewGenericTokenParser("${", "}", testVariableTokenHandler{
		variables: map[string]string{},
	})

	testCases := []struct {
		expected string
		text     string
	}{
		{expected: "${skipped} variable", text: "\\${skipped} variable"},
		{expected: "This is a ${skipped} variable", text: "This is a \\${skipped} variable"},
		{expected: "null ${skipped} variable", text: "${skipped} \\${skipped} variable"},
		{expected: "The null is ${skipped} variable", text: "The ${skipped} is \\${skipped} variable"},
	}

	for _, testCase := range testCases {
		if actual := parser.Parse(testCase.text); actual != testCase.expected {
			t.Fatalf("parse(%q) = %q, want %q", testCase.text, actual, testCase.expected)
		}
	}
}

func TestGenericTokenParserShouldParseLargeInputsLikeMyBatisIssue760(t *testing.T) {
	parser := NewGenericTokenParser("${", "}", testVariableTokenHandler{
		variables: map[string]string{
			"first_name": "James",
			"initial":    "T",
			"last_name":  "Kirk",
		},
	})

	var input strings.Builder
	var expected strings.Builder
	for index := 0; index < 10000; index++ {
		input.WriteString("${first_name} ${initial} ${last_name} reporting. ")
		expected.WriteString("James T Kirk reporting. ")
	}

	if actual := parser.Parse(input.String()); actual != expected.String() {
		t.Fatalf("large parse length = %d, want %d", len(actual), len(expected.String()))
	}
}
