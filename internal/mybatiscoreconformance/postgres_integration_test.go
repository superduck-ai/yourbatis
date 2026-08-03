package mybatiscoreconformance

import (
	"context"
	"database/sql"
	"os"
	"strings"
	"testing"

	yourbatis "github.com/superduck-ai/yourbatis"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func TestMigratedCoreTestsPostgres(t *testing.T) {
	dsn := os.Getenv("YOURBATIS_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("set YOURBATIS_POSTGRES_DSN to run the PostgreSQL integration test")
	}
	ctx := context.Background()
	raw, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := raw.Close(); err != nil {
			t.Errorf("close PostgreSQL: %v", err)
		}
	})
	raw.SetMaxOpenConns(1)
	if err := raw.PingContext(ctx); err != nil {
		t.Fatalf("ping PostgreSQL: %v", err)
	}
	fixture := strings.ReplaceAll(coreFixtureSQL, "CREATE TABLE", "CREATE TEMPORARY TABLE")
	if _, err := raw.ExecContext(ctx, fixture); err != nil {
		t.Fatalf("prepare PostgreSQL core fixture: %v", err)
	}
	if _, err := raw.ExecContext(
		ctx,
		"INSERT INTO migration_binary_values (id, payload) VALUES ($1, $2), ($3, $4)",
		1,
		[]byte{2, 3, 5},
		2,
		[]byte{8, 13},
	); err != nil {
		t.Fatalf("seed PostgreSQL BLOB fixture: %v", err)
	}
	runMigratedCoreTests(t, yourbatis.NewDB(raw, yourbatis.DialectPostgres))
}
