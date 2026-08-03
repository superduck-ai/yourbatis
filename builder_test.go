package yourbatis

import (
	"strings"
	"testing"
)

func TestBuilderRendersDialectPlaceholdersAndArguments(t *testing.T) {
	builder := NewBuilder(DialectPostgres)
	builder.Text("SELECT * FROM users WHERE id = ")
	builder.Arg("id", int64(42))
	builder.Text(" AND status = ")
	builder.Arg("status", "active")

	bound := builder.Build()
	if bound.SQL != "SELECT * FROM users WHERE id = $1 AND status = $2" {
		t.Fatalf("SQL = %q", bound.SQL)
	}
	if len(bound.Args) != 2 || bound.Args[0].Name != "id" || bound.Args[1].Name != "status" {
		t.Fatalf("Args = %#v", bound.Args)
	}
}

func TestBuilderTrimRemovesLeadingANDAndTrailingComma(t *testing.T) {
	builder := NewBuilder(DialectQuestion)

	where := NewBuilder(DialectQuestion)
	where.Text("\n AND status = ")
	where.Arg("status", "active")
	builder.AppendTrimmed(where, " WHERE ", myBatisWherePrefixOverrides(), "", nil)

	set := NewBuilder(DialectQuestion)
	set.Text(" name = ")
	set.Arg("name", "Arthur")
	set.Text(", ")
	builder.AppendTrimmed(set, " SET ", nil, "", []string{","})

	bound := builder.Build()
	if normalizeSQL(bound.SQL) != "WHERE status = ? SET name = ?" {
		t.Fatalf("SQL = %q", bound.SQL)
	}
}

func TestBuilderTrimDoesNotRemoveOperatorPrefixFromIdentifier(t *testing.T) {
	builder := NewBuilder(DialectQuestion)
	where := NewBuilder(DialectQuestion)
	where.Text(" ORDER_TYPE = ")
	where.Arg("orderType", "slow")

	builder.AppendTrimmed(where, " WHERE ", []string{"AND ", "OR "}, "", nil)

	bound := builder.Build()
	if normalizeSQL(bound.SQL) != "WHERE ORDER_TYPE = ?" {
		t.Fatalf("SQL = %q", bound.SQL)
	}
	if len(bound.Args) != 1 || bound.Args[0].Name != "orderType" {
		t.Fatalf("Args = %#v", bound.Args)
	}
}

func TestBuilderTrimMatchesCompleteMyBatisOverrideTokens(t *testing.T) {
	testCases := map[string]string{
		"AND status = 1":     "WHERE  status = 1",
		"or\nstatus = 1":     "WHERE \nstatus = 1",
		"AND,status = 1":     "WHERE AND,status = 1",
		"OR(status = 1)":     "WHERE OR(status = 1)",
		"AND/*comment*/true": "WHERE AND/*comment*/true",
		"ORACLE = 1":         "WHERE ORACLE = 1",
	}
	for input, expected := range testCases {
		t.Run(input, func(t *testing.T) {
			builder := NewBuilder(DialectQuestion)
			where := NewBuilder(DialectQuestion)
			where.Text(input)
			builder.AppendTrimmed(where, "WHERE ", myBatisWherePrefixOverrides(), "", nil)

			if actual := builder.Build().SQL; actual != expected {
				t.Fatalf("SQL = %q, want %q", actual, expected)
			}
		})
	}
}

func TestBuilderTrimHonorsBareCustomPrefixOverride(t *testing.T) {
	builder := NewBuilder(DialectQuestion)
	fragment := NewBuilder(DialectQuestion)
	fragment.Text("OR(status = 1)")

	builder.AppendTrimmed(fragment, "WHERE ", []string{"OR"}, "", nil)

	if actual := builder.Build().SQL; actual != "WHERE (status = 1)" {
		t.Fatalf("SQL = %q", actual)
	}
}

func TestBuilderWhereTrimMatchesMyBatisWhitespaceCases(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{name: "AND space", input: "   and ID = ?  ", expected: "SELECT * FROM BLOG WHERE  ID = ?"},
		{name: "AND LF", input: "   and\n ID = ?  ", expected: "SELECT * FROM BLOG WHERE \n ID = ?"},
		{name: "AND CRLF", input: "   and\r\n ID = ?  ", expected: "SELECT * FROM BLOG WHERE \r\n ID = ?"},
		{name: "AND tab", input: "   and\t ID = ?  ", expected: "SELECT * FROM BLOG WHERE \t ID = ?"},
		{name: "OR LF", input: "   or\n ID = ?  ", expected: "SELECT * FROM BLOG WHERE \n ID = ?"},
		{name: "OR CRLF", input: "   or\r\n ID = ?  ", expected: "SELECT * FROM BLOG WHERE \r\n ID = ?"},
		{name: "OR tab", input: "   or\t ID = ?  ", expected: "SELECT * FROM BLOG WHERE \t ID = ?"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			builder := NewBuilder(DialectQuestion)
			builder.Text("SELECT * FROM BLOG")
			where := NewBuilder(DialectQuestion)
			where.Text(testCase.input)
			builder.AppendTrimmed(where, " WHERE ", myBatisWherePrefixOverrides(), "", nil)
			if actual := builder.Build().SQL; actual != testCase.expected {
				t.Fatalf("SQL = %q, want %q", actual, testCase.expected)
			}
		})
	}
}

func TestBuilderSetTrimRemovesLeadingAndTrailingCommas(t *testing.T) {
	builder := NewBuilder(DialectQuestion)
	builder.Text("UPDATE BLOG")
	set := NewBuilder(DialectQuestion)
	set.Text(", NAME = ")
	set.Arg("name", "Arthur")
	set.Text(",")
	builder.AppendTrimmed(set, " SET ", []string{","}, "", []string{","})

	if actual := builder.Build().SQL; actual != "UPDATE BLOG SET  NAME = ?" {
		t.Fatalf("SQL = %q", actual)
	}
}

func TestBuilderSuffixOverrideMatchesMyBatisTrimmedFallback(t *testing.T) {
	builder := NewBuilder(DialectQuestion)
	fragment := NewBuilder(DialectQuestion)
	fragment.Text("value,")

	builder.AppendTrimmed(fragment, "(", nil, ")", []string{", "})

	if actual := builder.Build().SQL; actual != "(value)" {
		t.Fatalf("SQL = %q", actual)
	}
}

func TestBuilderSuffixOverrideSkipsWhitespaceOnlyRule(t *testing.T) {
	builder := NewBuilder(DialectQuestion)
	fragment := NewBuilder(DialectQuestion)
	fragment.Text("value,")

	builder.AppendTrimmed(fragment, "(", nil, ")", []string{" ", ","})

	if actual := builder.Build().SQL; actual != "(value)" {
		t.Fatalf("SQL = %q", actual)
	}
}

func TestBuilderTrimSkipsEmptyContent(t *testing.T) {
	builder := NewBuilder(DialectQuestion)
	builder.Text("SELECT * FROM BLOG")
	where := NewBuilder(DialectQuestion)
	where.Text(" \n\t ")
	builder.AppendTrimmed(where, " WHERE ", myBatisWherePrefixOverrides(), "", nil)

	if actual := builder.Build().SQL; actual != "SELECT * FROM BLOG" {
		t.Fatalf("SQL = %q", actual)
	}
}

func TestBuilderAppendSeparatedSkipsFilteredForeachItems(t *testing.T) {
	collection := NewBuilder(DialectQuestion)
	for _, id := range []int64{1, 0, 2} {
		item := NewBuilder(DialectQuestion)
		if id > 0 {
			item.Arg("id", id)
		}
		collection.AppendSeparated(item, ",")
	}

	builder := NewBuilder(DialectQuestion)
	if collection.HasContent() {
		builder.Text("id IN (")
		builder.Append(collection)
		builder.Text(")")
	}
	bound := builder.Build()
	if bound.SQL != "id IN (?,?)" {
		t.Fatalf("SQL = %q", bound.SQL)
	}
	if len(bound.Args) != 2 || bound.Args[0].Value != int64(1) || bound.Args[1].Value != int64(2) {
		t.Fatalf("Args = %#v", bound.Args)
	}
}

func TestBuilderForeachKeepsOpenCloseWhenAllItemsAreFiltered(t *testing.T) {
	collection := NewBuilder(DialectQuestion)
	for _, id := range []int64{0, -1} {
		item := NewBuilder(DialectQuestion)
		if id > 0 {
			item.Arg("id", id)
		}
		collection.AppendSeparated(item, ",")
	}

	builder := NewBuilder(DialectQuestion)
	if collection.HasIterations() {
		builder.Text("id IN (")
		builder.Append(collection)
		builder.Text(")")
	}
	if actual := builder.Build().SQL; actual != "id IN ()" {
		t.Fatalf("SQL = %q", actual)
	}
}

func TestBuilderAppendSeparatedSkipsEmptyCollection(t *testing.T) {
	collection := NewBuilder(DialectQuestion)
	if collection.HasIterations() {
		t.Fatal("empty collection unexpectedly reports iterations")
	}
	collection.AppendSeparated(NewBuilder(DialectQuestion), ",")
	if collection.HasContent() {
		t.Fatalf("collection unexpectedly has content: %#v", collection.Build())
	}
	if !collection.HasIterations() {
		t.Fatal("filtered collection item should still count as an iteration")
	}
}

func TestBuilderMyBatisForeachSQLNodeBoundaries(t *testing.T) {
	builder := NewBuilder(DialectQuestion)
	builder.BeginSQLNode()
	builder.Text("(")

	collection := NewBuilder(DialectQuestion)
	for _, value := range []int{1, 2, 3} {
		item := NewBuilder(DialectQuestion)
		item.BeginSQLNode()
		item.Arg("item", value)
		collection.AppendSeparated(item, ",")
	}
	builder.Append(collection)
	builder.BeginSQLNode()
	builder.Text(")")

	bound := builder.Build()
	if bound.SQL != "(  ? , ? , ? )" {
		t.Fatalf("SQL = %q", bound.SQL)
	}
	if len(bound.Args) != 3 {
		t.Fatalf("Args = %#v", bound.Args)
	}
}

func TestBuilderMyBatisForeachPrefixesFirstNonBlankSQLNode(t *testing.T) {
	builder := NewBuilder(DialectQuestion)
	builder.BeginSQLNode()
	builder.Text("(")

	collection := NewBuilder(DialectQuestion)
	item := NewBuilder(DialectQuestion)
	item.BeginSQLNode()
	item.Text("\n")
	item.BeginSQLNode()
	item.Arg("item", 1)
	collection.AppendSeparated(item, ",")
	builder.Append(collection)
	builder.BeginSQLNode()
	builder.Text(")")

	if actual := builder.Build().SQL; actual != "( \n  ? )" {
		t.Fatalf("SQL = %q", actual)
	}
}

func normalizeSQL(value string) string {
	return strings.Join(strings.Fields(value), " ")
}

func myBatisWherePrefixOverrides() []string {
	return []string{
		"AND ", "OR ",
		"AND\n", "OR\n",
		"AND\r", "OR\r",
		"AND\t", "OR\t",
	}
}
