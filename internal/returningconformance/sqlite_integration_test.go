package returningconformance

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"testing"

	yourbatis "github.com/superduck-ai/yourbatis"

	_ "modernc.org/sqlite"
)

func TestSQLiteDMLReturning(t *testing.T) {
	ctx := context.Background()
	_, mapper, logger := newSQLiteReturningMapper(t)

	arthurID, err := mapper.CreateID(ctx, "Arthur")
	if err != nil {
		t.Fatalf("create scalar ID: %v", err)
	}
	arthur, err := mapper.Get(ctx, arthurID)
	if err != nil {
		t.Fatalf("get scalar-created user: %v", err)
	}
	if arthur.Name != "Arthur" || arthur.Status != "active" {
		t.Fatalf("scalar-created user = %#v", arthur)
	}

	bob, err := mapper.Create(ctx, "Bob", "pending")
	if err != nil {
		t.Fatalf("create mapped row: %v", err)
	}
	if bob.ID == 0 || bob.Name != "Bob" || bob.Status != "pending" || bob.CreatedAt.IsZero() {
		t.Fatalf("created Bob = %#v", bob)
	}

	pending, found, err := mapper.FindByStatus(ctx, "pending")
	if err != nil {
		t.Fatalf("find pending user: %v", err)
	}
	if !found || pending.ID != bob.ID {
		t.Fatalf("find pending user = (%#v, %t), want Bob", pending, found)
	}

	missing, found, err := mapper.FindByStatus(ctx, "missing")
	if err != nil {
		t.Fatalf("find missing status: %v", err)
	}
	if found || missing != (CreatedUser{}) {
		t.Fatalf("find missing status = (%#v, %t), want zero user and false", missing, found)
	}

	many, err := mapper.CreateMany(ctx, []string{"Carol", "Dave"})
	if err != nil {
		t.Fatalf("create many: %v", err)
	}
	if got := userNames(many); !equalStrings(got, []string{"Carol", "Dave"}) {
		t.Fatalf("created names = %#v", got)
	}
	if _, found, err := mapper.FindByStatus(ctx, "active"); !errors.Is(err, yourbatis.ErrTooManyRows) ||
		found {
		t.Fatalf("find active users = (found %t, error %v), want false and ErrTooManyRows", found, err)
	}

	renamed, err := mapper.Rename(ctx, bob.ID, "Bobby")
	if err != nil {
		t.Fatalf("rename: %v", err)
	}
	if renamed.ID != bob.ID || renamed.Name != "Bobby" {
		t.Fatalf("renamed user = %#v", renamed)
	}

	disabled, err := mapper.Disable(ctx, []int64{many[0].ID, many[1].ID})
	if err != nil {
		t.Fatalf("disable many: %v", err)
	}
	if len(disabled) != 2 {
		t.Fatalf("disabled users = %#v", disabled)
	}
	for _, user := range disabled {
		if user.Status != "disabled" {
			t.Fatalf("disabled user = %#v", user)
		}
	}

	deleted, err := mapper.Delete(ctx, bob.ID)
	if err != nil {
		t.Fatalf("delete returning: %v", err)
	}
	if deleted.ID != bob.ID || deleted.Name != "Bobby" {
		t.Fatalf("deleted user = %#v", deleted)
	}
	if _, err := mapper.Get(ctx, bob.ID); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("get deleted error = %v, want sql.ErrNoRows", err)
	}

	if _, err := mapper.Rename(ctx, 999999, "missing"); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("zero-row returning error = %v, want sql.ErrNoRows", err)
	}

	event := logger.find("ReturningMapper.Create")
	if event == nil || event.Statement.Kind != yourbatis.StatementInsert {
		t.Fatalf("create log event = %#v", event)
	}
}

func TestSQLiteDMLReturningTransactionRollback(t *testing.T) {
	ctx := context.Background()
	db, mapper, _ := newSQLiteReturningMapper(t)
	forcedRollback := errors.New("force returning rollback")

	var createdID int64
	err := db.Transaction(ctx, func(tx yourbatis.Executor) error {
		txMapper := NewReturningMapper(tx)
		created, err := txMapper.Create(ctx, "Rollback", "pending")
		if err != nil {
			return err
		}
		createdID = created.ID
		return forcedRollback
	})
	if !errors.Is(err, forcedRollback) {
		t.Fatalf("transaction error = %v, want forced rollback", err)
	}
	if _, err := mapper.Get(ctx, createdID); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("rolled-back user lookup error = %v, want sql.ErrNoRows", err)
	}
}

func TestSQLiteDMLReturningTooManyRowsCanBeRolledBack(t *testing.T) {
	ctx := context.Background()
	db, mapper, _ := newSQLiteReturningMapper(t)

	before, err := mapper.Count(ctx)
	if err != nil {
		t.Fatal(err)
	}
	err = db.Transaction(ctx, func(tx yourbatis.Executor) error {
		_, err := NewReturningMapper(tx).CreateManyAsOne(ctx, []string{"One", "Two"})
		return err
	})
	if !errors.Is(err, yourbatis.ErrTooManyRows) {
		t.Fatalf("CreateManyAsOne error = %v, want ErrTooManyRows", err)
	}
	after, err := mapper.Count(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if after != before {
		t.Fatalf("count after rollback = %d, want %d", after, before)
	}
}

type returningEventRecorder struct {
	events []yourbatis.Event
}

func (recorder *returningEventRecorder) LogSQL(_ context.Context, event yourbatis.Event) {
	recorder.events = append(recorder.events, event)
}

func (recorder *returningEventRecorder) find(statementID string) *yourbatis.Event {
	for index := range recorder.events {
		if recorder.events[index].Statement.ID == statementID {
			return &recorder.events[index]
		}
	}
	return nil
}

func newSQLiteReturningMapper(
	t *testing.T,
) (*yourbatis.DB, ReturningMapper, *returningEventRecorder) {
	t.Helper()

	ctx := context.Background()
	raw, err := sql.Open(
		"sqlite",
		"file:"+sanitizeSQLiteName(t.Name())+"?mode=memory&cache=shared&_time_format=sqlite&_texttotime=1",
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
		CREATE TABLE users (
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			name       TEXT NOT NULL,
			status     TEXT NOT NULL DEFAULT 'active',
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`); err != nil {
		t.Fatalf("create users: %v", err)
	}

	logger := &returningEventRecorder{}
	db := yourbatis.NewDB(
		raw,
		yourbatis.DialectSQLite,
		yourbatis.WithDatabaseID("sqlite"),
		yourbatis.WithLogger(logger),
	)
	return db, NewReturningMapper(db), logger
}

func sanitizeSQLiteName(value string) string {
	return fmt.Sprintf("%x", []byte(value))
}

func userNames(users []CreatedUser) []string {
	names := make([]string, 0, len(users))
	for _, user := range users {
		names = append(names, user.Name)
	}
	sort.Strings(names)
	return names
}

func equalStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}
