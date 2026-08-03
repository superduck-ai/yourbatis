package mybatiscoreconformance

import (
	"bytes"
	"context"
	"errors"
	"testing"

	yourbatis "github.com/superduck-ai/yourbatis"
)

func runMigratedCoreTests(t *testing.T, executor yourbatis.Executor) {
	t.Helper()
	ctx := context.Background()

	t.Run("NoResultTypeMapTest_shouldGetAUser", func(t *testing.T) {
		user, err := NewNoResultTypeOneMapper(executor).Run(ctx, 1)
		if err != nil {
			t.Fatal(err)
		}
		if user != (InferredUser{ID: 1, Name: "User1"}) {
			t.Fatalf("user = %#v", user)
		}
	})

	t.Run("NoResultTypeMapTest_shouldGetAllUsers", func(t *testing.T) {
		users, err := NewNoResultTypeManyMapper(executor).Run(ctx)
		if err != nil {
			t.Fatal(err)
		}
		want := []InferredUser{{ID: 1, Name: "User1"}, {ID: 2, Name: "User2"}, {ID: 3, Name: "User3"}}
		if len(users) != len(want) {
			t.Fatalf("users = %#v, want %#v", users, want)
		}
		for index := range want {
			if users[index] != want[index] {
				t.Fatalf("users = %#v, want %#v", users, want)
			}
		}
	})

	t.Run("ForEachTest_shouldRemoveItemVariableInTheContext", func(t *testing.T) {
		count, err := NewForeachItemScopeMapper(executor).Run(ctx, 5, []int64{1, 2}, []int64{3, 4})
		if err != nil {
			t.Fatal(err)
		}
		if count != 5 {
			t.Fatalf("count = %d, want 5", count)
		}
	})

	t.Run("ForEachTest_shouldRemoveIndexVariableInTheContext", func(t *testing.T) {
		count, err := NewForeachIndexScopeMapper(executor).Run(ctx, 4, []int64{6, 7}, []int64{8, 9})
		if err != nil {
			t.Fatal(err)
		}
		if count != 4 {
			t.Fatalf("count = %d, want 4", count)
		}
	})

	t.Run("CamelCaseMappingTest_list", func(t *testing.T) {
		rows, err := NewCamelCaseMapper(executor).Run(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if len(rows) != 2 || rows[0].ID != 1 || rows[0].FirstName != "Fred" || rows[1].FirstName != "Wilma" {
			t.Fatalf("rows = %#v", rows)
		}
	})

	t.Run("QuotedColumnNamesTest_it", func(t *testing.T) {
		rows, err := NewQuotedAliasMapper(executor).Run(ctx)
		if err != nil {
			t.Fatal(err)
		}
		want := []QuotedAliasRow{
			{ID: 1, FirstName: "Fred", LastName: "Flintstone"},
			{ID: 2, FirstName: "Wilma", LastName: "Flintstone"},
		}
		if len(rows) != len(want) {
			t.Fatalf("rows = %#v, want %#v", rows, want)
		}
		for index := range want {
			if rows[index] != want[index] {
				t.Fatalf("rows = %#v, want %#v", rows, want)
			}
		}
	})

	t.Run("ArrayResultTypeTest_ensureByteArrayBackwardCompatibility", func(t *testing.T) {
		mapper := NewByteScalarMapper(executor)
		value, err := mapper.One(ctx, 1)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(value, []byte{2, 3, 5}) {
			t.Fatalf("BLOB = %v, want [2 3 5]", value)
		}
		if _, err := mapper.Many(ctx); !errors.Is(err, yourbatis.ErrTooManyRows) {
			t.Fatalf("multi-row []byte error = %v, want ErrTooManyRows", err)
		}
	})

	t.Run("BlobTest_insertBlobThenSelectAll", func(t *testing.T) {
		mapper := NewBlobMapper(executor)
		want := BlobRow{ID: 1, Payload: []byte{1, 2, 3, 4, 5}}
		rows, err := mapper.Insert(ctx, want)
		if err != nil {
			t.Fatal(err)
		}
		if rows != 1 {
			t.Fatalf("rows affected = %d, want 1", rows)
		}
		got, err := mapper.SelectAll(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 1 || got[0].ID != want.ID || !bytes.Equal(got[0].Payload, want.Payload) {
			t.Fatalf("BLOB rows = %#v, want %#v", got, []BlobRow{want})
		}
	})

	t.Run("ReturnInstanceForEmptyRowTest_shouldSimpleTypeBeNull", func(t *testing.T) {
		value, err := NewNullScalarMapper(executor).Run(ctx, 1)
		if err != nil {
			t.Fatal(err)
		}
		if value != nil {
			t.Fatalf("value = %q, want nil", *value)
		}
	})

	t.Run("ReturnInstanceForEmptyRowTest_shouldObjectTypeNotBeNull", func(t *testing.T) {
		row, err := NewNullStructMapper(executor).Run(ctx, 1)
		if err != nil {
			t.Fatal(err)
		}
		if row.First != nil || row.Second != nil {
			t.Fatalf("row = %#v, want non-nil row with nil fields", row)
		}
	})

	t.Run("ComplexTypeTest_shouldUpdateProps", func(t *testing.T) {
		mapper := NewNestedValuesMapper(executor)
		item := NestedItem{
			ID: 10,
			Properties: []NestedProperty{
				{ID: 11, Value: "value11"},
				{ID: 12, Value: "value12"},
			},
		}
		rows, err := mapper.Insert(ctx, item)
		if err != nil {
			t.Fatal(err)
		}
		if rows != 2 {
			t.Fatalf("rows affected = %d, want 2", rows)
		}
		count, err := mapper.Count(ctx, item.ID)
		if err != nil {
			t.Fatal(err)
		}
		if count != 2 {
			t.Fatalf("stored properties = %d, want 2", count)
		}
	})

	t.Run("EncodingTest_encoding2", func(t *testing.T) {
		value, err := NewUTF8LiteralMapper(executor).Run(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if value != "Marañón" {
			t.Fatalf("value = %q, want Marañón", value)
		}
	})

	t.Run("LiteralStringTest_shouldRetainControlCharactersInMapperXml", func(t *testing.T) {
		mapper := NewControlCharacterMapper(executor)
		rows, err := mapper.Insert(ctx, 3)
		if err != nil {
			t.Fatal(err)
		}
		if rows != 1 {
			t.Fatalf("rows affected = %d, want 1", rows)
		}
		value, err := mapper.Get(ctx, 3)
		if err != nil {
			t.Fatal(err)
		}
		if value != "\ta  b\r\n\tc\n" {
			t.Fatalf("value = %q, want exact control characters", value)
		}
	})

	t.Run("SQLFragmentCanonicalId_bareAndQualifiedReferences", func(t *testing.T) {
		mapper := NewCanonicalIncludeMapper(executor)
		for name, query := range map[string]func(context.Context, int64) (InferredUser, error){
			"bare":      mapper.Bare,
			"qualified": mapper.Qualified,
		} {
			t.Run(name, func(t *testing.T) {
				user, err := query(ctx, 1)
				if err != nil {
					t.Fatal(err)
				}
				if user != (InferredUser{ID: 1, Name: "User1"}) {
					t.Fatalf("user = %#v", user)
				}
			})
		}
	})

	t.Run("ResultMapCanonicalId_bareAndQualifiedDefinitionsAndReferences", func(t *testing.T) {
		mapper := NewCanonicalResultMapMapper(executor)
		for name, query := range map[string]func(context.Context, int64) (InferredUser, error){
			"bare definition and qualified reference": mapper.FromBareDefinition,
			"qualified definition and bare reference": mapper.FromQualifiedDefinition,
		} {
			t.Run(name, func(t *testing.T) {
				user, err := query(ctx, 1)
				if err != nil {
					t.Fatal(err)
				}
				if user != (InferredUser{ID: 1, Name: "User1"}) {
					t.Fatalf("user = %#v", user)
				}
			})
		}
	})
}

const coreFixtureSQL = `
CREATE TABLE migration_users (
    id   BIGINT PRIMARY KEY,
    name TEXT NOT NULL
);
CREATE TABLE migration_scope_values (
    id BIGINT PRIMARY KEY
);
CREATE TABLE migration_names (
    id         BIGINT PRIMARY KEY,
    first_name TEXT NOT NULL,
    last_name  TEXT NOT NULL
);
CREATE TABLE migration_binary_values (
    id      BIGINT PRIMARY KEY,
    payload BYTEA NOT NULL
);
CREATE TABLE migration_blob_rows (
    id      BIGINT PRIMARY KEY,
    payload BYTEA NOT NULL
);
CREATE TABLE migration_nullable_rows (
    id           BIGINT PRIMARY KEY,
    first_value  TEXT,
    second_value TEXT
);
CREATE TABLE migration_nested_values (
    item_id        BIGINT NOT NULL,
    property_id    BIGINT NOT NULL,
    property_value TEXT NOT NULL
);
CREATE TABLE migration_control_characters (
    id    BIGINT PRIMARY KEY,
    value TEXT NOT NULL
);

INSERT INTO migration_users (id, name) VALUES
    (1, 'User1'),
    (2, 'User2'),
    (3, 'User3');
INSERT INTO migration_scope_values (id) VALUES
    (1), (2), (3), (4), (5), (6), (7), (8), (9);
INSERT INTO migration_names (id, first_name, last_name) VALUES
    (1, 'Fred', 'Flintstone'),
    (2, 'Wilma', 'Flintstone');
INSERT INTO migration_nullable_rows (id, first_value, second_value) VALUES
    (1, NULL, NULL);
`
