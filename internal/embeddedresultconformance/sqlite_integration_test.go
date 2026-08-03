package embeddedresultconformance

import (
	"context"
	"database/sql"
	"testing"

	yourbatis "github.com/superduck-ai/yourbatis"

	_ "modernc.org/sqlite"
)

func TestEmbeddedResultMapperSQLitePromotedField(t *testing.T) {
	ctx := context.Background()
	raw, err := sql.Open(
		"sqlite",
		"file:embedded_result_conformance?mode=memory&cache=shared",
	)
	if err != nil {
		t.Fatalf("open SQLite: %v", err)
	}
	t.Cleanup(func() {
		if err := raw.Close(); err != nil {
			t.Errorf("close SQLite: %v", err)
		}
	})
	raw.SetMaxOpenConns(1)

	if _, err := raw.ExecContext(ctx, `
		CREATE TABLE embedded_result_users (
			id   INTEGER PRIMARY KEY,
			name TEXT NOT NULL
		);
		INSERT INTO embedded_result_users (id, name)
		VALUES (1, 'SQLite Embedded')
	`); err != nil {
		t.Fatalf("create SQLite embedded result fixture: %v", err)
	}

	db := yourbatis.NewDB(raw, yourbatis.DialectSQLite, yourbatis.WithDatabaseID("sqlite"))
	user, err := NewEmbeddedResultMapper(db).Get(ctx, 1)
	if err != nil {
		t.Fatalf("get SQLite embedded result: %v", err)
	}
	if user.ID != 1 || user.Name != "SQLite Embedded" {
		t.Fatalf("SQLite embedded result = %#v", user)
	}
}
