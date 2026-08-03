package yourbatis

import "testing"

func TestNewIdentifierValidatesAndQuotesEachPart(t *testing.T) {
	identifier, err := NewIdentifier("main.user_table")
	if err != nil {
		t.Fatalf("NewIdentifier() error = %v", err)
	}

	testCases := []struct {
		name    string
		dialect Dialect
		want    string
	}{
		{name: "question", dialect: DialectQuestion, want: `"main"."user_table"`},
		{name: "postgres", dialect: DialectPostgres, want: `"main"."user_table"`},
		{name: "sqlite", dialect: DialectSQLite, want: `"main"."user_table"`},
		{name: "mysql", dialect: DialectMySQL, want: "`main`.`user_table`"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			if got := testCase.dialect.QuoteIdentifier(identifier); got != testCase.want {
				t.Fatalf("QuoteIdentifier() = %q, want %q", got, testCase.want)
			}
		})
	}
}

func TestNewIdentifierRejectsUnsafeInput(t *testing.T) {
	for _, value := range []string{
		"",
		"users; DROP TABLE users",
		"users.name DESC",
		"users..name",
		"1users",
	} {
		t.Run(value, func(t *testing.T) {
			if _, err := NewIdentifier(value); err == nil {
				t.Fatalf("NewIdentifier(%q) succeeded", value)
			}
		})
	}
}

func TestIdentifierConversionStillQuotesItsContents(t *testing.T) {
	testCases := []struct {
		name       string
		dialect    Dialect
		identifier Identifier
		want       string
	}{
		{
			name:       "ansi double quote",
			dialect:    DialectQuestion,
			identifier: Identifier(`users"; DROP TABLE users; --`),
			want:       `"users""; DROP TABLE users; --"`,
		},
		{
			name:       "mysql backtick",
			dialect:    DialectMySQL,
			identifier: Identifier("users`; DROP TABLE users; --"),
			want:       "`users``; DROP TABLE users; --`",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			builder := NewBuilder(testCase.dialect)
			AppendIdentifier(builder, testCase.dialect, testCase.identifier)
			if got := builder.Build().SQL; got != testCase.want {
				t.Fatalf("SQL = %q, want %q", got, testCase.want)
			}
		})
	}
}

func TestTypedTextAppendHelpers(t *testing.T) {
	builder := NewBuilder(DialectQuestion)
	AppendSigned(builder, -2)
	builder.Text("/")
	AppendUnsigned(builder, 3)
	builder.Text("/")
	AppendFloat(builder, 4.5)
	builder.Text("/")
	AppendBool(builder, true)
	builder.Text("/")
	AppendIdentifier(builder, DialectQuestion, MustIdentifier("users.id"))
	builder.Text("/")
	builder.Text(string(TrustedFragment("DESC")))

	if got := builder.Build().SQL; got != `-2/3/4.5/true/"users"."id"/DESC` {
		t.Fatalf("SQL = %q", got)
	}
}
