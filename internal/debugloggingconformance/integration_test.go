package debugloggingconformance

import (
	"bytes"
	"context"
	"database/sql"
	"log"
	"os"
	"strings"
	"testing"

	"github.com/superduck-ai/yourbatis"

	_ "github.com/jackc/pgx/v5/stdlib"
	_ "modernc.org/sqlite"
)

const debugUUID = "550e8400-e29b-41d4-a716-446655440000"

func TestDebugUUIDSQLite(t *testing.T) {
	raw, err := sql.Open("sqlite", "file:debug_uuid?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("open SQLite: %v", err)
	}
	t.Cleanup(func() {
		if err := raw.Close(); err != nil {
			t.Errorf("close SQLite: %v", err)
		}
	})
	testDebugUUID(t, raw, yourbatis.DialectSQLite)
}

func TestDebugUUIDPostgres(t *testing.T) {
	dsn := os.Getenv("YOURBATIS_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("set YOURBATIS_POSTGRES_DSN to run the PostgreSQL integration test")
	}
	raw, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("open PostgreSQL: %v", err)
	}
	t.Cleanup(func() {
		if err := raw.Close(); err != nil {
			t.Errorf("close PostgreSQL: %v", err)
		}
	})
	testDebugUUID(t, raw, yourbatis.DialectPostgres)
}

func testDebugUUID(t *testing.T, raw *sql.DB, dialect yourbatis.Dialect) {
	t.Helper()
	ctx := context.Background()
	if err := raw.PingContext(ctx); err != nil {
		t.Fatalf("ping %s: %v", dialect, err)
	}
	output := captureDebugLog(t)
	db := yourbatis.NewDB(raw, dialect)
	id := UUID(debugUUID)
	got, err := NewDebugLoggingMapper(db).EchoUUID(ctx, id)
	if err != nil {
		t.Fatalf("echo UUID with %s: %v", dialect, err)
	}
	if got != debugUUID {
		t.Fatalf("echoed UUID with %s = %q, want %q", dialect, got, debugUUID)
	}
	want := `id="` + debugUUID + `"`
	if logged := output.String(); !strings.Contains(logged, want) {
		t.Fatalf("%s debug output does not contain %q:\n%s", dialect, want, logged)
	}
}

func captureDebugLog(t *testing.T) *bytes.Buffer {
	t.Helper()
	t.Setenv(yourbatis.DebugEnvironmentVariable, "true")
	output := &bytes.Buffer{}
	previousOutput := log.Writer()
	previousFlags := log.Flags()
	previousPrefix := log.Prefix()
	log.SetOutput(output)
	log.SetFlags(0)
	log.SetPrefix("")
	t.Cleanup(func() {
		log.SetOutput(previousOutput)
		log.SetFlags(previousFlags)
		log.SetPrefix(previousPrefix)
	})
	return output
}
