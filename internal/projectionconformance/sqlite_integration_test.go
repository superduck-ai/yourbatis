package projectionconformance

import (
	"context"
	"database/sql"
	"testing"

	yourbatis "github.com/superduck-ai/yourbatis"

	_ "modernc.org/sqlite"
)

func TestProjectionMapperSQLiteExpressions(t *testing.T) {
	ctx := context.Background()
	raw, err := sql.Open("sqlite", "file:projection_conformance?mode=memory&cache=shared")
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
		CREATE TABLE projection_orders (
			id       INTEGER PRIMARY KEY,
			price    INTEGER NOT NULL,
			discount INTEGER NOT NULL
		);
		INSERT INTO projection_orders (id, price, discount)
		VALUES (1, 100, 20)
	`); err != nil {
		t.Fatalf("create SQLite projection fixture: %v", err)
	}

	db := yourbatis.NewDB(raw, yourbatis.DialectSQLite, yourbatis.WithDatabaseID("sqlite"))
	order, err := NewProjectionMapper(db).Get(ctx, 1)
	if err != nil {
		t.Fatalf("get SQLite computed projection: %v", err)
	}
	if order != (OrderProjection{Price: 100, Discount: 20, FinalPrice: 80}) {
		t.Fatalf("SQLite computed projection = %#v", order)
	}
	mapper := NewProjectionMapper(db)
	for _, testCase := range []struct {
		id   int64
		want bool
	}{
		{id: 1, want: true},
		{id: 2, want: false},
	} {
		exists, err := mapper.Exists(ctx, testCase.id)
		if err != nil {
			t.Fatalf("check SQLite unaliased EXISTS for id %d: %v", testCase.id, err)
		}
		if exists != testCase.want {
			t.Fatalf("SQLite unaliased EXISTS for id %d = %t, want %t", testCase.id, exists, testCase.want)
		}
	}
}
