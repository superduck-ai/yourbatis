package user

import (
	"strings"
	"testing"

	yourbatis "github.com/superduck-ai/yourbatis"
)

func TestGeneratedSearchBuilder(t *testing.T) {
	status := UserStatusActive
	params := SearchUserParams{
		Keyword: "%arthur%",
		Status:  &status,
		IDs:     []int64{7, 9},
		Limit:   20,
		Offset:  40,
	}

	bound := buildUserMapperSearch(yourbatis.DialectPostgres, params)
	sqlText := strings.Join(strings.Fields(bound.SQL), " ")
	expected := "SELECT id, name, email, status, created_at FROM users WHERE name LIKE $1 AND status = $2 AND id IN ( $3 , $4 ) ORDER BY created_at DESC LIMIT $5 OFFSET $6"
	if sqlText != expected {
		t.Fatalf("SQL:\n got: %s\nwant: %s", sqlText, expected)
	}

	names := make([]string, len(bound.Args))
	for index, argument := range bound.Args {
		names[index] = argument.Name
	}
	if strings.Join(names, ",") != "keyword,params.Status,userID,userID,params.Limit,params.Offset" {
		t.Fatalf("argument names = %v", names)
	}
}

func TestGeneratedSearchBuilderUsesOtherwiseAndOmitsEmptyForeach(t *testing.T) {
	params := SearchUserParams{Limit: 10}
	bound := buildUserMapperSearch(yourbatis.DialectQuestion, params)
	sqlText := strings.Join(strings.Fields(bound.SQL), " ")
	if !strings.Contains(sqlText, "WHERE status != 'deleted'") {
		t.Fatalf("SQL should contain choose/otherwise branch: %s", sqlText)
	}
	if strings.Contains(sqlText, "id IN") {
		t.Fatalf("SQL should omit empty foreach: %s", sqlText)
	}
}

func TestGeneratedSearchBuilderSkipsFilteredForeachItemsWithoutExtraSeparators(t *testing.T) {
	params := SearchUserParams{
		IDs:    []int64{7, 0, -1, 9},
		Limit:  10,
		Offset: 0,
	}
	bound := buildUserMapperSearch(yourbatis.DialectQuestion, params)
	sqlText := strings.Join(strings.Fields(bound.SQL), " ")
	if !strings.Contains(sqlText, "id IN ( ? , ? )") {
		t.Fatalf("SQL should contain only non-filtered foreach items: %s", sqlText)
	}
	if len(bound.Args) != 4 {
		t.Fatalf("arguments = %#v", bound.Args)
	}
	if bound.Args[0].Value != int64(7) || bound.Args[1].Value != int64(9) {
		t.Fatalf("foreach arguments = %#v", bound.Args[:2])
	}
}

func TestGeneratedSearchBuilderOmitsPredicateWhenAllForeachItemsAreFiltered(t *testing.T) {
	params := SearchUserParams{
		IncludeDeleted: true,
		IDs:            []int64{0, -1},
		Limit:          10,
	}
	bound := buildUserMapperSearch(yourbatis.DialectQuestion, params)
	sqlText := strings.Join(strings.Fields(bound.SQL), " ")
	if strings.Contains(sqlText, "id IN") || strings.Contains(sqlText, "WHERE") {
		t.Fatalf("SQL should omit the empty filtered predicate: %s", sqlText)
	}
	if len(bound.Args) != 2 {
		t.Fatalf("arguments = %#v", bound.Args)
	}
}

func TestGeneratedSearchEachBuilderUsesBusinessParameterAlias(t *testing.T) {
	status := UserStatusDisabled
	bound := buildUserMapperSearchEach(
		yourbatis.DialectPostgres,
		SearchUserParams{Status: &status},
	)
	sqlText := strings.Join(strings.Fields(bound.SQL), " ")
	if !strings.Contains(sqlText, "WHERE status = $1") {
		t.Fatalf("SearchEach SQL = %s", sqlText)
	}
	if len(bound.Args) != 1 ||
		bound.Args[0].Name != "_parameter.Status" ||
		bound.Args[0].Value != &status {
		t.Fatalf("SearchEach arguments = %#v", bound.Args)
	}
}

func TestGeneratedSensitiveArgument(t *testing.T) {
	bound := buildUserMapperCreate(yourbatis.DialectQuestion, CreateUserParams{
		Name:   "Arthur",
		Email:  "arthur@example.com",
		Status: UserStatusActive,
	})
	if len(bound.Args) != 3 || !bound.Args[1].Sensitive {
		t.Fatalf("arguments = %#v", bound.Args)
	}
}

func TestGeneratedIdentifierUsesDialectQuotes(t *testing.T) {
	testCases := []struct {
		name     string
		dialect  yourbatis.Dialect
		expected string
	}{
		{name: "postgres", dialect: yourbatis.DialectPostgres, expected: `SELECT name FROM users ORDER BY "name"`},
		{name: "sqlite", dialect: yourbatis.DialectSQLite, expected: `SELECT name FROM users ORDER BY "name"`},
		{name: "mysql", dialect: yourbatis.DialectMySQL, expected: "SELECT name FROM users ORDER BY `name`"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			bound := buildUserMapperListNamesOrderedBy(
				testCase.dialect,
				yourbatis.MustIdentifier("name"),
			)
			if got := strings.Join(strings.Fields(bound.SQL), " "); got != testCase.expected {
				t.Fatalf("SQL = %q, want %q", got, testCase.expected)
			}
		})
	}
}

func TestGeneratedCreateManyBuildsOneMultiValuesInsert(t *testing.T) {
	params := []CreateUserParams{
		{
			Name:   "Arthur",
			Email:  "arthur@example.com",
			Status: UserStatusActive,
		},
		{
			Name:   "Ford",
			Email:  "ford@example.com",
			Status: UserStatusDisabled,
		},
	}

	bound := buildUserMapperCreateMany(yourbatis.DialectPostgres, params)
	sqlText := strings.Join(strings.Fields(bound.SQL), " ")
	expected := "INSERT INTO users ( name, email, status ) VALUES ( $1, $2, $3 ) , ( $4, $5, $6 )"
	if sqlText != expected {
		t.Fatalf("SQL:\n got: %s\nwant: %s", sqlText, expected)
	}

	questionBound := buildUserMapperCreateMany(yourbatis.DialectQuestion, params)
	questionSQL := strings.Join(strings.Fields(questionBound.SQL), " ")
	questionExpected := "INSERT INTO users ( name, email, status ) VALUES ( ?, ?, ? ) , ( ?, ?, ? )"
	if questionSQL != questionExpected {
		t.Fatalf("question SQL:\n got: %s\nwant: %s", questionSQL, questionExpected)
	}

	if len(bound.Args) != 6 {
		t.Fatalf("arguments = %#v", bound.Args)
	}
	for index, argument := range bound.Args {
		if argument.Name != []string{
			"user.Name",
			"user.Email",
			"user.Status",
			"user.Name",
			"user.Email",
			"user.Status",
		}[index] {
			t.Fatalf("argument %d name = %q", index, argument.Name)
		}
		wantSensitive := index == 1 || index == 4
		if argument.Sensitive != wantSensitive {
			t.Fatalf(
				"argument %d sensitive = %t, want %t",
				index,
				argument.Sensitive,
				wantSensitive,
			)
		}
	}
}
