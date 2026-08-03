package yourbatis

import (
	"bytes"
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"log"
	"strings"
	"testing"

	_ "modernc.org/sqlite"
)

type debugStringerStruct struct {
	Value string
}

func (value debugStringerStruct) String() string {
	return "stringer:" + value.Value
}

type debugUUID struct {
	text string
}

func (value debugUUID) String() string {
	return value.text
}

func (value debugUUID) Value() (driver.Value, error) {
	return value.text, nil
}

func TestEnvironmentDebugLoggerPrintsBoundSQLAndRedactedArguments(t *testing.T) {
	t.Setenv(DebugEnvironmentVariable, "on")
	var output bytes.Buffer
	previousOutput := log.Writer()
	previousFlags := log.Flags()
	previousPrefix := log.Prefix()
	log.SetOutput(&output)
	log.SetFlags(0)
	log.SetPrefix("")
	t.Cleanup(func() {
		log.SetOutput(previousOutput)
		log.SetFlags(previousFlags)
		log.SetPrefix(previousPrefix)
	})

	var recorded *Event
	db := NewDB(
		nil,
		DialectSQLite,
		WithLogger(LoggerFunc(func(_ context.Context, event Event) {
			recorded = &event
		})),
	)
	event := Event{
		Statement: Statement{ID: "UserMapper.Find", Kind: StatementSelect},
		SQL: `
			SELECT id
			FROM users
			WHERE id = ? AND token = ?
		`,
		Args: []Argument{
			{Name: "id", Value: int64(42)},
			{Name: "token", Value: "secret-token", Sensitive: true},
		},
	}
	db.logger.LogSQL(context.Background(), event)

	logged := output.String()
	for _, expected := range []string{
		"yourbatis SQL",
		"  Statement: UserMapper.Find",
		"  SQL:\n    SELECT id\n    FROM users\n    WHERE id = ? AND token = ?",
		"  Args:\n    [0] id=42\n    [1] token=<redacted>",
	} {
		if !strings.Contains(logged, expected) {
			t.Fatalf("debug output does not contain %q:\n%s", expected, logged)
		}
	}
	if strings.Contains(logged, "secret-token") {
		t.Fatalf("debug output exposed a sensitive argument:\n%s", logged)
	}
	if recorded == nil || recorded.SQL != event.SQL || len(recorded.Args) != 2 {
		t.Fatalf("configured logger event = %#v, want the original event", recorded)
	}
}

func TestEnvironmentDebugLoggerStaysDisabledForFalseValue(t *testing.T) {
	t.Setenv(DebugEnvironmentVariable, "false")

	db := NewDB(nil, DialectSQLite)
	if db.logger != nil {
		t.Fatalf("NewDB logger = %T, want nil when %s=false", db.logger, DebugEnvironmentVariable)
	}
}

func TestDebugArgumentFormatsUUIDAsCanonicalString(t *testing.T) {
	id := debugUUID{text: "550e8400-e29b-41d4-a716-446655440000"}
	for _, testCase := range []struct {
		name  string
		value any
		want  string
	}{
		{
			name:  "value",
			value: id,
			want:  `id="550e8400-e29b-41d4-a716-446655440000"`,
		},
		{
			name:  "pointer",
			value: &id,
			want:  `id="550e8400-e29b-41d4-a716-446655440000"`,
		},
		{
			name:  "nil pointer",
			value: (*debugUUID)(nil),
			want:  "id=(*yourbatis.debugUUID)(nil)",
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			got := (debugArgument{Name: "id", Value: testCase.value}).String()
			if got != testCase.want {
				t.Fatalf("debug UUID = %q, want %q", got, testCase.want)
			}
		})
	}
}

func TestDebugArgumentFormatsNonUUIDDatabaseStringer(t *testing.T) {
	value := debugUUID{text: "not-a-uuid"}
	got := (debugArgument{Name: "id", Value: value}).String()
	want := `id="not-a-uuid"`
	if got != want {
		t.Fatalf("debug database stringer = %q, want %q", got, want)
	}
}

func TestDebugArgumentKeepsNonUUIDStringerAsGoSyntax(t *testing.T) {
	value := debugStringerStruct{Value: "visible"}
	got := (debugArgument{Name: "value", Value: value}).String()
	want := `value=yourbatis.debugStringerStruct{Value:"visible"}`
	if got != want {
		t.Fatalf("debug struct = %q, want %q", got, want)
	}
}

func TestTxImplementsExecutorAndWrapsLifecycleErrors(t *testing.T) {
	ctx := context.Background()
	raw, err := sql.Open("sqlite", "file:tx_lifecycle?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := raw.Close(); err != nil {
			t.Errorf("close SQLite: %v", err)
		}
	})

	db := NewDB(
		raw,
		DialectSQLite,
		WithDatabaseID("sqlite"),
	)
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if tx.Dialect() != DialectSQLite {
		t.Fatalf("transaction dialect = %q, want %q", tx.Dialect(), DialectSQLite)
	}
	if got := ExecutorDatabaseID(tx); got != "sqlite" {
		t.Fatalf("transaction database ID = %q, want sqlite", got)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	err = tx.Rollback()
	if !errors.Is(err, sql.ErrTxDone) ||
		!strings.Contains(err.Error(), "yourbatis: rollback transaction") {
		t.Fatalf("rollback after commit error = %v, want wrapped sql.ErrTxDone", err)
	}

	tx, err = db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := tx.Rollback(); err != nil {
		t.Fatal(err)
	}
	err = tx.Commit()
	if !errors.Is(err, sql.ErrTxDone) ||
		!strings.Contains(err.Error(), "yourbatis: commit transaction") {
		t.Fatalf("commit after rollback error = %v, want wrapped sql.ErrTxDone", err)
	}
}

func TestTransactionRejectsNilCallbackBeforeBeginning(t *testing.T) {
	db := NewDB(nil, DialectSQLite)

	err := db.Transaction(context.Background(), nil)
	if !errors.Is(err, ErrNilTransactionCallback) {
		t.Fatalf("Transaction() error = %v, want ErrNilTransactionCallback", err)
	}
}

func TestBeginTxRejectsUnsupportedDialectBeforeBeginning(t *testing.T) {
	db := NewDB(nil, DialectQuestion)

	_, err := db.BeginTx(context.Background(), nil)
	var dialectError *UnsupportedDialectError
	if !errors.As(err, &dialectError) {
		t.Fatalf("BeginTx() error = %T %v, want UnsupportedDialectError", err, err)
	}
}
