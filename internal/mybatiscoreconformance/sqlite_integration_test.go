package mybatiscoreconformance

import (
	"context"
	"database/sql"
	"testing"

	yourbatis "github.com/superduck-ai/yourbatis"

	_ "modernc.org/sqlite"
)

func TestMigratedCoreTestsSQLite(t *testing.T) {
	ctx := context.Background()
	raw, err := sql.Open("sqlite", "file:mybatis_core_migrations?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := raw.Close(); err != nil {
			t.Errorf("close SQLite: %v", err)
		}
	})
	raw.SetMaxOpenConns(1)
	if _, err := raw.ExecContext(ctx, coreFixtureSQL); err != nil {
		t.Fatalf("prepare SQLite core fixture: %v", err)
	}
	if _, err := raw.ExecContext(
		ctx,
		"INSERT INTO migration_binary_values (id, payload) VALUES (?, ?), (?, ?)",
		1,
		[]byte{2, 3, 5},
		2,
		[]byte{8, 13},
	); err != nil {
		t.Fatalf("seed SQLite BLOB fixture: %v", err)
	}
	runMigratedCoreTests(t, yourbatis.NewDB(raw, yourbatis.DialectSQLite))
}
