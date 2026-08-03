package bindingconformance

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	yourbatis "github.com/superduck-ai/yourbatis"
	"github.com/superduck-ai/yourbatis/internal/generator"
)

func TestBindingTestMigratedGeneratedSQL(t *testing.T) {
	author := BindingAuthor{
		ID:       201,
		Username: "migration",
		Password: "secret",
		Email:    "migration@example.com",
		Bio:      "bio",
	}

	testCases := []struct {
		name       string
		bound      yourbatis.BoundSQL
		wantSQL    string
		wantNames  []string
		wantValues []any
	}{
		{
			name:       "B02 shouldFindPostsInList",
			bound:      buildBindingB02MapperRun(yourbatis.DialectSQLite, []int64{1, 3, 5}),
			wantSQL:    "SELECT id, blog_id, subject, body FROM post WHERE id IN (?, ?, ?) ORDER BY id",
			wantNames:  []string{"ids[0]", "ids[1]", "ids[2]"},
			wantValues: []any{int64(1), int64(3), int64(5)},
		},
		{
			name:       "B03 shouldFindPostsInArray",
			bound:      buildBindingB03MapperRun(yourbatis.DialectSQLite, [3]int64{1, 3, 5}),
			wantSQL:    "SELECT id, blog_id, subject, body FROM post WHERE id IN (?, ?, ?) ORDER BY id",
			wantNames:  []string{"ids[0]", "ids[1]", "ids[2]"},
			wantValues: []any{int64(1), int64(3), int64(5)},
		},
		{
			name:       "B04 shouldFindThreeSpecificPosts",
			bound:      buildBindingB04MapperRun(yourbatis.DialectSQLite, 1, 3, 5, 1, 1),
			wantSQL:    "SELECT id, blog_id, subject, body FROM post WHERE id IN (?, ?, ?) ORDER BY id LIMIT ? OFFSET ?",
			wantNames:  []string{"one", "two", "three", "limit", "offset"},
			wantValues: []any{int64(1), int64(3), int64(5), int64(1), int64(1)},
		},
		{
			name:       "B07 verifyErrorMessageFromInsertAfterSelectKey",
			bound:      buildBindingB07MapperRun(yourbatis.DialectSQLite, author),
			wantSQL:    "INSERT INTO author ( id, username, password, email, bio, missing_column ) VALUES ( ?, ?, ?, ?, ?, 'invalid' )",
			wantNames:  []string{"author.ID", "author.Username", "author.Password", "author.Email", "author.Bio"},
			wantValues: []any{author.ID, author.Username, author.Password, author.Email, author.Bio},
		},
		{
			name:    "B09 shouldSelectRandom",
			bound:   buildBindingB09MapperRun(yourbatis.DialectSQLite),
			wantSQL: "SELECT 1 AS value",
		},
		{
			name:    "B10 shouldExecuteBoundSelectListOfBlogsStatement",
			bound:   buildBindingB10MapperRun(yourbatis.DialectSQLite),
			wantSQL: "SELECT id, title FROM blog ORDER BY id",
		},
		{
			name:    "B12 resultHandlerBetweenSessions",
			bound:   buildBindingB12MapperRun(yourbatis.DialectSQLite),
			wantSQL: "SELECT id, title FROM blog ORDER BY id",
		},
		{
			name:    "B13 resultHandlerInSameSession",
			bound:   buildBindingB13MapperRun(yourbatis.DialectSQLite),
			wantSQL: "SELECT id, title FROM blog ORDER BY id",
		},
		{
			name:    "B16 shouldSelectListOfBlogsUsingXMLConfig",
			bound:   buildBindingB16MapperRun(yourbatis.DialectSQLite),
			wantSQL: "SELECT id, title FROM blog ORDER BY id",
		},
		{
			name:       "B19 shouldSelectListOfPostsLike",
			bound:      buildBindingB19MapperRun(yourbatis.DialectSQLite, "%a%", 1, 1),
			wantSQL:    "SELECT id, blog_id, subject, body FROM post WHERE subject LIKE ? ORDER BY id LIMIT ? OFFSET ?",
			wantNames:  []string{"query", "limit", "offset"},
			wantValues: []any{"%a%", int64(1), int64(1)},
		},
		{
			name:       "B20 shouldSelectListOfPostsLikeTwoParameters",
			bound:      buildBindingB20MapperRun(yourbatis.DialectSQLite, "%a%", "%a%", 1, 1),
			wantSQL:    "SELECT id, blog_id, subject, body FROM post WHERE subject LIKE ? AND body LIKE ? ORDER BY id LIMIT ? OFFSET ?",
			wantNames:  []string{"subjectQuery", "bodyQuery", "limit", "offset"},
			wantValues: []any{"%a%", "%a%", int64(1), int64(1)},
		},
		{
			name:       "B21 shouldExecuteBoundSelectOneBlogStatement",
			bound:      buildBindingB21MapperRun(yourbatis.DialectSQLite, 1),
			wantSQL:    "SELECT id, title FROM blog WHERE id = ?",
			wantNames:  []string{"id"},
			wantValues: []any{int64(1)},
		},
		{
			name:       "B28 shouldSelectOneAuthor",
			bound:      buildBindingB28MapperRun(yourbatis.DialectSQLite, 101),
			wantSQL:    "SELECT id, username, password, email, bio FROM author WHERE id = ?",
			wantNames:  []string{"id"},
			wantValues: []any{int64(101)},
		},
		{
			name:       "B37 shouldFailWhenSelectingOneBlogWithNullParam",
			bound:      buildBindingB37MapperRun(yourbatis.DialectSQLite, (*int64)(nil)),
			wantSQL:    "SELECT id, title FROM blog WHERE id = ?",
			wantNames:  []string{"id"},
			wantValues: []any{(*int64)(nil)},
		},
		{
			name:       "B38 missingNestedMapParam",
			bound:      buildBindingB38MapperRun(yourbatis.DialectSQLite, map[string]string{}),
			wantSQL:    "SELECT id, title FROM blog WHERE id = 1 AND title = ?",
			wantNames:  []string{`params["nonExistentParam"]`},
			wantValues: []any{""},
		},
		{
			name:       "B39 default30ParamNamesGoFirst",
			bound:      buildBindingB39MapperRun(yourbatis.DialectSQLite, 1, "Jim Business"),
			wantSQL:    "SELECT id, title FROM blog WHERE id = ? AND title = ?",
			wantNames:  []string{"id", "title"},
			wantValues: []any{int64(1), "Jim Business"},
		},
		{
			name:       "B40 default31ParamNamesGoFirst",
			bound:      buildBindingB40MapperRun(yourbatis.DialectSQLite, 1, "Jim Business"),
			wantSQL:    "SELECT id, title FROM blog WHERE id = ? AND title = ?",
			wantNames:  []string{"id", "title"},
			wantValues: []any{int64(1), "Jim Business"},
		},
		{
			name:       "B41 namedValueUsesIdentifier",
			bound:      buildBindingB41MapperRun(yourbatis.DialectPostgres, yourbatis.MustIdentifier("id"), 1, "Jim Business"),
			wantSQL:    `SELECT id, title FROM blog WHERE "id" = $1 AND title = $2`,
			wantNames:  []string{"id", "title"},
			wantValues: []any{int64(1), "Jim Business"},
		},
		{
			name:       "B45 resultHandlerAndRowBoundsGoFirst",
			bound:      buildBindingB45MapperRun(yourbatis.DialectSQLite, 1, 1),
			wantSQL:    "SELECT id, title FROM blog ORDER BY id LIMIT ? OFFSET ?",
			wantNames:  []string{"limit", "offset"},
			wantValues: []any{int64(1), int64(1)},
		},
		{
			name:       "B47 cursorAndRowBoundsGoFirst",
			bound:      buildBindingB47MapperRun(yourbatis.DialectSQLite, 1, 1),
			wantSQL:    "SELECT id, title FROM blog ORDER BY id LIMIT ? OFFSET ?",
			wantNames:  []string{"limit", "offset"},
			wantValues: []any{int64(1), int64(1)},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			if got := normalizeBindingSQL(testCase.bound.SQL); got != testCase.wantSQL {
				t.Fatalf("SQL = %q, want %q", got, testCase.wantSQL)
			}
			gotNames := make([]string, len(testCase.bound.Args))
			gotValues := make([]any, len(testCase.bound.Args))
			for index, argument := range testCase.bound.Args {
				gotNames[index] = argument.Name
				gotValues[index] = argument.Value
			}
			if testCase.wantNames == nil {
				testCase.wantNames = []string{}
			}
			if testCase.wantValues == nil {
				testCase.wantValues = []any{}
			}
			if !reflect.DeepEqual(gotNames, testCase.wantNames) {
				t.Fatalf("argument names = %#v, want %#v", gotNames, testCase.wantNames)
			}
			if !reflect.DeepEqual(gotValues, testCase.wantValues) {
				t.Fatalf("argument values = %#v, want %#v", gotValues, testCase.wantValues)
			}
		})
	}
}

func TestBindingB36RejectsNonExistentParameterDuringGeneration(t *testing.T) {
	output := filepath.Join(t.TempDir(), "binding_b36_mapper.sqlmap.gen.go")
	err := generator.Generate(generator.Config{
		PackageDir:    ".",
		Mapper:        "BindingB36Mapper",
		SQLFile:       "./fixtures/b36.xml",
		Output:        output,
		RuntimeImport: "github.com/superduck-ai/yourbatis",
	})
	if err == nil || !strings.Contains(err.Error(), `unknown identifier "nonExistentParam"`) {
		t.Fatalf("generation error = %v", err)
	}
	if _, statErr := os.Stat(output); !os.IsNotExist(statErr) {
		t.Fatalf("negative fixture wrote output: %v", statErr)
	}
}

func TestBindingTestMigrationInventory(t *testing.T) {
	migrated := []string{
		"02", "03", "04", "07", "09", "10", "12", "13", "16", "19", "20",
		"21", "28", "36", "37", "38", "39", "40", "41", "45", "47",
	}
	for _, id := range migrated {
		if _, err := os.Stat(filepath.Join("fixtures", "b"+id+".xml")); err != nil {
			t.Errorf("B%s XML fixture: %v", id, err)
		}
		if id == "36" {
			continue
		}
		generated := "binding_b" + id + "_mapper.sqlmap.gen.go"
		if _, err := os.Stat(generated); err != nil {
			t.Errorf("B%s generated file: %v", id, err)
		}
	}
}

func normalizeBindingSQL(value string) string {
	return strings.Join(strings.Fields(value), " ")
}
