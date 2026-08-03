package mybatisconformance

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	yourbatis "github.com/superduck-ai/yourbatis"

	_ "modernc.org/sqlite"
)

func TestGeneratedDynamicSQLExecutesWithSQLiteAndTransaction(t *testing.T) {
	ctx := context.Background()
	raw, err := sql.Open(
		"sqlite",
		"file:"+t.Name()+"?mode=memory&cache=shared",
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
		CREATE TABLE common (id INTEGER PRIMARY KEY, name TEXT NOT NULL);
		INSERT INTO common (id, name) VALUES (1, 'common');
		CREATE TABLE names (id INTEGER PRIMARY KEY, name TEXT NOT NULL);
		INSERT INTO names (id, name) VALUES (1, 'Arthur');
		CREATE TABLE users (
			id INTEGER NOT NULL,
			first_attr_1 TEXT,
			first_attr_2 TEXT,
			second_attr_1 TEXT,
			second_attr_2 TEXT
		);
	`); err != nil {
		t.Fatalf("initialize SQLite: %v", err)
	}

	logger := &conformanceEventRecorder{}
	db := yourbatis.NewDB(
		raw,
		yourbatis.DialectSQLite,
		yourbatis.WithDatabaseID("sqlite"),
		yourbatis.WithLogger(logger),
	)
	mapper := NewConformanceMapper(db)
	databaseID, err := mapper.SelectDatabaseID(ctx)
	if err != nil {
		t.Fatalf("select _databaseId parameter: %v", err)
	}
	if databaseID != "sqlite" {
		t.Fatalf("selected _databaseId = %q, want sqlite", databaseID)
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

	if err := NewJavaE10Mapper(db).Run(ctx, 1); err != nil {
		t.Fatalf("execute _databaseId mapper: %v", err)
	}
	if event := logger.last(); event == nil || event.SQL != "SELECT name FROM  common  WHERE id = ?" {
		t.Fatalf("_databaseId SQL event = %#v", event)
	}

	if err := NewJavaE23Mapper(db).Run(
		ctx,
		yourbatis.MustIdentifier("main"),
		[]int64{1},
	); err != nil {
		t.Fatalf("execute typed schema mapper: %v", err)
	}

	if err := NewJavaE25Mapper(db).Run(
		ctx,
		[]string{"first-a", "first-b"},
		[]string{"second-a", "second-b"},
	); err != nil {
		t.Fatalf("execute multiple foreach mapper: %v", err)
	}
	assertSQLiteUserRows(t, ctx, raw, 1)

	forcedRollback := errors.New("force rollback")
	err = db.Transaction(ctx, func(executor yourbatis.Executor) error {
		if got := yourbatis.ExecutorDatabaseID(executor); got != "sqlite" {
			t.Fatalf("transaction database ID = %q", got)
		}
		if err := NewJavaE25Mapper(executor).Run(
			ctx,
			[]string{"rolled", "back"},
			[]string{"inside", "transaction"},
		); err != nil {
			return err
		}
		return forcedRollback
	})
	if !errors.Is(err, forcedRollback) {
		t.Fatalf("transaction error = %v", err)
	}
	assertSQLiteUserRows(t, ctx, raw, 1)
}

func assertSQLiteUserRows(
	t *testing.T,
	ctx context.Context,
	raw *sql.DB,
	expected int,
) {
	t.Helper()
	var count int
	if err := raw.QueryRowContext(ctx, "SELECT COUNT(*) FROM users").Scan(&count); err != nil {
		t.Fatalf("count SQLite users: %v", err)
	}
	if count != expected {
		t.Fatalf("SQLite user rows = %d, want %d", count, expected)
	}
}

type conformanceEventRecorder struct {
	events []yourbatis.Event
}

func (logger *conformanceEventRecorder) LogSQL(
	_ context.Context,
	event yourbatis.Event,
) {
	logger.events = append(logger.events, event)
}

func (logger *conformanceEventRecorder) last() *yourbatis.Event {
	if len(logger.events) == 0 {
		return nil
	}
	return &logger.events[len(logger.events)-1]
}
