package embeddedresultconformance

import (
	"context"
	"database/sql"
	"os"
	"testing"

	yourbatis "github.com/superduck-ai/yourbatis"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func TestEmbeddedResultMapperPostgresPromotedField(t *testing.T) {
	dsn := os.Getenv("YOURBATIS_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("set YOURBATIS_POSTGRES_DSN to run the PostgreSQL integration test")
	}

	ctx := context.Background()
	raw, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("open PostgreSQL: %v", err)
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
	if _, err := raw.ExecContext(ctx, `
		CREATE TEMPORARY TABLE embedded_result_users (
			id   BIGINT PRIMARY KEY,
			name TEXT NOT NULL
		);
		INSERT INTO embedded_result_users (id, name)
		VALUES (1, 'Postgres Embedded')
	`); err != nil {
		t.Fatalf("create PostgreSQL embedded result fixture: %v", err)
	}

	db := yourbatis.NewDB(raw, yourbatis.DialectPostgres, yourbatis.WithDatabaseID("postgres"))
	user, err := NewEmbeddedResultMapper(db).Get(ctx, 1)
	if err != nil {
		t.Fatalf("get PostgreSQL embedded result: %v", err)
	}
	if user.ID != 1 || user.Name != "Postgres Embedded" {
		t.Fatalf("PostgreSQL embedded result = %#v", user)
	}
}
