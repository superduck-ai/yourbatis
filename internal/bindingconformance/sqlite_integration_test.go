package bindingconformance

import (
	"context"
	"database/sql"
	"testing"

	yourbatis "github.com/superduck-ai/yourbatis"

	_ "modernc.org/sqlite"
)

func TestBindingTestMigrationsSQLite(t *testing.T) {
	ctx := context.Background()
	raw, err := sql.Open("sqlite", "file:binding_test_migrations?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := raw.Close(); err != nil {
			t.Errorf("close SQLite: %v", err)
		}
	})
	raw.SetMaxOpenConns(1)

	if _, err := raw.ExecContext(ctx, bindingFixtureSQL); err != nil {
		t.Fatalf("prepare SQLite binding fixture: %v", err)
	}
	runMigratedBindingTests(t, yourbatis.NewDB(raw, yourbatis.DialectSQLite))
}

const bindingFixtureSQL = `
CREATE TABLE blog (
    id    BIGINT PRIMARY KEY,
    title TEXT NOT NULL
);
CREATE TABLE post (
    id      BIGINT PRIMARY KEY,
    blog_id BIGINT NOT NULL,
    subject TEXT NOT NULL,
    body    TEXT NOT NULL
);
CREATE TABLE author (
    id       BIGINT PRIMARY KEY,
    username TEXT NOT NULL,
    password TEXT NOT NULL,
    email    TEXT NOT NULL,
    bio      TEXT NOT NULL
);

INSERT INTO blog (id, title) VALUES
    (1, 'Jim Business'),
    (2, 'Mary Business');
INSERT INTO post (id, blog_id, subject, body) VALUES
    (1, 1, 'alpha',   'apple'),
    (2, 2, 'beta',    'banana'),
    (3, 1, 'gamma',   'carrot'),
    (4, 2, 'delta',   'date'),
    (5, 1, 'epsilon', 'avocado');
INSERT INTO author (id, username, password, email, bio) VALUES
    (101, 'jim', '********', 'jim@ibatis.apache.org', '');
`
