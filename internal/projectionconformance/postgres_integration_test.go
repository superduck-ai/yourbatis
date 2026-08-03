package projectionconformance

import (
	"context"
	"database/sql"
	"os"
	"testing"

	yourbatis "github.com/superduck-ai/yourbatis"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func TestProjectionMapperPostgresComputedAlias(t *testing.T) {
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
		CREATE TEMPORARY TABLE projection_orders (
			id       BIGINT PRIMARY KEY,
			price    BIGINT NOT NULL,
			discount BIGINT NOT NULL
		);
		INSERT INTO projection_orders (id, price, discount)
		VALUES (1, 100, 20)
	`); err != nil {
		t.Fatalf("create PostgreSQL projection fixture: %v", err)
	}

	db := yourbatis.NewDB(raw, yourbatis.DialectPostgres, yourbatis.WithDatabaseID("postgres"))
	order, err := NewProjectionMapper(db).Get(ctx, 1)
	if err != nil {
		t.Fatalf("get PostgreSQL computed projection: %v", err)
	}
	if order != (OrderProjection{Price: 100, Discount: 20, FinalPrice: 80}) {
		t.Fatalf("PostgreSQL computed projection = %#v", order)
	}
}
