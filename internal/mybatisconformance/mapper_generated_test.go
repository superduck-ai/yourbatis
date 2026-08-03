package mybatisconformance

import (
	"strings"
	"testing"

	yourbatis "github.com/superduck-ai/yourbatis"
)

func TestGeneratedEscapedDollarPlaceholderStaysLiteral(t *testing.T) {
	testCases := []struct {
		name    string
		dialect yourbatis.Dialect
	}{
		{
			name:    "PostgreSQL",
			dialect: yourbatis.DialectPostgres,
		},
		{
			name:    "SQLite",
			dialect: yourbatis.DialectSQLite,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			bound := buildConformanceMapperSelectEscapedDollarLiteral(testCase.dialect)
			if sql := normalize(bound.SQL); sql != `SELECT '${literal}' AS literal_value` {
				t.Fatalf("SQL = %q", sql)
			}
			if len(bound.Args) != 0 {
				t.Fatalf("Args = %#v", bound.Args)
			}
		})
	}
}

func TestGeneratedCustomTrimPreservesTrailingOverrideSpace(t *testing.T) {
	for _, dialect := range []yourbatis.Dialect{yourbatis.DialectPostgres, yourbatis.DialectSQLite} {
		bound := buildConformanceMapperSelectCustomTrimPreservesTrailingSpace(dialect, 1)
		if sql := normalize(bound.SQL); sql !=
			"SELECT id FROM (SELECT 1 AS id UNION ALL SELECT 2 AS id) AS trim_fixture WHERE NOT(id = "+dialect.Placeholder(1)+") ORDER BY id" {
			t.Fatalf("%s SQL = %q", dialect, sql)
		}
	}
}

func TestGeneratedTrimSkipsWhitespaceOnlySuffixOverride(t *testing.T) {
	for _, dialect := range []yourbatis.Dialect{yourbatis.DialectPostgres, yourbatis.DialectSQLite} {
		bound := buildConformanceMapperSelectWhitespaceSuffixOverride(dialect, 1)
		expected := "SELECT id FROM (SELECT 1 AS id UNION ALL SELECT 2 AS id) AS suffix_fixture WHERE id IN ( " +
			dialect.Placeholder(1) + " ) ORDER BY id"
		if sql := normalize(bound.SQL); sql != expected {
			t.Fatalf("%s SQL = %q, want %q", dialect, sql, expected)
		}
	}
}

func TestGeneratedForeachRenderingMatchesMyBatis(t *testing.T) {
	for _, dialect := range []yourbatis.Dialect{yourbatis.DialectPostgres, yourbatis.DialectSQLite} {
		bound := buildConformanceMapperSelectForeachRendering(dialect, []int64{1, 3})
		expected := "SELECT id FROM (SELECT 1 AS id UNION ALL SELECT 2 AS id UNION ALL SELECT 3 AS id) AS foreach_fixture WHERE id IN  (  " +
			dialect.Placeholder(1) + " , " + dialect.Placeholder(2) + " )  ORDER BY id"
		if bound.SQL != expected {
			t.Fatalf("%s SQL = %q, want %q", dialect, bound.SQL, expected)
		}
	}
}

func TestGeneratedMapForeachUsesKeyAndValue(t *testing.T) {
	bound := buildConformanceMapperInsertMap(yourbatis.DialectQuestion, map[string]int64{
		"first":  1,
		"second": 2,
	})
	if normalize(bound.SQL) != "INSERT INTO pairs (key, value) VALUES (?, ?) , (?, ?)" {
		t.Fatalf("SQL = %q", normalize(bound.SQL))
	}
	if len(bound.Args) != 4 {
		t.Fatalf("Args = %#v", bound.Args)
	}
	seen := map[string]int64{}
	for index := 0; index < len(bound.Args); index += 2 {
		key, ok := bound.Args[index].Value.(string)
		if !ok {
			t.Fatalf("map key argument = %#v", bound.Args[index])
		}
		value, ok := bound.Args[index+1].Value.(int64)
		if !ok {
			t.Fatalf("map value argument = %#v", bound.Args[index+1])
		}
		seen[key] = value
	}
	if seen["first"] != 1 || seen["second"] != 2 {
		t.Fatalf("map arguments = %#v", seen)
	}
}

func TestGeneratedForeachCanBeTheFirstSQLNode(t *testing.T) {
	bound := buildConformanceMapperInsertForeachFirst(yourbatis.DialectQuestion, []int64{3, 8})
	if normalize(bound.SQL) != "INSERT INTO ids (id) VALUES (?) , (?)" {
		t.Fatalf("SQL = %q", normalize(bound.SQL))
	}
	if len(bound.Args) != 2 || bound.Args[0].Value != int64(3) || bound.Args[1].Value != int64(8) {
		t.Fatalf("Args = %#v", bound.Args)
	}
}

func TestGeneratedBindInsideForeach(t *testing.T) {
	bound := buildConformanceMapperInsertBound(yourbatis.DialectQuestion, []int64{2, 4, 6})
	if len(bound.Args) != 6 {
		t.Fatalf("Args = %#v", bound.Args)
	}
	for index, expected := range []int64{3, 5, 7} {
		if bound.Args[index*2].Value != index || bound.Args[index*2+1].Value != expected {
			t.Fatalf("Args = %#v", bound.Args)
		}
	}
}

func TestGeneratedForeachSkipsNilItemsWithoutExtraSeparator(t *testing.T) {
	first, second := int64(3), int64(8)
	bound := buildConformanceMapperInsertFiltered(
		yourbatis.DialectQuestion,
		[]*int64{&first, nil, &second},
	)
	if normalize(bound.SQL) != "INSERT INTO ids (id) VALUES (?) , (?)" {
		t.Fatalf("SQL = %q", normalize(bound.SQL))
	}
	if len(bound.Args) != 2 || bound.Args[0].Value != first || bound.Args[1].Value != second {
		t.Fatalf("Args = %#v", bound.Args)
	}
}

func TestGeneratedNestedForeach(t *testing.T) {
	bound := buildConformanceMapperInsertNested(
		yourbatis.DialectQuestion,
		[][]int64{{10, 11}, {}, {20}},
	)
	if normalize(bound.SQL) != "INSERT INTO grouped_ids (group_position, id) VALUES (?, ?) , (?, ?) , (?, ?)" {
		t.Fatalf("SQL = %q", normalize(bound.SQL))
	}
	if len(bound.Args) != 6 {
		t.Fatalf("Args = %#v", bound.Args)
	}
	expected := []any{0, int64(10), 0, int64(11), 2, int64(20)}
	for index := range expected {
		if bound.Args[index].Value != expected[index] {
			t.Fatalf("Args = %#v", bound.Args)
		}
	}
}

func TestGeneratedForeachUsesLexicalNamesWithoutPrefixSubstitution(t *testing.T) {
	bound := buildConformanceMapperInsertStrictNames(
		yourbatis.DialectQuestion,
		map[string]string{"u": "root value"},
		[]ConformanceBean{{ID: "item value"}},
	)
	if normalize(bound.SQL) !=
		"INSERT INTO strict_names (root_value, item_value) VALUES (?, ?)" {
		t.Fatalf("SQL = %q", normalize(bound.SQL))
	}
	if len(bound.Args) != 2 {
		t.Fatalf("Args = %#v", bound.Args)
	}
	if bound.Args[0].Name != `uuu["u"]` ||
		bound.Args[0].Value != "root value" ||
		bound.Args[1].Name != "u.ID" ||
		bound.Args[1].Value != "item value" {
		t.Fatalf("Args = %#v", bound.Args)
	}
}

func TestGeneratedChooseUsesFirstMatchingBranchAndDefault(t *testing.T) {
	testCases := map[int]string{
		1:  "('first', ?)",
		2:  "('second', ?)",
		99: "('default', ?)",
	}
	for mode, expected := range testCases {
		bound := buildConformanceMapperInsertChosen(yourbatis.DialectQuestion, mode, 42)
		if !strings.Contains(normalize(bound.SQL), expected) {
			t.Fatalf("mode %d SQL = %q", mode, normalize(bound.SQL))
		}
		if len(bound.Args) != 1 || bound.Args[0].Value != int64(42) {
			t.Fatalf("mode %d Args = %#v", mode, bound.Args)
		}
	}
}

func TestGeneratedAdjacentDynamicNodesStaySeparated(t *testing.T) {
	bound := buildConformanceMapperDeleteAdjacent(yourbatis.DialectQuestion, true, true)
	if normalize(bound.SQL) != "DELETE FROM flags WHERE 1 = 1 AND left_enabled = 1 AND right_enabled = 1" {
		t.Fatalf("SQL = %q", normalize(bound.SQL))
	}
}

func TestJavaXMLScriptBuilderTestShouldWhereInsertWhitespace(t *testing.T) {
	bound := buildConformanceMapperDeleteWhereSpacing(
		yourbatis.DialectQuestion,
		true,
		true,
		true,
		true,
	)
	if normalize(bound.SQL) !=
		"DELETE FROM users WHERE id = 1 AND id > 0 AND id = 1 AND id > 0" {
		t.Fatalf("SQL = %q", normalize(bound.SQL))
	}
}

func TestGeneratedCustomTrimAddsAffixSpacingAndRemovesSuffix(t *testing.T) {
	bound := buildConformanceMapperInsertTrimmed(yourbatis.DialectQuestion, 42)
	if normalize(bound.SQL) != "INSERT INTO ids ( id ) VALUES ( ? )" {
		t.Fatalf("SQL = %q", normalize(bound.SQL))
	}
	if len(bound.Args) != 1 || bound.Args[0].Value != int64(42) {
		t.Fatalf("Args = %#v", bound.Args)
	}
}

func TestGeneratedSetTrimsLeadingAndTrailingCommas(t *testing.T) {
	name, status := "Arthur", "active"
	bound := buildConformanceMapperUpdateOptional(yourbatis.DialectQuestion, &name, &status)
	if normalize(bound.SQL) != "UPDATE users SET name = ?, status = ?" {
		t.Fatalf("SQL = %q", normalize(bound.SQL))
	}
	if len(bound.Args) != 2 {
		t.Fatalf("Args = %#v", bound.Args)
	}

	empty := buildConformanceMapperUpdateOptional(yourbatis.DialectQuestion, nil, nil)
	if normalize(empty.SQL) != "UPDATE users" {
		t.Fatalf("empty SQL = %q", normalize(empty.SQL))
	}
}

func normalize(value string) string {
	return strings.Join(strings.Fields(value), " ")
}
