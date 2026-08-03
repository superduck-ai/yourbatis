package user

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"strings"
	"testing"

	mysqlDriver "github.com/go-sql-driver/mysql"

	yourbatis "github.com/superduck-ai/yourbatis"
)

func TestUserMapperMySQLCRUDAndLastInsertID(t *testing.T) {
	dsn := os.Getenv("YOURBATIS_MYSQL_DSN")
	if dsn == "" {
		t.Skip("set YOURBATIS_MYSQL_DSN to run the MySQL integration test")
	}

	config, err := mysqlDriver.ParseDSN(dsn)
	if err != nil {
		t.Fatalf("parse MySQL DSN: %v", err)
	}
	config.ParseTime = true

	ctx := context.Background()
	raw, err := sql.Open("mysql", config.FormatDSN())
	if err != nil {
		t.Fatalf("open MySQL: %v", err)
	}
	t.Cleanup(func() {
		if err := raw.Close(); err != nil {
			t.Errorf("close MySQL: %v", err)
		}
	})

	// Temporary tables are connection-local. One pooled connection keeps the
	// fixture isolated from persistent application tables.
	raw.SetMaxOpenConns(1)
	if err := raw.PingContext(ctx); err != nil {
		t.Fatalf("ping MySQL: %v", err)
	}
	if _, err := raw.ExecContext(ctx, `
		CREATE TEMPORARY TABLE users (
			id         BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
			name       VARCHAR(255) NOT NULL,
			email      VARCHAR(255) NOT NULL UNIQUE,
			status     VARCHAR(32) NOT NULL,
			created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
			updated_at DATETIME(6) NULL
		) ENGINE=InnoDB
	`); err != nil {
		t.Fatalf("create MySQL fixture table: %v", err)
	}

	db := yourbatis.NewDB(
		raw,
		yourbatis.DialectMySQL,
		yourbatis.WithDatabaseID("mysql"),
	)
	mapper := NewUserMapper(db)

	id, err := mapper.Create(ctx, CreateUserParams{
		Name:   "MySQL User",
		Email:  "mysql@example.com",
		Status: UserStatusActive,
	})
	if err != nil {
		t.Fatalf("create with LastInsertId: %v", err)
	}
	if id == 0 {
		t.Fatal("created ID = 0")
	}

	created, err := mapper.GetByID(ctx, id)
	if err != nil {
		t.Fatalf("get created user: %v", err)
	}
	if created.ID != id ||
		created.Name != "MySQL User" ||
		created.Status != UserStatusActive {
		t.Fatalf("created user = %#v", created)
	}
	var streamedIDs []int64
	err = mapper.SearchEach(ctx, SearchUserParams{}, func(current User) error {
		streamedIDs = append(streamedIDs, current.ID)
		return nil
	})
	if err != nil {
		t.Fatalf("stream MySQL users: %v", err)
	}
	if len(streamedIDs) != 1 || streamedIDs[0] != id {
		t.Fatalf("streamed MySQL IDs = %v, want [%d]", streamedIDs, id)
	}

	foundUser, found, err := mapper.FindByEmail(ctx, "mysql@example.com")
	if err != nil {
		t.Fatalf("find created user by email: %v", err)
	}
	if !found || foundUser.ID != id {
		t.Fatalf("find created user = (%#v, %t), want ID %d", foundUser, found, id)
	}

	missingUser, found, err := mapper.FindByEmail(ctx, "missing@example.com")
	if err != nil {
		t.Fatalf("find missing user by email: %v", err)
	}
	if found || missingUser != (User{}) {
		t.Fatalf("find missing user = (%#v, %t), want zero user and false", missingUser, found)
	}

	batchParams := []CreateUserParams{
		{
			Name:   "MySQL Batch One",
			Email:  "mysql-batch-one@example.com",
			Status: UserStatusActive,
		},
		{
			Name:   "MySQL Batch Two",
			Email:  "mysql-batch-two@example.com",
			Status: UserStatusDisabled,
		},
	}
	affected, err := mapper.CreateMany(ctx, batchParams)
	if err != nil {
		t.Fatalf("create many MySQL users: %v", err)
	}
	if affected != int64(len(batchParams)) {
		t.Fatalf("CreateMany affected rows = %d, want %d", affected, len(batchParams))
	}
	for _, params := range batchParams {
		created, found, err := mapper.FindByEmail(ctx, params.Email)
		if err != nil {
			t.Fatalf("find batch-created MySQL user %q: %v", params.Email, err)
		}
		if !found ||
			created.Name != params.Name ||
			created.Status != params.Status {
			t.Fatalf("batch-created MySQL user = (%#v, %t), want %#v", created, found, params)
		}
	}
	names, err := mapper.ListNamesOrderedBy(ctx, yourbatis.MustIdentifier("name"))
	if err != nil {
		t.Fatalf("list MySQL names by dynamic identifier: %v", err)
	}
	if got := strings.Join(names, ","); got != "MySQL Batch One,MySQL Batch Two,MySQL User" {
		t.Fatalf("names ordered by dynamic identifier = %q", got)
	}

	affected, err = mapper.UpdateStatus(ctx, id, UserStatusDisabled)
	if err != nil {
		t.Fatalf("update status: %v", err)
	}
	if affected != 1 {
		t.Fatalf("updated rows = %d, want 1", affected)
	}

	forcedRollback := errors.New("force MySQL rollback")
	var rolledBackID int64
	err = db.Transaction(ctx, func(tx yourbatis.Executor) error {
		txMapper := NewUserMapper(tx)
		var err error
		rolledBackID, err = txMapper.Create(ctx, CreateUserParams{
			Name:   "MySQL Rollback",
			Email:  "mysql-rollback@example.com",
			Status: UserStatusActive,
		})
		if err != nil {
			return err
		}
		found := false
		if err := txMapper.SearchEach(ctx, SearchUserParams{}, func(current User) error {
			found = found || current.ID == rolledBackID
			return nil
		}); err != nil {
			return err
		}
		if !found {
			return errors.New("stream did not observe uncommitted MySQL row")
		}
		return forcedRollback
	})
	if !errors.Is(err, forcedRollback) {
		t.Fatalf("transaction error = %v, want forced rollback", err)
	}
	if _, err := mapper.GetByID(ctx, rolledBackID); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("rolled-back user lookup error = %v, want sql.ErrNoRows", err)
	}

	if err := mapper.Delete(ctx, id); err != nil {
		t.Fatalf("delete user: %v", err)
	}
	if _, err := mapper.GetByID(ctx, id); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("deleted user lookup error = %v, want sql.ErrNoRows", err)
	}
}
