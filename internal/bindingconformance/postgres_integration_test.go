package bindingconformance

import (
	"context"
	"database/sql"
	"os"
	"strings"
	"testing"

	yourbatis "github.com/superduck-ai/yourbatis"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func TestBindingTestMigrationsPostgres(t *testing.T) {
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
	// PostgreSQL temporary tables are connection-local. Keeping one connection
	// also makes the two Mapper-instance tests exercise the same fixture state.
	raw.SetMaxOpenConns(1)
	if err := raw.PingContext(ctx); err != nil {
		t.Fatalf("ping PostgreSQL: %v", err)
	}
	postgresFixtureSQL := strings.ReplaceAll(bindingFixtureSQL, "CREATE TABLE", "CREATE TEMPORARY TABLE")
	if _, err := raw.ExecContext(ctx, postgresFixtureSQL); err != nil {
		t.Fatalf("prepare PostgreSQL binding fixture: %v", err)
	}
	runMigratedBindingTests(t, yourbatis.NewDB(raw, yourbatis.DialectPostgres))
}
