package mybatisconformance

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	yourbatis "github.com/superduck-ai/yourbatis"
	"github.com/superduck-ai/yourbatis/internal/generator"
)

func TestJavaMigrationFixtureInventory(t *testing.T) {
	expectedFixtures := []string{
		"d01", "d02", "d03", "d04", "d05", "d06", "d07",
		"d08", "d09", "d10", "d11", "d12", "d13", "d14",
		"d15", "d16", "d17", "d18", "d19", "d20", "d21",
		"d23", "d24", "d25", "d27",
		"p01", "p02", "p06", "p07", "p11", "p12", "p14",
		"x01", "x02",
		"e01", "e02", "e03", "e04", "e05", "e06", "e07", "e08", "e09", "e10",
		"e11", "e12", "e13", "e14", "e15", "e16", "o01", "o02",
		"e17", "e18", "e19", "e20", "e21",
		"e22", "e23", "e24", "e25", "d26",
	}
	fixtures, err := filepath.Glob("javafixtures/*.xml")
	if err != nil {
		t.Fatal(err)
	}
	actualFixtures := make([]string, 0, len(fixtures))
	for _, fixture := range fixtures {
		actualFixtures = append(
			actualFixtures,
			strings.TrimSuffix(filepath.Base(fixture), filepath.Ext(fixture)),
		)
	}
	sort.Strings(expectedFixtures)
	sort.Strings(actualFixtures)
	if strings.Join(actualFixtures, ",") != strings.Join(expectedFixtures, ",") {
		t.Fatalf("fixtures = %#v, want %#v", actualFixtures, expectedFixtures)
	}

	expectedGenerated := []string{
		"java_d01.sqlmap.gen.go", "java_d02.sqlmap.gen.go", "java_d03.sqlmap.gen.go",
		"java_d04.sqlmap.gen.go", "java_d05.sqlmap.gen.go", "java_d06.sqlmap.gen.go",
		"java_d07.sqlmap.gen.go", "java_d08.sqlmap.gen.go", "java_d09.sqlmap.gen.go",
		"java_d10.sqlmap.gen.go", "java_d11.sqlmap.gen.go", "java_d12.sqlmap.gen.go",
		"java_d13.sqlmap.gen.go", "java_d14.sqlmap.gen.go", "java_d15.sqlmap.gen.go",
		"java_d16.sqlmap.gen.go", "java_d17.sqlmap.gen.go", "java_d18.sqlmap.gen.go",
		"java_d19.sqlmap.gen.go", "java_d20.sqlmap.gen.go", "java_d21.sqlmap.gen.go",
		"java_d23.sqlmap.gen.go", "java_d24.sqlmap.gen.go", "java_d25.sqlmap.gen.go",
		"java_d27.sqlmap.gen.go", "java_p01.sqlmap.gen.go", "java_p06.sqlmap.gen.go",
		"java_p07.sqlmap.gen.go", "java_p12.sqlmap.gen.go", "java_x01.sqlmap.gen.go",
		"java_e01.sqlmap.gen.go", "java_e02.sqlmap.gen.go", "java_e03.sqlmap.gen.go",
		"java_e04.sqlmap.gen.go", "java_e05.sqlmap.gen.go", "java_e06.sqlmap.gen.go",
		"java_e07.sqlmap.gen.go", "java_e08.sqlmap.gen.go", "java_e09.sqlmap.gen.go",
		"java_e10.sqlmap.gen.go",
		"java_e11.sqlmap.gen.go", "java_e12.sqlmap.gen.go", "java_e13.sqlmap.gen.go",
		"java_e14.sqlmap.gen.go", "java_e15.sqlmap.gen.go", "java_e16.sqlmap.gen.go",
		"java_o02.sqlmap.gen.go",
		"java_e17.sqlmap.gen.go", "java_e18.sqlmap.gen.go", "java_e19.sqlmap.gen.go",
		"java_e20.sqlmap.gen.go", "java_e21.sqlmap.gen.go",
		"java_e22.sqlmap.gen.go", "java_e23.sqlmap.gen.go", "java_e24.sqlmap.gen.go",
		"java_e25.sqlmap.gen.go",
	}
	generated, err := filepath.Glob("java_*.sqlmap.gen.go")
	if err != nil {
		t.Fatal(err)
	}
	actualGenerated := make([]string, 0, len(generated))
	for _, output := range generated {
		actualGenerated = append(actualGenerated, filepath.Base(output))
	}
	sort.Strings(expectedGenerated)
	sort.Strings(actualGenerated)
	if strings.Join(actualGenerated, ",") != strings.Join(expectedGenerated, ",") {
		t.Fatalf("generated files = %#v, want %#v", actualGenerated, expectedGenerated)
	}
}

func TestJavaDynamicSqlSourceShouldDemonstrateSimpleExpectedTextWithNoLoopsOrConditionals(t *testing.T) {
	assertJavaGeneratedSQL(
		t,
		buildJavaD01MapperRun(yourbatis.DialectQuestion),
		"SELECT * FROM BLOG",
	)
}

func TestJavaDynamicSqlSourceShouldDemonstrateMultipartExpectedTextWithNoLoopsOrConditionals(t *testing.T) {
	assertJavaGeneratedSQL(
		t,
		buildJavaD02MapperRun(yourbatis.DialectQuestion),
		"SELECT * FROM BLOG WHERE ID = ?",
	)
}

func TestJavaDynamicSqlSourceShouldConditionallyIncludeWhere(t *testing.T) {
	assertJavaGeneratedSQL(
		t,
		buildJavaD03MapperRun(yourbatis.DialectQuestion),
		"SELECT * FROM BLOG WHERE ID = ?",
	)
}

func TestJavaDynamicSqlSourceShouldConditionallyExcludeWhere(t *testing.T) {
	assertJavaGeneratedSQL(
		t,
		buildJavaD04MapperRun(yourbatis.DialectQuestion),
		"SELECT * FROM BLOG",
	)
}

func TestJavaDynamicSqlSourceShouldConditionallyDefault(t *testing.T) {
	assertJavaGeneratedSQL(
		t,
		buildJavaD05MapperRun(yourbatis.DialectQuestion),
		"SELECT * FROM BLOG WHERE CATEGORY = 'DEFAULT'",
	)
}

func TestJavaDynamicSqlSourceShouldConditionallyChooseFirst(t *testing.T) {
	assertJavaGeneratedSQL(
		t,
		buildJavaD06MapperRun(yourbatis.DialectQuestion),
		"SELECT * FROM BLOG WHERE CATEGORY = ?",
	)
}

func TestJavaDynamicSqlSourceShouldConditionallyChooseSecond(t *testing.T) {
	assertJavaGeneratedSQL(
		t,
		buildJavaD07MapperRun(yourbatis.DialectQuestion),
		"SELECT * FROM BLOG WHERE CATEGORY = 'NONE'",
	)
}

func TestJavaDynamicSqlSourceShouldTrimWHEREInsteadOfANDForFirstCondition(t *testing.T) {
	assertJavaGeneratedSQL(
		t,
		buildJavaD08MapperRun(yourbatis.DialectQuestion),
		"SELECT * FROM BLOG WHERE  ID = ?",
	)
}

func TestJavaDynamicSqlSourceShouldTrimWHEREANDWithLFForFirstCondition(t *testing.T) {
	assertJavaGeneratedSQL(
		t,
		buildJavaD09MapperRun(yourbatis.DialectQuestion),
		"SELECT * FROM BLOG WHERE \n ID = ?",
	)
}

func TestJavaDynamicSqlSourceShouldTrimWHEREANDWithCRLFForFirstCondition(t *testing.T) {
	assertJavaGeneratedSQL(
		t,
		buildJavaD10MapperRun(yourbatis.DialectQuestion),
		"SELECT * FROM BLOG WHERE \r\n ID = ?",
	)
}

func TestJavaDynamicSqlSourceShouldTrimWHEREANDWithTABForFirstCondition(t *testing.T) {
	assertJavaGeneratedSQL(
		t,
		buildJavaD11MapperRun(yourbatis.DialectQuestion),
		"SELECT * FROM BLOG WHERE \t ID = ?",
	)
}

func TestJavaDynamicSqlSourceShouldTrimWHEREORWithLFForFirstCondition(t *testing.T) {
	assertJavaGeneratedSQL(
		t,
		buildJavaD12MapperRun(yourbatis.DialectQuestion),
		"SELECT * FROM BLOG WHERE \n ID = ?",
	)
}

func TestJavaDynamicSqlSourceShouldTrimWHEREORWithCRLFForFirstCondition(t *testing.T) {
	assertJavaGeneratedSQL(
		t,
		buildJavaD13MapperRun(yourbatis.DialectQuestion),
		"SELECT * FROM BLOG WHERE \r\n ID = ?",
	)
}

func TestJavaDynamicSqlSourceShouldTrimWHEREORWithTABForFirstCondition(t *testing.T) {
	assertJavaGeneratedSQL(
		t,
		buildJavaD14MapperRun(yourbatis.DialectQuestion),
		"SELECT * FROM BLOG WHERE \t ID = ?",
	)
}

func TestJavaDynamicSqlSourceShouldTrimWHEREInsteadOfORForSecondCondition(t *testing.T) {
	assertJavaGeneratedSQL(
		t,
		buildJavaD15MapperRun(yourbatis.DialectQuestion),
		"SELECT * FROM BLOG WHERE  NAME = ?",
	)
}

func TestJavaDynamicSqlSourceShouldTrimWHEREInsteadOfANDForBothConditions(t *testing.T) {
	assertJavaGeneratedSQL(
		t,
		buildJavaD16MapperRun(yourbatis.DialectQuestion),
		"SELECT * FROM BLOG WHERE  ID = ?    OR NAME = ?",
	)
}

func TestJavaDynamicSqlSourceShouldTrimNoWhereClause(t *testing.T) {
	assertJavaGeneratedSQL(
		t,
		buildJavaD17MapperRun(yourbatis.DialectQuestion),
		"SELECT * FROM BLOG",
	)
}

func TestJavaDynamicSqlSourceShouldTrimSETInsteadOfCOMMAForBothConditions(t *testing.T) {
	assertJavaGeneratedSQL(
		t,
		buildJavaD18MapperRun(yourbatis.DialectQuestion),
		"UPDATE BLOG SET ID = ?,   NAME = ?",
	)
}

func TestJavaDynamicSqlSourceShouldTrimCommaAfterSET(t *testing.T) {
	assertJavaGeneratedSQL(
		t,
		buildJavaD19MapperRun(yourbatis.DialectQuestion),
		"UPDATE BLOG SET  NAME = ?",
	)
}

func TestJavaDynamicSqlSourceShouldTrimNoSetClause(t *testing.T) {
	assertJavaGeneratedSQL(
		t,
		buildJavaD20MapperRun(yourbatis.DialectQuestion),
		"UPDATE BLOG",
	)
}

func TestJavaDynamicSqlSourceShouldIterateOnceForEachItemInCollection(t *testing.T) {
	bound := buildJavaD21MapperRun(
		yourbatis.DialectQuestion,
		[]string{"one", "two", "three"},
	)
	assertJavaGeneratedSQL(
		t,
		bound,
		"SELECT * FROM BLOG WHERE ID in (  ?  AND  ?  AND  ? )",
	)
	assertJavaGeneratedArguments(
		t,
		bound,
		[]string{"item", "item", "item"},
		[]any{"one", "two", "three"},
		nil,
	)
}

func TestJavaDynamicSqlSourceShouldSkipForEachWhenCollectionIsEmpty(t *testing.T) {
	bound := buildJavaD23MapperRun(yourbatis.DialectQuestion, []int64{})
	assertJavaGeneratedSQL(t, bound, "SELECT * FROM BLOG")
	assertJavaGeneratedArguments(t, bound, nil, nil, nil)
}

func TestJavaDynamicSqlSourceShouldPerformStrictMatchOnForEachVariableSubstitution(t *testing.T) {
	bound := buildJavaD24MapperRun(
		yourbatis.DialectQuestion,
		map[string]string{"u": "root value"},
		[]ConformanceBean{{ID: "item value"}},
	)
	assertJavaGeneratedSQL(
		t,
		bound,
		"INSERT INTO BLOG (ROOT_VALUE, ITEM_VALUE) VALUES   (?, ?)",
	)
	assertJavaGeneratedArguments(
		t,
		bound,
		[]string{`uuu["u"]`, "u.ID"},
		[]any{"root value", "item value"},
		nil,
	)
}

func TestJavaDynamicSqlSourceCornerCaseForeachComesFirst(t *testing.T) {
	bound := buildJavaD25MapperRun(
		yourbatis.DialectQuestion,
		[]ConformanceBean{{ID: "bean id 1"}, {ID: "bean id 2"}},
	)
	assertJavaGeneratedSQL(t, bound, "(  ? , ? )")
	assertJavaGeneratedArguments(
		t,
		bound,
		[]string{"b.ID", "b.ID"},
		[]any{"bean id 1", "bean id 2"},
		nil,
	)
}

func TestJavaDynamicSqlSourceShrinkWhitespacesInSQLGoFirst(t *testing.T) {
	assertJavaGeneratedSQL(
		t,
		buildJavaD27MapperRun(yourbatis.DialectQuestion),
		"SELECT * \n        FROM user\n \t        WHERE user_id = 1",
	)
}

func TestJavaParameterExpressionSimpleProperty(t *testing.T) {
	bound := buildJavaP01MapperRun(yourbatis.DialectQuestion, 42)
	assertJavaGeneratedSQL(t, bound, "DELETE FROM BLOG WHERE ID = ?")
	assertJavaGeneratedArguments(t, bound, []string{"id"}, []any{int64(42)}, []bool{false})
}

func TestJavaParameterExpressionSimplePropertyWithOneAttributeGoFirst(t *testing.T) {
	bound := buildJavaP06MapperRun(yourbatis.DialectQuestion, 42)
	assertJavaGeneratedSQL(t, bound, "DELETE FROM BLOG WHERE ID = ?")
	assertJavaGeneratedArguments(t, bound, []string{"id"}, []any{int64(42)}, []bool{true})
}

func TestJavaParameterExpressionExpressionWithOneAttributeGoFirst(t *testing.T) {
	bound := buildJavaP07MapperRun(yourbatis.DialectQuestion, 41)
	assertJavaGeneratedSQL(t, bound, "DELETE FROM BLOG WHERE ID = ?")
	assertJavaGeneratedArguments(
		t,
		bound,
		[]string{"(id + 1)"},
		[]any{int64(42)},
		[]bool{true},
	)
}

func TestJavaParameterExpressionShouldIgnoreLeadingAndTrailingSpacesGoFirst(t *testing.T) {
	bound := buildJavaP12MapperRun(yourbatis.DialectQuestion, 42)
	assertJavaGeneratedSQL(t, bound, "DELETE FROM BLOG WHERE ID = ?")
	assertJavaGeneratedArguments(t, bound, []string{"id"}, []any{int64(42)}, []bool{true})
}

func TestJavaXMLScriptBuilderShouldWhereInsertWhitespace(t *testing.T) {
	assertJavaGeneratedSQL(
		t,
		buildJavaX01MapperRun(yourbatis.DialectQuestion),
		"DELETE FROM user WHERE  id = 1 and id > 0 and id = 1 and id > 0",
	)
}

func TestJavaExpressionEvaluatorShouldCompareStringsReturnTrue(t *testing.T) {
	assertJavaGeneratedSQL(
		t,
		buildJavaE01MapperRun(yourbatis.DialectQuestion, "cbegin"),
		"SELECT 1  WHERE matched = 1",
	)
}

func TestJavaExpressionEvaluatorShouldCompareStringsReturnFalse(t *testing.T) {
	assertJavaGeneratedSQL(
		t,
		buildJavaE02MapperRun(yourbatis.DialectQuestion, "cbegin"),
		"SELECT 1",
	)
}

func TestJavaExpressionEvaluatorShouldReturnTrueIfNotNull(t *testing.T) {
	username := "cbegin"
	assertJavaGeneratedSQL(
		t,
		buildJavaE03MapperRun(yourbatis.DialectQuestion, &username),
		"SELECT 1  WHERE matched = 1",
	)
}

func TestJavaExpressionEvaluatorShouldReturnFalseIfNull(t *testing.T) {
	assertJavaGeneratedSQL(
		t,
		buildJavaE04MapperRun(yourbatis.DialectQuestion, nil),
		"SELECT 1",
	)
}

func TestJavaExpressionEvaluatorShouldReturnTrueIfNotZero(t *testing.T) {
	assertJavaGeneratedSQL(
		t,
		buildJavaE05MapperRun(yourbatis.DialectQuestion, 1),
		"SELECT 1  WHERE matched = 1",
	)
}

func TestJavaExpressionEvaluatorShouldReturnFalseIfZero(t *testing.T) {
	assertJavaGeneratedSQL(
		t,
		buildJavaE06MapperRun(yourbatis.DialectQuestion, 0),
		"SELECT 1",
	)
}

func TestJavaExpressionEvaluatorShouldReturnFalseIfZeroWithScale(t *testing.T) {
	assertJavaGeneratedSQL(
		t,
		buildJavaE07MapperRun(yourbatis.DialectQuestion, 0.0),
		"SELECT 1",
	)
}

func TestJavaExpressionEvaluatorShouldIterateOverIterable(t *testing.T) {
	bound := buildJavaE08MapperRun(
		yourbatis.DialectQuestion,
		[]string{"1", "2", "3"},
	)
	assertJavaGeneratedSQL(t, bound, "SELECT 1  WHERE value IN (  ? , ? , ? )")
	assertJavaGeneratedArguments(
		t,
		bound,
		[]string{"value", "value", "value"},
		[]any{"1", "2", "3"},
		nil,
	)
}

func TestJavaSubmittedDynsqlSelectLikeUsesParameterAlias(t *testing.T) {
	bound := buildJavaE09MapperRun(yourbatis.DialectQuestion, "Ba")
	assertJavaGeneratedSQL(t, bound, "SELECT * FROM names WHERE description LIKE ?")
	assertJavaGeneratedArguments(
		t,
		bound,
		[]string{"pattern"},
		[]any{"%Ba%"},
		nil,
	)
}

func TestJavaSubmittedMultidbUsesDatabaseIDInDynamicIf(t *testing.T) {
	hsql := buildJavaE10MapperRun(yourbatis.DialectQuestion, "hsql", 1)
	assertJavaGeneratedSQL(t, hsql, "SELECT name FROM  hsql  WHERE id = ?")
	assertJavaGeneratedArguments(t, hsql, []string{"id"}, []any{int64(1)}, nil)

	common := buildJavaE10MapperRun(yourbatis.DialectQuestion, "sqlite", 1)
	assertJavaGeneratedSQL(t, common, "SELECT name FROM  common  WHERE id = ?")
	assertJavaGeneratedArguments(t, common, []string{"id"}, []any{int64(1)}, nil)
}

func TestJavaDynamicSqlSourceIndexOfAdapter(t *testing.T) {
	assertJavaGeneratedSQL(
		t,
		buildJavaE11MapperRun(yourbatis.DialectQuestion, "Steve"),
		"SELECT 1  WHERE matched = 1",
	)
	assertJavaGeneratedSQL(
		t,
		buildJavaE11MapperRun(yourbatis.DialectQuestion, "Bob"),
		"SELECT 1",
	)
}

func TestJavaDynamicSqlSourceMembershipExpression(t *testing.T) {
	assertJavaGeneratedSQL(
		t,
		buildJavaE12MapperRun(yourbatis.DialectQuestion, "Steve"),
		"SELECT 1  WHERE matched = 1",
	)
	assertJavaGeneratedSQL(
		t,
		buildJavaE12MapperRun(yourbatis.DialectQuestion, "Alice"),
		"SELECT 1",
	)
}

func TestJavaDynamicSqlSourceTernaryExpression(t *testing.T) {
	yes := buildJavaE13MapperRun(yourbatis.DialectQuestion, "Steve")
	assertJavaGeneratedSQL(t, yes, "SELECT ?")
	assertJavaGeneratedArguments(t, yes, []string{"answer"}, []any{"yes"}, nil)

	no := buildJavaE13MapperRun(yourbatis.DialectQuestion, "Alice")
	assertJavaGeneratedArguments(t, no, []string{"answer"}, []any{"no"}, nil)
}

func TestJavaSubmittedOgnlStaticNotInGoFirst(t *testing.T) {
	value := "active"
	assertJavaGeneratedSQL(
		t,
		buildJavaE14MapperRun(yourbatis.DialectQuestion, &value),
		"SELECT 1  WHERE matched = 1",
	)
	empty := ""
	assertJavaGeneratedSQL(
		t,
		buildJavaE14MapperRun(yourbatis.DialectQuestion, &empty),
		"SELECT 1",
	)
	assertJavaGeneratedSQL(
		t,
		buildJavaE14MapperRun(yourbatis.DialectQuestion, nil),
		"SELECT 1",
	)
}

func TestJavaSubmittedOgnlEnumToStringGoFirst(t *testing.T) {
	assertJavaGeneratedSQL(
		t,
		buildJavaE15MapperRun(yourbatis.DialectQuestion, ConformanceEmployee),
		"SELECT 1  WHERE employee = 1",
	)
	assertJavaGeneratedSQL(
		t,
		buildJavaE15MapperRun(yourbatis.DialectQuestion, ConformanceDirector),
		"SELECT 1",
	)
}

func TestJavaSubmittedDynsqlStaticHelperGoFirst(t *testing.T) {
	bound := buildJavaE16MapperRun(yourbatis.DialectQuestion, "Ba")
	assertJavaGeneratedSQL(t, bound, "SELECT ?")
	assertJavaGeneratedArguments(t, bound, []string{"escaped"}, []any{"%Ba%"}, nil)
}

func TestJavaOgnlCacheCompareToGoFirst(t *testing.T) {
	assertJavaGeneratedSQL(
		t,
		buildJavaO02MapperRun(yourbatis.DialectQuestion, 1, 2),
		"SELECT 1  WHERE before = 1",
	)
	assertJavaGeneratedSQL(
		t,
		buildJavaO02MapperRun(yourbatis.DialectQuestion, 2, 1),
		"SELECT 1",
	)
}

func TestJavaOgnlCachePrivateMemberAccessIsUnsupported(t *testing.T) {
	assertJavaFixtureGenerationError(
		t,
		"JavaO01Mapper",
		"./javafixtures/o01.xml",
		`type conformancePrivateData has no exported field "id"`,
	)
}

func TestJavaSubmittedBindInForeach(t *testing.T) {
	bound := buildJavaE17MapperRun(yourbatis.DialectQuestion, []int{1, 2})
	assertJavaGeneratedSQL(
		t,
		bound,
		"INSERT INTO users (id, name) VALUES  (?, ?) , (?, ?)",
	)
	assertJavaGeneratedArguments(
		t,
		bound,
		[]string{"id", "name", "id", "name"},
		[]any{1, "User1", 2, "User2"},
		nil,
	)
}

func TestJavaSubmittedForeachMapStringString(t *testing.T) {
	bound := buildJavaE18MapperRun(
		yourbatis.DialectQuestion,
		map[string]string{"key": "value"},
	)
	assertJavaGeneratedSQL(
		t,
		bound,
		"INSERT INTO string_string (key, value) VALUES  (?, ?)",
	)
	assertJavaGeneratedArguments(
		t,
		bound,
		[]string{"key", "item"},
		[]any{"key", "value"},
		nil,
	)
}

func TestJavaSubmittedForeachMapIntBool(t *testing.T) {
	bound := buildJavaE19MapperRun(
		yourbatis.DialectQuestion,
		map[int]bool{7: true},
	)
	assertJavaGeneratedArguments(
		t,
		bound,
		[]string{"key", "item"},
		[]any{7, true},
		nil,
	)
}

func TestJavaSubmittedForeachMapNestedBean(t *testing.T) {
	key := ConformancePair{A: 1, B: "key"}
	value := ConformancePair{A: 2, B: "value"}
	bound := buildJavaE20MapperRun(
		yourbatis.DialectQuestion,
		map[ConformancePair]ConformancePair{key: value},
	)
	assertJavaGeneratedArguments(
		t,
		bound,
		[]string{"key.A", "key.B", "item.A", "item.B"},
		[]any{1, "key", 2, "value"},
		nil,
	)
}

func TestJavaSubmittedNestedForeach(t *testing.T) {
	bound := buildJavaE21MapperRun(
		yourbatis.DialectQuestion,
		[]ConformanceName{{
			LastName:   "Smith",
			FirstNames: []string{"Alice", "Bob"},
		}},
	)
	assertJavaGeneratedSQL(
		t,
		bound,
		"SELECT * FROM names WHERE (last_name = ? AND first_name = ?)  OR  (last_name = ? AND first_name = ?)",
	)
	assertJavaGeneratedArguments(
		t,
		bound,
		[]string{"name.LastName", "firstName", "name.LastName", "firstName"},
		[]any{"Smith", "Alice", "Smith", "Bob"},
		nil,
	)
}

func TestJavaDynamicSqlSourceShouldHandleTextExpression(t *testing.T) {
	assertJavaGeneratedSQL(
		t,
		buildJavaE22MapperRun(yourbatis.DialectQuestion, "Steve"),
		"Expression test: 3 / yes.",
	)
	assertJavaGeneratedSQL(
		t,
		buildJavaE22MapperRun(yourbatis.DialectQuestion, "Alice"),
		"Expression test: -1 / no.",
	)
}

func TestJavaSubmittedDynsqlUsesTypedSchemaIdentifier(t *testing.T) {
	testCases := []struct {
		name     string
		dialect  yourbatis.Dialect
		expected string
	}{
		{
			name:     "question",
			dialect:  yourbatis.DialectQuestion,
			expected: `select * from "main".names where id = ?  union  select * from "main".names where id = ?`,
		},
		{
			name:     "mysql",
			dialect:  yourbatis.DialectMySQL,
			expected: "select * from `main`.names where id = ?  union  select * from `main`.names where id = ?",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			bound := buildJavaE23MapperRun(
				testCase.dialect,
				yourbatis.MustIdentifier("main"),
				[]int64{1, 2},
			)
			assertJavaGeneratedSQL(t, bound, testCase.expected)
			assertJavaGeneratedArguments(
				t,
				bound,
				[]string{"id", "id"},
				[]any{int64(1), int64(2)},
				nil,
			)
		})
	}
}

func TestJavaSubmittedForeachMapUsesTypedDynamicKey(t *testing.T) {
	bound := buildJavaE24MapperRun(
		yourbatis.DialectQuestion,
		map[yourbatis.Identifier]int{yourbatis.MustIdentifier("col_a"): 22},
	)
	assertJavaGeneratedSQL(
		t,
		bound,
		`select count(*) from key_cols where   "col_a" = ?`,
	)
	assertJavaGeneratedArguments(
		t,
		bound,
		[]string{"item"},
		[]any{22},
		nil,
	)
}

func TestJavaSubmittedMultipleIteratesUsesNumericDynamicColumns(t *testing.T) {
	bound := buildJavaE25MapperRun(
		yourbatis.DialectQuestion,
		[]string{"first-a", "first-b"},
		[]string{"second-a", "second-b"},
	)
	assertJavaGeneratedSQL(
		t,
		bound,
		"insert into users (id,  first_attr_1 , first_attr_2 ,  second_attr_1 , second_attr_2 ) values (1,  ? , ? ,  ? , ? )",
	)
	assertJavaGeneratedArguments(
		t,
		bound,
		[]string{"attr", "attr", "attr", "attr"},
		[]any{"first-a", "first-b", "second-a", "second-b"},
		nil,
	)
}

func TestJavaDynamicSqlSourceNullStringTextIsUnsupported(t *testing.T) {
	assertJavaFixtureGenerationError(
		t,
		"JavaD26Mapper",
		"./javafixtures/d26.xml",
		"not an approved SQL text type; use #{} binding",
	)
}

func TestJavaParameterExpressionPropertyWithSpacesInsideFailsGeneration(t *testing.T) {
	assertJavaFixtureGenerationError(
		t,
		"JavaP02Mapper",
		"./javafixtures/p02.xml",
		`invalid parameter "with spaces"`,
	)
}

func TestJavaParameterExpressionSimplePropertyWithSpaceAndManyAttributesFailsGeneration(t *testing.T) {
	assertJavaFixtureGenerationError(
		t,
		"JavaP11Mapper",
		"./javafixtures/p11.xml",
		`invalid parameter "user name"`,
	)
}

func TestJavaParameterExpressionInvalidJDBCTypeOptUsingExpressionFailsGeneration(t *testing.T) {
	assertJavaFixtureGenerationError(
		t,
		"JavaP14Mapper",
		"./javafixtures/p14.xml",
		`invalid parameter "(expression)+"`,
	)
}

func TestJavaXMLScriptBuilderShouldThrowIfUnknownElementFoundDuringGeneration(t *testing.T) {
	assertJavaFixtureGenerationError(
		t,
		"JavaX02Mapper",
		"./javafixtures/x02.xml",
		"choose only accepts when and otherwise",
	)
}

func assertJavaGeneratedSQL(t *testing.T, bound yourbatis.BoundSQL, expected string) {
	t.Helper()
	if bound.SQL != expected {
		t.Fatalf("SQL = %q, want %q", bound.SQL, expected)
	}
}

func assertJavaGeneratedArguments(
	t *testing.T,
	bound yourbatis.BoundSQL,
	names []string,
	values []any,
	sensitive []bool,
) {
	t.Helper()
	if len(bound.Args) != len(names) || len(bound.Args) != len(values) {
		t.Fatalf("Args = %#v, want %d arguments", bound.Args, len(names))
	}
	for index := range names {
		if bound.Args[index].Name != names[index] || bound.Args[index].Value != values[index] {
			t.Fatalf("Args[%d] = %#v, want name=%q value=%#v", index, bound.Args[index], names[index], values[index])
		}
		if sensitive != nil && bound.Args[index].Sensitive != sensitive[index] {
			t.Fatalf("Args[%d].Sensitive = %v, want %v", index, bound.Args[index].Sensitive, sensitive[index])
		}
	}
}

func assertJavaFixtureGenerationError(
	t *testing.T,
	mapper string,
	fixture string,
	message string,
) {
	t.Helper()
	packageDirectory, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	err = generator.Generate(generator.Config{
		PackageDir: packageDirectory,
		Mapper:     mapper,
		SQLFile:    fixture,
		Output:     filepath.Join(t.TempDir(), strings.ToLower(mapper)+".sqlmap.gen.go"),
	})
	if err == nil || !strings.Contains(err.Error(), message) {
		t.Fatalf("generation error = %v, want message containing %q", err, message)
	}
}
