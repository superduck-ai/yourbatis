package user

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"testing"

	yourbatis "github.com/superduck-ai/yourbatis"

	_ "modernc.org/sqlite"
)

func TestUserMapperSQLiteCRUD(t *testing.T) {
	ctx := context.Background()
	debugOutput := captureEnvironmentDebugOutput(t)
	_, mapper, logger := newSQLiteMapper(t)

	arthurID, err := mapper.Create(ctx, CreateUserParams{
		Name:   "Arthur",
		Email:  "arthur@example.com",
		Status: UserStatusActive,
	})
	if err != nil {
		t.Fatalf("create Arthur: %v", err)
	}
	assertEnvironmentDebugOutput(
		t,
		debugOutput.String(),
		[]string{
			"Statement: UserMapper.Create",
			"INSERT INTO users",
			"Arthur",
			"<redacted>",
		},
		[]string{"arthur@example.com"},
	)

	deletedID, err := mapper.Create(ctx, CreateUserParams{
		Name:   "Deleted User",
		Email:  "deleted@example.com",
		Status: UserStatusDeleted,
	})
	if err != nil {
		t.Fatalf("create deleted user: %v", err)
	}

	got, err := mapper.GetByID(ctx, arthurID)
	if err != nil {
		t.Fatalf("get Arthur: %v", err)
	}
	if got.ID != arthurID ||
		got.Name != "Arthur" ||
		got.Email != "arthur@example.com" ||
		got.Status != UserStatusActive ||
		got.CreatedAt.IsZero() {
		t.Fatalf("unexpected mapped user: %#v", got)
	}

	foundUser, found, err := mapper.FindByEmail(ctx, "arthur@example.com")
	if err != nil {
		t.Fatalf("find Arthur by email: %v", err)
	}
	if !found || foundUser.ID != arthurID {
		t.Fatalf("find Arthur by email = (%#v, %t), want ID %d", foundUser, found, arthurID)
	}

	missingUser, found, err := mapper.FindByEmail(ctx, "missing@example.com")
	if err != nil {
		t.Fatalf("find missing user by email: %v", err)
	}
	if found || missingUser != (User{}) {
		t.Fatalf("find missing user = (%#v, %t), want zero user and false", missingUser, found)
	}

	label, err := mapper.GetLabel(ctx, arthurID)
	if err != nil {
		t.Fatalf("get Arthur label: %v", err)
	}
	if label.UserID != arthurID || label.DisplayName != "Arthur <arthur@example.com>" {
		t.Fatalf("unexpected mapped label: %#v", label)
	}

	users, err := mapper.Search(ctx, SearchUserParams{
		Keyword: "%Arthur%",
		IDs:     []int64{arthurID, deletedID},
		Limit:   10,
	})
	if err != nil {
		t.Fatalf("search active users: %v", err)
	}
	if len(users) != 1 || users[0].ID != arthurID {
		t.Fatalf("default choose branch should exclude deleted users: %#v", users)
	}

	users, err = mapper.Search(ctx, SearchUserParams{
		IDs:   []int64{0, -1},
		Limit: 10,
	})
	if err != nil {
		t.Fatalf("search with all IDs filtered out: %v", err)
	}
	if len(users) != 1 || users[0].ID != arthurID {
		t.Fatalf("all-filtered IDs should omit the ID predicate: %#v", users)
	}

	deletedStatus := UserStatusDeleted
	users, err = mapper.Search(ctx, SearchUserParams{
		Status: &deletedStatus,
		Limit:  10,
	})
	if err != nil {
		t.Fatalf("search deleted users: %v", err)
	}
	if len(users) != 1 || users[0].ID != deletedID {
		t.Fatalf("status choose branch should select deleted users: %#v", users)
	}

	activeCount, err := mapper.CountByStatus(ctx, UserStatusActive)
	if err != nil {
		t.Fatalf("count active users: %v", err)
	}
	if activeCount != 1 {
		t.Fatalf("active user count = %d, want 1", activeCount)
	}

	affected, err := mapper.UpdateStatus(ctx, arthurID, UserStatusDisabled)
	if err != nil {
		t.Fatalf("update Arthur: %v", err)
	}
	if affected != 1 {
		t.Fatalf("update affected rows = %d, want 1", affected)
	}

	updated, err := mapper.GetByID(ctx, arthurID)
	if err != nil {
		t.Fatalf("get updated Arthur: %v", err)
	}
	if updated.Status != UserStatusDisabled {
		t.Fatalf("updated status = %q, want %q", updated.Status, UserStatusDisabled)
	}

	if err := mapper.Delete(ctx, arthurID); err != nil {
		t.Fatalf("delete Arthur: %v", err)
	}
	if _, err := mapper.GetByID(ctx, arthurID); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("get deleted Arthur error = %v, want sql.ErrNoRows", err)
	}

	createEvent := logger.find("UserMapper.Create")
	if createEvent == nil {
		t.Fatal("Create SQL event was not logged")
	}
	if len(createEvent.Args) != 3 || !createEvent.Args[1].Sensitive {
		t.Fatalf("Create SQL event arguments = %#v", createEvent.Args)
	}
}

func TestUserMapperSQLiteQuotesDynamicIdentifier(t *testing.T) {
	ctx := context.Background()
	_, mapper, _ := newSQLiteMapper(t)

	for index, name := range []string{"Zulu", "Alpha"} {
		if _, err := mapper.Create(ctx, CreateUserParams{
			Name:   name,
			Email:  fmt.Sprintf("identifier-%d@example.com", index),
			Status: UserStatusActive,
		}); err != nil {
			t.Fatalf("create SQLite identifier fixture: %v", err)
		}
	}

	names, err := mapper.ListNamesOrderedBy(ctx, yourbatis.MustIdentifier("name"))
	if err != nil {
		t.Fatalf("list SQLite names by dynamic identifier: %v", err)
	}
	if got := strings.Join(names, ","); got != "Alpha,Zulu" {
		t.Fatalf("names ordered by dynamic identifier = %q, want %q", got, "Alpha,Zulu")
	}
}

func TestUserMapperSQLiteCreateMany(t *testing.T) {
	ctx := context.Background()
	db, mapper, logger := newSQLiteMapper(t)

	params := []CreateUserParams{
		{
			Name:   "Batch One",
			Email:  "batch-one@example.com",
			Status: UserStatusActive,
		},
		{
			Name:   "Batch Two",
			Email:  "batch-two@example.com",
			Status: UserStatusDisabled,
		},
	}
	affected, err := mapper.CreateMany(ctx, params)
	if err != nil {
		t.Fatalf("create many SQLite users: %v", err)
	}
	if affected != int64(len(params)) {
		t.Fatalf("create many affected rows = %d, want %d", affected, len(params))
	}
	for _, params := range params {
		created, found, err := mapper.FindByEmail(ctx, params.Email)
		if err != nil {
			t.Fatalf("find batch-created user %q: %v", params.Email, err)
		}
		if !found ||
			created.Name != params.Name ||
			created.Status != params.Status {
			t.Fatalf("batch-created user = (%#v, %t), want %#v", created, found, params)
		}
	}

	event := logger.find("UserMapper.CreateMany")
	if event == nil {
		t.Fatal("CreateMany SQL event was not logged")
	}
	if len(event.Args) != 6 ||
		!event.Args[1].Sensitive ||
		!event.Args[4].Sensitive {
		t.Fatalf("CreateMany SQL event arguments = %#v", event.Args)
	}
	createManyEvents := 0
	for _, event := range logger.events {
		if event.Statement.ID == "UserMapper.CreateMany" {
			createManyEvents++
		}
	}
	if createManyEvents != 1 {
		t.Fatalf("CreateMany SQL events = %d, want one multi-values execution", createManyEvents)
	}

	err = db.Transaction(ctx, func(tx yourbatis.Executor) error {
		_, err := NewUserMapper(tx).CreateMany(ctx, []CreateUserParams{
			{
				Name:   "Committed One",
				Email:  "committed-one@example.com",
				Status: UserStatusActive,
			},
			{
				Name:   "Committed Two",
				Email:  "committed-two@example.com",
				Status: UserStatusActive,
			},
		})
		return err
	})
	if err != nil {
		t.Fatalf("commit SQLite CreateMany transaction: %v", err)
	}
	if _, found, err := mapper.FindByEmail(ctx, "committed-two@example.com"); err != nil || !found {
		t.Fatalf("find committed CreateMany user = (found %t, error %v)", found, err)
	}

	forcedRollback := errors.New("force SQLite CreateMany rollback")
	err = db.Transaction(ctx, func(tx yourbatis.Executor) error {
		if _, err := NewUserMapper(tx).CreateMany(ctx, []CreateUserParams{
			{
				Name:   "Rolled Back One",
				Email:  "rolled-back-one@example.com",
				Status: UserStatusActive,
			},
			{
				Name:   "Rolled Back Two",
				Email:  "rolled-back-two@example.com",
				Status: UserStatusActive,
			},
		}); err != nil {
			return err
		}
		return forcedRollback
	})
	if !errors.Is(err, forcedRollback) {
		t.Fatalf("CreateMany rollback error = %v, want %v", err, forcedRollback)
	}
	if _, found, err := mapper.FindByEmail(ctx, "rolled-back-one@example.com"); err != nil || found {
		t.Fatalf("find rolled-back CreateMany user = (found %t, error %v)", found, err)
	}

	_, err = mapper.CreateMany(ctx, []CreateUserParams{
		{
			Name:   "Duplicate One",
			Email:  "duplicate@example.com",
			Status: UserStatusActive,
		},
		{
			Name:   "Duplicate Two",
			Email:  "duplicate@example.com",
			Status: UserStatusActive,
		},
	})
	var statementErr *yourbatis.StatementError
	if !errors.As(err, &statementErr) ||
		statementErr.Statement.ID != "UserMapper.CreateMany" {
		t.Fatalf("duplicate CreateMany error = %#v, want UserMapper.CreateMany StatementError", err)
	}
	if _, found, findErr := mapper.FindByEmail(ctx, "duplicate@example.com"); findErr != nil || found {
		t.Fatalf(
			"duplicate CreateMany should be atomic, lookup = (found %t, error %v)",
			found,
			findErr,
		)
	}
}

func TestUserMapperSQLiteSearchEach(t *testing.T) {
	ctx := context.Background()
	db, mapper, logger := newSQLiteMapper(t)

	activeID, err := mapper.Create(ctx, CreateUserParams{
		Name:   "Active",
		Email:  "active@example.com",
		Status: UserStatusActive,
	})
	if err != nil {
		t.Fatal(err)
	}
	disabledID, err := mapper.Create(ctx, CreateUserParams{
		Name:   "Disabled",
		Email:  "disabled@example.com",
		Status: UserStatusDisabled,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := mapper.Create(ctx, CreateUserParams{
		Name:   "Deleted",
		Email:  "deleted@example.com",
		Status: UserStatusDeleted,
	}); err != nil {
		t.Fatal(err)
	}

	var streamedIDs []int64
	err = mapper.SearchEach(ctx, SearchUserParams{}, func(current User) error {
		streamedIDs = append(streamedIDs, current.ID)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(streamedIDs) != 2 ||
		streamedIDs[0] != activeID ||
		streamedIDs[1] != disabledID {
		t.Fatalf("streamed IDs = %v, want [%d %d]", streamedIDs, activeID, disabledID)
	}

	missingStatus := UserStatus("missing")
	yieldCalls := 0
	err = mapper.SearchEach(
		ctx,
		SearchUserParams{Status: &missingStatus},
		func(User) error {
			yieldCalls++
			return nil
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if yieldCalls != 0 {
		t.Fatalf("empty stream invoked yield %d times", yieldCalls)
	}

	yieldErr := errors.New("stop SQLite stream")
	err = mapper.SearchEach(ctx, SearchUserParams{}, func(User) error {
		return yieldErr
	})
	if err != yieldErr {
		t.Fatalf("yield error = %#v, want original error %#v", err, yieldErr)
	}
	if event := logger.find("UserMapper.SearchEach"); event == nil ||
		event.Statement.Kind != yourbatis.StatementSelect {
		t.Fatalf("SearchEach log event = %#v", event)
	}

	var committedID int64
	err = db.Transaction(ctx, func(tx yourbatis.Executor) error {
		txMapper := NewUserMapper(tx)
		var err error
		committedID, err = txMapper.Create(ctx, CreateUserParams{
			Name:   "Stream Commit",
			Email:  "stream-commit@example.com",
			Status: UserStatusActive,
		})
		if err != nil {
			return err
		}
		found := false
		err = txMapper.SearchEach(ctx, SearchUserParams{}, func(current User) error {
			found = found || current.ID == committedID
			return nil
		})
		if err != nil {
			return err
		}
		if !found {
			return errors.New("stream did not observe uncommitted row")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("stream transaction commit: %v", err)
	}
	if _, err := mapper.GetByID(ctx, committedID); err != nil {
		t.Fatalf("get stream-committed user: %v", err)
	}

	var rolledBackID int64
	err = db.Transaction(ctx, func(tx yourbatis.Executor) error {
		txMapper := NewUserMapper(tx)
		var err error
		rolledBackID, err = txMapper.Create(ctx, CreateUserParams{
			Name:   "Stream Rollback",
			Email:  "stream-rollback@example.com",
			Status: UserStatusActive,
		})
		if err != nil {
			return err
		}
		return txMapper.SearchEach(ctx, SearchUserParams{}, func(User) error {
			return yieldErr
		})
	})
	if err != yieldErr {
		t.Fatalf("stream transaction rollback error = %#v, want %#v", err, yieldErr)
	}
	if _, err := mapper.GetByID(ctx, rolledBackID); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("rolled-back stream user lookup error = %v, want sql.ErrNoRows", err)
	}
}

func TestUserMapperSQLiteTransaction(t *testing.T) {
	ctx := context.Background()
	db, mapper, _ := newSQLiteMapper(t)

	forcedRollback := errors.New("force rollback")
	var rolledBackID int64
	err := db.Transaction(ctx, func(tx yourbatis.Executor) error {
		if got := yourbatis.ExecutorDatabaseID(tx); got != "sqlite" {
			t.Fatalf("transaction database ID = %q, want sqlite", got)
		}
		txMapper := NewUserMapper(tx)
		var err error
		rolledBackID, err = txMapper.Create(ctx, CreateUserParams{
			Name:   "Rolled Back",
			Email:  "rollback@example.com",
			Status: UserStatusActive,
		})
		if err != nil {
			return err
		}
		return forcedRollback
	})
	if !errors.Is(err, forcedRollback) {
		t.Fatalf("transaction error = %v, want forced rollback", err)
	}
	if _, err := mapper.GetByID(ctx, rolledBackID); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("rolled-back user lookup error = %v, want sql.ErrNoRows", err)
	}

	var committedID int64
	err = db.TransactionOptions(ctx, &sql.TxOptions{}, func(tx yourbatis.Executor) error {
		txMapper := NewUserMapper(tx)
		var err error
		committedID, err = txMapper.Create(ctx, CreateUserParams{
			Name:   "Committed",
			Email:  "committed@example.com",
			Status: UserStatusActive,
		})
		return err
	})
	if err != nil {
		t.Fatalf("commit transaction: %v", err)
	}
	if _, err := mapper.GetByID(ctx, committedID); err != nil {
		t.Fatalf("get committed user: %v", err)
	}

	if err := db.Transaction(ctx, nil); !errors.Is(err, yourbatis.ErrNilTransactionCallback) {
		t.Fatalf("nil transaction callback error = %v", err)
	}
}

func TestUserMapperSQLiteTransactionRollsBackPanic(t *testing.T) {
	ctx := context.Background()
	db, mapper, _ := newSQLiteMapper(t)

	const panicValue = "transaction panic"
	var insertedID int64
	var recovered any
	func() {
		defer func() {
			recovered = recover()
		}()

		_ = db.Transaction(ctx, func(tx yourbatis.Executor) error {
			txMapper := NewUserMapper(tx)
			var err error
			insertedID, err = txMapper.Create(ctx, CreateUserParams{
				Name:   "Panicked",
				Email:  "panicked@example.com",
				Status: UserStatusActive,
			})
			if err != nil {
				return err
			}
			panic(panicValue)
		})
	}()

	if recovered != panicValue {
		t.Fatalf("recovered panic = %v, want %q", recovered, panicValue)
	}
	if _, err := mapper.GetByID(ctx, insertedID); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("panicked transaction was not rolled back: %v", err)
	}
}

func TestUserMapperSQLiteManualTransaction(t *testing.T) {
	ctx := context.Background()
	db, mapper, _ := newSQLiteMapper(t)

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("begin transaction: %v", err)
	}
	txMapper := NewUserMapper(tx)
	committedID, err := txMapper.Create(ctx, CreateUserParams{
		Name:   "Manual Commit",
		Email:  "manual-commit@example.com",
		Status: UserStatusActive,
	})
	if err != nil {
		_ = tx.Rollback()
		t.Fatalf("create in manual transaction: %v", err)
	}
	if _, err := tx.SQLTx().ExecContext(ctx, `
		INSERT INTO users (name, email, status)
		VALUES (?, ?, ?)
	`, "SQLTx Commit", "sqltx-commit@example.com", UserStatusActive); err != nil {
		_ = tx.Rollback()
		t.Fatalf("create through SQLTx in manual transaction: %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("commit manual transaction: %v", err)
	}
	if _, err := mapper.GetByID(ctx, committedID); err != nil {
		t.Fatalf("get manually committed user: %v", err)
	}
	if _, found, err := mapper.FindByEmail(ctx, "sqltx-commit@example.com"); err != nil || !found {
		t.Fatalf("find SQLTx-committed user = (found %t, error %v)", found, err)
	}

	tx, err = db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		t.Fatalf("begin rollback transaction: %v", err)
	}
	txMapper = NewUserMapper(tx)
	rolledBackID, err := txMapper.Create(ctx, CreateUserParams{
		Name:   "Manual Rollback",
		Email:  "manual-rollback@example.com",
		Status: UserStatusActive,
	})
	if err != nil {
		_ = tx.Rollback()
		t.Fatalf("create before manual rollback: %v", err)
	}
	if _, err := tx.SQLTx().ExecContext(ctx, `
		INSERT INTO users (name, email, status)
		VALUES (?, ?, ?)
	`, "SQLTx Rollback", "sqltx-rollback@example.com", UserStatusActive); err != nil {
		_ = tx.Rollback()
		t.Fatalf("create through SQLTx before manual rollback: %v", err)
	}
	if err := tx.Rollback(); err != nil {
		t.Fatalf("rollback manual transaction: %v", err)
	}
	if _, err := mapper.GetByID(ctx, rolledBackID); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("manually rolled-back user lookup error = %v, want sql.ErrNoRows", err)
	}
	if _, found, err := mapper.FindByEmail(ctx, "sqltx-rollback@example.com"); err != nil || found {
		t.Fatalf("find SQLTx-rolled-back user = (found %t, error %v)", found, err)
	}
}

func newSQLiteMapper(t *testing.T) (*yourbatis.DB, UserMapper, *sqlEventRecorder) {
	t.Helper()

	ctx := context.Background()
	raw, err := sql.Open(
		"sqlite",
		"file:"+t.Name()+"?mode=memory&cache=shared&_time_format=sqlite&_texttotime=1",
	)
	if err != nil {
		t.Fatalf("open SQLite: %v", err)
	}
	t.Cleanup(func() {
		if err := raw.Close(); err != nil {
			t.Errorf("close SQLite: %v", err)
		}
	})

	// A single connection keeps the lifetime of this in-memory database
	// deterministic and avoids SQLite write-lock contention in tests.
	raw.SetMaxOpenConns(1)

	if err := raw.PingContext(ctx); err != nil {
		t.Fatalf("ping SQLite: %v", err)
	}

	_, err = raw.ExecContext(ctx, `
		CREATE TABLE users (
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			name       TEXT NOT NULL,
			email      TEXT NOT NULL UNIQUE,
			status     TEXT NOT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("create users table: %v", err)
	}

	logger := &sqlEventRecorder{}
	db := yourbatis.NewDB(
		raw,
		yourbatis.DialectSQLite,
		yourbatis.WithDatabaseID("sqlite"),
		yourbatis.WithLogger(logger),
	)
	return db, NewUserMapper(db), logger
}

type sqlEventRecorder struct {
	events []yourbatis.Event
}

func (logger *sqlEventRecorder) LogSQL(_ context.Context, event yourbatis.Event) {
	logger.events = append(logger.events, event)
}

func (logger *sqlEventRecorder) find(statementID string) *yourbatis.Event {
	for index := range logger.events {
		if logger.events[index].Statement.ID == statementID {
			return &logger.events[index]
		}
	}
	return nil
}
