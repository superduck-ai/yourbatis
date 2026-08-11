package yourbatis

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"os"
	"strings"
	"time"
)

type Executor interface {
	Dialect() Dialect
	Query(context.Context, Statement, BoundSQL) (*sql.Rows, error)
	Exec(context.Context, Statement, BoundSQL) (sql.Result, error)
}

type DatabaseIDProvider interface {
	DatabaseID() string
}

func ExecutorDatabaseID(executor Executor) string {
	if provider, ok := executor.(DatabaseIDProvider); ok {
		return provider.DatabaseID()
	}
	return ""
}

var ErrNilTransactionCallback = errors.New("yourbatis: transaction callback is nil")

type Event struct {
	Statement Statement
	SQL       string
	Args      []Argument
	Duration  time.Duration
	Err       error
}

type Logger interface {
	LogSQL(context.Context, Event)
}

type LoggerFunc func(context.Context, Event)

func (logger LoggerFunc) LogSQL(ctx context.Context, event Event) {
	logger(ctx, event)
}

// DebugEnvironmentVariable enables SQL and argument logging when NewDB is
// created. Supported enabled values are 1, true, yes, and on, ignoring case.
const DebugEnvironmentVariable = "YOURBATIS_DEBUG"

type debugArgument struct {
	Name      string
	Value     any
	Sensitive bool
}

type debugDatabaseStringer interface {
	fmt.Stringer
	driver.Valuer
}

func (argument debugArgument) String() string {
	if argument.Sensitive {
		if argument.Name == "" {
			return "<redacted>"
		}
		return argument.Name + "=<redacted>"
	}
	if argument.Name == "" {
		return formatDebugValue(argument.Value)
	}
	return argument.Name + "=" + formatDebugValue(argument.Value)
}

func formatDebugValue(value any) string {
	if current, ok := value.(debugDatabaseStringer); ok {
		formatted := fmt.Sprintf("%q", current)
		if formatted != "<nil>" {
			return formatted
		}
	}
	return fmt.Sprintf("%#v", value)
}

type environmentDebugLogger struct{}

func (environmentDebugLogger) LogSQL(_ context.Context, event Event) {
	log.Print(formatDebugEvent(event))
}

type loggerGroup []Logger

func (group loggerGroup) LogSQL(ctx context.Context, event Event) {
	for _, logger := range group {
		if logger != nil {
			logger.LogSQL(ctx, event)
		}
	}
}

func debugArguments(arguments []Argument) []debugArgument {
	result := make([]debugArgument, len(arguments))
	for index, argument := range arguments {
		result[index] = debugArgument{
			Name:      argument.Name,
			Value:     argument.Value,
			Sensitive: argument.Sensitive,
		}
	}
	return result
}

func formatDebugEvent(event Event) string {
	var output strings.Builder
	output.WriteString("yourbatis SQL")
	fmt.Fprintf(&output, "\n  Statement: %s", event.Statement.ID)
	fmt.Fprintf(&output, "\n  Kind: %s", event.Statement.Kind)
	fmt.Fprintf(&output, "\n  Source: %s", event.Statement.Source)
	output.WriteString("\n  SQL:")
	lines := debugSQLLines(event.SQL)
	if len(lines) == 0 {
		output.WriteString("\n    <empty>")
	} else {
		for _, line := range lines {
			output.WriteString("\n    ")
			output.WriteString(line)
		}
	}
	output.WriteString("\n  Args:")
	arguments := debugArguments(event.Args)
	if len(arguments) == 0 {
		output.WriteString(" []")
	} else {
		for index, argument := range arguments {
			fmt.Fprintf(&output, "\n    [%d] %s", index, argument)
		}
	}
	fmt.Fprintf(&output, "\n  Duration: %s", event.Duration)
	if event.Err != nil {
		fmt.Fprintf(&output, "\n  Error: %v", event.Err)
	}
	return output.String()
}

func debugSQLLines(value string) []string {
	value = strings.ReplaceAll(value, "\r\n", "\n")
	value = strings.ReplaceAll(value, "\r", "\n")
	lines := strings.Split(value, "\n")
	for len(lines) > 0 && strings.TrimSpace(lines[0]) == "" {
		lines = lines[1:]
	}
	for len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "" {
		lines = lines[:len(lines)-1]
	}
	if len(lines) == 0 {
		return nil
	}

	commonIndent := -1
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		indent := len(line) - len(strings.TrimLeft(line, " \t"))
		if commonIndent < 0 || indent < commonIndent {
			commonIndent = indent
		}
	}
	if commonIndent > 0 {
		for index := range lines {
			if len(lines[index]) >= commonIndent {
				lines[index] = lines[index][commonIndent:]
			}
		}
	}
	return lines
}

func environmentDebugEnabled() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(DebugEnvironmentVariable))) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

type SlogLogger struct {
	Logger *slog.Logger
}

func (logger SlogLogger) LogSQL(ctx context.Context, event Event) {
	target := logger.Logger
	if target == nil {
		target = slog.Default()
	}
	attributes := []any{
		"statement", event.Statement.ID,
		"kind", event.Statement.Kind,
		"source", event.Statement.Source,
		"sql", event.SQL,
		"duration", event.Duration,
	}
	if event.Err != nil {
		attributes = append(attributes, "error", event.Err)
		target.ErrorContext(ctx, "yourbatis statement failed", attributes...)
		return
	}
	target.DebugContext(ctx, "yourbatis statement", attributes...)
}

type dbtx interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}

type DB struct {
	raw        *sql.DB
	dialect    Dialect
	databaseID string
	logger     Logger
}

type Option func(*DB)

func WithLogger(logger Logger) Option {
	return func(db *DB) {
		db.logger = logger
	}
}

func WithDatabaseID(databaseID string) Option {
	return func(db *DB) {
		db.databaseID = databaseID
	}
}

func NewDB(raw *sql.DB, dialect Dialect, options ...Option) *DB {
	db := &DB{raw: raw, dialect: dialect}
	for _, option := range options {
		if option != nil {
			option(db)
		}
	}
	if environmentDebugEnabled() {
		db.logger = loggerGroup{
			environmentDebugLogger{},
			db.logger,
		}
	}
	return db
}

func (db *DB) Dialect() Dialect {
	return db.dialect
}

func (db *DB) DatabaseID() string {
	return db.databaseID
}

// SQLDB returns the underlying database/sql database for integrations that
// require *sql.DB. The returned database remains owned by DB; callers must not
// close it independently.
func (db *DB) SQLDB() *sql.DB {
	return db.raw
}

// Close closes the underlying database/sql database.
func (db *DB) Close() error {
	if db == nil || db.raw == nil {
		return nil
	}
	return db.raw.Close()
}

func (db *DB) Query(ctx context.Context, statement Statement, bound BoundSQL) (*sql.Rows, error) {
	if err := validateRuntimeDialect(db.dialect); err != nil {
		return nil, err
	}
	return query(ctx, db.raw, db.logger, statement, bound)
}

func (db *DB) Exec(ctx context.Context, statement Statement, bound BoundSQL) (sql.Result, error) {
	if err := validateRuntimeDialect(db.dialect); err != nil {
		return nil, err
	}
	return exec(ctx, db.raw, db.logger, statement, bound)
}

// Transaction executes fn in a database transaction.
//
// A nil return value from fn commits the transaction. An error return value
// rolls it back. If fn panics, Transaction rolls the transaction back and
// resumes the panic.
func (db *DB) Transaction(
	ctx context.Context,
	fn func(Executor) error,
) error {
	return db.TransactionOptions(ctx, nil, fn)
}

// TransactionOptions executes fn in a database transaction with options.
func (db *DB) TransactionOptions(
	ctx context.Context,
	options *sql.TxOptions,
	fn func(Executor) error,
) (err error) {
	if fn == nil {
		return ErrNilTransactionCallback
	}

	tx, err := db.BeginTx(ctx, options)
	if err != nil {
		return err
	}

	finished := false
	defer func() {
		recovered := recover()
		if !finished {
			rollbackErr := tx.Rollback()
			if recovered == nil &&
				rollbackErr != nil &&
				!errors.Is(rollbackErr, sql.ErrTxDone) {
				if err == nil {
					err = rollbackErr
				} else {
					err = errors.Join(err, rollbackErr)
				}
			}
		}
		if recovered != nil {
			panic(recovered)
		}
	}()

	if err = fn(tx); err != nil {
		return err
	}
	if err = tx.Commit(); err != nil {
		return err
	}
	finished = true
	return nil
}

// BeginTx begins a transaction for callers that need to manage its lifecycle
// manually. The returned Tx implements Executor and can be passed directly to
// generated Mapper constructors.
func (db *DB) BeginTx(
	ctx context.Context,
	options *sql.TxOptions,
) (*Tx, error) {
	if err := validateRuntimeDialect(db.dialect); err != nil {
		return nil, err
	}
	raw, err := db.raw.BeginTx(ctx, options)
	if err != nil {
		return nil, fmt.Errorf("yourbatis: begin transaction: %w", err)
	}
	return &Tx{
		raw:        raw,
		dialect:    db.dialect,
		databaseID: db.databaseID,
		logger:     db.logger,
	}, nil
}

// Tx is a database transaction and an Executor for generated Mappers.
type Tx struct {
	raw        *sql.Tx
	dialect    Dialect
	databaseID string
	logger     Logger
}

var (
	_ Executor = (*DB)(nil)
	_ Executor = (*Tx)(nil)
)

func (tx *Tx) Dialect() Dialect {
	return tx.dialect
}

func (tx *Tx) DatabaseID() string {
	return tx.databaseID
}

// SQLTx returns the underlying database/sql transaction for integrations
// requiring *sql.Tx.
//
// The returned transaction remains owned by Tx. Callers must not call Commit
// or Rollback and must not retain it after the Yourbatis transaction finishes.
func (tx *Tx) SQLTx() *sql.Tx {
	return tx.raw
}

func (tx *Tx) Query(
	ctx context.Context,
	statement Statement,
	bound BoundSQL,
) (*sql.Rows, error) {
	if err := validateRuntimeDialect(tx.dialect); err != nil {
		return nil, err
	}
	return query(ctx, tx.raw, tx.logger, statement, bound)
}

func (tx *Tx) Exec(
	ctx context.Context,
	statement Statement,
	bound BoundSQL,
) (sql.Result, error) {
	if err := validateRuntimeDialect(tx.dialect); err != nil {
		return nil, err
	}
	return exec(ctx, tx.raw, tx.logger, statement, bound)
}

// Commit commits the transaction.
func (tx *Tx) Commit() error {
	if err := tx.raw.Commit(); err != nil {
		return fmt.Errorf("yourbatis: commit transaction: %w", err)
	}
	return nil
}

// Rollback aborts the transaction.
func (tx *Tx) Rollback() error {
	if err := tx.raw.Rollback(); err != nil {
		return fmt.Errorf("yourbatis: rollback transaction: %w", err)
	}
	return nil
}

func validateRuntimeDialect(dialect Dialect) error {
	if dialect.Supported() {
		return nil
	}
	return &UnsupportedDialectError{Dialect: dialect}
}

func query(
	ctx context.Context,
	raw dbtx,
	logger Logger,
	statement Statement,
	bound BoundSQL,
) (*sql.Rows, error) {
	start := time.Now()
	rows, err := raw.QueryContext(ctx, bound.SQL, bound.Values()...)
	logEvent(ctx, logger, statement, bound, start, err)
	return rows, err
}

func exec(
	ctx context.Context,
	raw dbtx,
	logger Logger,
	statement Statement,
	bound BoundSQL,
) (sql.Result, error) {
	start := time.Now()
	result, err := raw.ExecContext(ctx, bound.SQL, bound.Values()...)
	logEvent(ctx, logger, statement, bound, start, err)
	return result, err
}

func logEvent(
	ctx context.Context,
	logger Logger,
	statement Statement,
	bound BoundSQL,
	start time.Time,
	err error,
) {
	if logger == nil {
		return
	}
	logger.LogSQL(ctx, Event{
		Statement: statement,
		SQL:       bound.SQL,
		Args:      append([]Argument(nil), bound.Args...),
		Duration:  time.Since(start),
		Err:       err,
	})
}
