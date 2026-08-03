package mybatiscoreconformance

import (
	"reflect"
	"strings"
	"testing"

	yourbatis "github.com/superduck-ai/yourbatis"
)

func TestForeachItemVariableIsRestoredAfterNestedLoop(t *testing.T) {
	bound := buildForeachItemScopeMapperRun(
		yourbatis.DialectSQLite,
		5,
		[]int64{1, 2},
		[]int64{3, 4},
	)
	wantSQL := "SELECT COUNT(*) AS count FROM migration_scope_values WHERE id IN ( ? , ? , ? , ? , ? , ? ) OR id = ?"
	if got := normalizeCoreSQL(bound.SQL); got != wantSQL {
		t.Fatalf("SQL = %q, want %q", got, wantSQL)
	}
	wantValues := []any{int64(3), int64(4), int64(1), int64(3), int64(4), int64(2), int64(5)}
	if got := bound.Values(); !reflect.DeepEqual(got, wantValues) {
		t.Fatalf("arguments = %#v, want %#v", got, wantValues)
	}
	for index, argument := range bound.Args {
		if argument.Name != "id" {
			t.Fatalf("argument %d name = %q, want id", index, argument.Name)
		}
	}
}

func TestForeachIndexVariableIsRestoredAfterNestedLoop(t *testing.T) {
	bound := buildForeachIndexScopeMapperRun(
		yourbatis.DialectPostgres,
		4,
		[]int64{6, 7},
		[]int64{8, 9},
	)
	wantSQL := "SELECT COUNT(*) AS count FROM migration_scope_values WHERE id IN ( $1 , $2 , $3 , $4 , $5 , $6 ) OR id = $7"
	if got := normalizeCoreSQL(bound.SQL); got != wantSQL {
		t.Fatalf("SQL = %q, want %q", got, wantSQL)
	}
	wantValues := []any{0, 1, 2, 0, 1, 3, int64(4)}
	if got := bound.Values(); !reflect.DeepEqual(got, wantValues) {
		t.Fatalf("arguments = %#v, want %#v", got, wantValues)
	}
	wantNames := []string{"idx", "idx", "idx + 2", "idx", "idx", "idx + 2", "idx"}
	for index, argument := range bound.Args {
		if argument.Name != wantNames[index] {
			t.Fatalf("argument %d name = %q, want %q", index, argument.Name, wantNames[index])
		}
	}
}

func TestCanonicalIncludeReferencesGenerateEquivalentSQL(t *testing.T) {
	for name, build := range map[string]func(yourbatis.Dialect, int64) yourbatis.BoundSQL{
		"bare":      buildCanonicalIncludeMapperBare,
		"qualified": buildCanonicalIncludeMapperQualified,
	} {
		t.Run(name, func(t *testing.T) {
			bound := build(yourbatis.DialectPostgres, 1)
			wantSQL := "SELECT id, name FROM migration_users WHERE id = $1"
			if got := normalizeCoreSQL(bound.SQL); got != wantSQL {
				t.Fatalf("SQL = %q, want %q", got, wantSQL)
			}
			if got := bound.Values(); !reflect.DeepEqual(got, []any{int64(1)}) {
				t.Fatalf("arguments = %#v", got)
			}
		})
	}
}

func normalizeCoreSQL(value string) string {
	return strings.Join(strings.Fields(value), " ")
}
