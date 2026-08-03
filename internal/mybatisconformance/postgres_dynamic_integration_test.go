package mybatisconformance

import (
	"context"
	"database/sql"
	"os"
	"testing"

	yourbatis "github.com/superduck-ai/yourbatis"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func TestGeneratedDatabaseIDParameterExecutesWithPostgres(t *testing.T) {
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
	if err := raw.PingContext(ctx); err != nil {
		t.Fatalf("ping PostgreSQL: %v", err)
	}

	db := yourbatis.NewDB(
		raw,
		yourbatis.DialectPostgres,
		yourbatis.WithDatabaseID("postgres"),
	)
	mapper := NewConformanceMapper(db)
	databaseID, err := mapper.SelectDatabaseID(ctx)
	if err != nil {
		t.Fatalf("select _databaseId parameter: %v", err)
	}
	if databaseID != "postgres" {
		t.Fatalf("selected _databaseId = %q, want postgres", databaseID)
	}
	literal, err := mapper.SelectEscapedDollarLiteral(ctx)
	if err != nil {
		t.Fatalf("select escaped dollar literal: %v", err)
	}
	if literal != "${literal}" {
		t.Fatalf("escaped dollar literal = %q, want %q", literal, "${literal}")
	}
	ids, err := mapper.SelectCustomTrimPreservesTrailingSpace(ctx, 1)
	if err != nil {
		t.Fatalf("select using custom trim override: %v", err)
	}
	if len(ids) != 1 || ids[0] != 2 {
		t.Fatalf("custom trim IDs = %#v, want [2]", ids)
	}
	ids, err = mapper.SelectWhitespaceSuffixOverride(ctx, 1)
	if err != nil {
		t.Fatalf("select using whitespace-only suffix override: %v", err)
	}
	if len(ids) != 1 || ids[0] != 1 {
		t.Fatalf("whitespace suffix override IDs = %#v, want [1]", ids)
	}
	ids, err = mapper.SelectForeachRendering(ctx, []int64{1, 3})
	if err != nil {
		t.Fatalf("select using foreach rendering: %v", err)
	}
	if len(ids) != 2 || ids[0] != 1 || ids[1] != 3 {
		t.Fatalf("foreach IDs = %#v, want [1 3]", ids)
	}
	ids, err = mapper.SelectOtherwiseOnly(ctx, 2)
	if err != nil {
		t.Fatalf("select using otherwise-only choose: %v", err)
	}
	if len(ids) != 1 || ids[0] != 2 {
		t.Fatalf("otherwise-only IDs = %#v, want [2]", ids)
	}
	id, err := mapper.SelectEmptyChoose(ctx)
	if err != nil {
		t.Fatalf("select using empty choose: %v", err)
	}
	if id != 1 {
		t.Fatalf("empty-choose ID = %d, want 1", id)
	}
}
