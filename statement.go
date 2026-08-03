package yourbatis

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// StatementKind describes the database operation represented by a mapper method.
type StatementKind string

const (
	StatementSelect StatementKind = "select"
	StatementInsert StatementKind = "insert"
	StatementUpdate StatementKind = "update"
	StatementDelete StatementKind = "delete"
)

// Statement identifies generated SQL in logs and errors.
type Statement struct {
	ID     string
	Source string
	Kind   StatementKind
}

var (
	ErrTooManyRows = errors.New("yourbatis: expected one row, got more than one")
	ErrNilYield    = errors.New("yourbatis: yield callback is nil")
)

// UnsupportedDialectError reports a dialect outside the project's supported
// PostgreSQL, SQLite, and MySQL scope.
type UnsupportedDialectError struct {
	Dialect Dialect
}

func (err *UnsupportedDialectError) Error() string {
	return fmt.Sprintf("yourbatis: unsupported database dialect %q", err.Dialect)
}

// UnsupportedFeatureError reports a result mode that the selected dialect
// cannot execute safely.
type UnsupportedFeatureError struct {
	Dialect   Dialect
	Feature   Feature
	Statement Statement
}

func (err *UnsupportedFeatureError) Error() string {
	switch err.Feature {
	case FeatureDMLReturning:
		return fmt.Sprintf(
			"yourbatis: dialect %s does not support DML returning for statement %s",
			err.Dialect,
			err.Statement.ID,
		)
	case FeatureLastInsertID:
		return fmt.Sprintf(
			"yourbatis: dialect %s does not support LastInsertId for statement %s; use result=\"returning\"",
			err.Dialect,
			err.Statement.ID,
		)
	default:
		return fmt.Sprintf(
			"yourbatis: dialect %s does not support feature %s for statement %s",
			err.Dialect,
			err.Feature,
			err.Statement.ID,
		)
	}
}

// StatementError adds mapper identity to an execution or scan error.
type StatementError struct {
	Statement Statement
	Err       error
}

func (err *StatementError) Error() string {
	return fmt.Sprintf("yourbatis statement %s (%s): %v", err.Statement.ID, err.Statement.Source, err.Err)
}

func (err *StatementError) Unwrap() error {
	return err.Err
}

// Scanner is implemented by *sql.Row and *sql.Rows.
type Scanner interface {
	Scan(dest ...any) error
}

// ScanFunc is generated for each result type/result map.
type ScanFunc[T any] func(Scanner) (T, error)

func QueryOne[T any](
	ctx context.Context,
	executor Executor,
	statement Statement,
	bound BoundSQL,
	scan ScanFunc[T],
) (T, error) {
	var zero T
	value, found, err := queryAtMostOne(ctx, executor, statement, bound, scan)
	if err != nil {
		return zero, err
	}
	if !found {
		return zero, wrapStatementError(statement, sql.ErrNoRows)
	}
	return value, nil
}

func queryAtMostOne[T any](
	ctx context.Context,
	executor Executor,
	statement Statement,
	bound BoundSQL,
	scan ScanFunc[T],
) (T, bool, error) {
	var zero T
	if err := requireDMLReturning(executor, statement); err != nil {
		return zero, false, wrapStatementError(statement, err)
	}
	rows, err := executor.Query(ctx, statement, bound)
	if err != nil {
		return zero, false, wrapStatementError(statement, err)
	}
	defer rows.Close()

	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return zero, false, wrapStatementError(statement, err)
		}
		return zero, false, nil
	}

	value, err := scan(rows)
	if err != nil {
		return zero, false, wrapStatementError(statement, err)
	}
	if rows.Next() {
		return zero, false, wrapStatementError(statement, ErrTooManyRows)
	}
	if err := rows.Err(); err != nil {
		return zero, false, wrapStatementError(statement, err)
	}
	return value, true, nil
}

func QueryMany[T any](
	ctx context.Context,
	executor Executor,
	statement Statement,
	bound BoundSQL,
	scan ScanFunc[T],
) ([]T, error) {
	if err := requireDMLReturning(executor, statement); err != nil {
		return nil, wrapStatementError(statement, err)
	}
	rows, err := executor.Query(ctx, statement, bound)
	if err != nil {
		return nil, wrapStatementError(statement, err)
	}
	defer rows.Close()

	result := make([]T, 0)
	for rows.Next() {
		value, err := scan(rows)
		if err != nil {
			return nil, wrapStatementError(statement, err)
		}
		result = append(result, value)
	}
	if err := rows.Err(); err != nil {
		return nil, wrapStatementError(statement, err)
	}
	return result, nil
}

func SelectOne[T any](
	ctx context.Context,
	executor Executor,
	statement Statement,
	bound BoundSQL,
	scan ScanFunc[T],
) (T, error) {
	return QueryOne(ctx, executor, statement, bound, scan)
}

func SelectOptional[T any](
	ctx context.Context,
	executor Executor,
	statement Statement,
	bound BoundSQL,
	scan ScanFunc[T],
) (T, bool, error) {
	return queryAtMostOne(ctx, executor, statement, bound, scan)
}

func SelectMany[T any](
	ctx context.Context,
	executor Executor,
	statement Statement,
	bound BoundSQL,
	scan ScanFunc[T],
) ([]T, error) {
	return QueryMany(ctx, executor, statement, bound, scan)
}

// SelectEach scans each selected row and passes it synchronously to yield.
// A yield error stops iteration and is returned unchanged.
func SelectEach[T any](
	ctx context.Context,
	executor Executor,
	statement Statement,
	bound BoundSQL,
	scan ScanFunc[T],
	yield func(T) error,
) error {
	if yield == nil {
		return wrapStatementError(statement, ErrNilYield)
	}
	if err := validateRuntimeDialect(executor.Dialect()); err != nil {
		return wrapStatementError(statement, err)
	}
	rows, err := executor.Query(ctx, statement, bound)
	if err != nil {
		return wrapStatementError(statement, err)
	}
	defer rows.Close()

	for rows.Next() {
		value, err := scan(rows)
		if err != nil {
			return wrapStatementError(statement, err)
		}
		if err := yield(value); err != nil {
			return err
		}
	}
	if err := rows.Err(); err != nil {
		return wrapStatementError(statement, err)
	}
	return nil
}

func Execute(
	ctx context.Context,
	executor Executor,
	statement Statement,
	bound BoundSQL,
) error {
	if err := validateRuntimeDialect(executor.Dialect()); err != nil {
		return wrapStatementError(statement, err)
	}
	_, err := executor.Exec(ctx, statement, bound)
	return wrapStatementError(statement, err)
}

func ExecuteRows(
	ctx context.Context,
	executor Executor,
	statement Statement,
	bound BoundSQL,
) (int64, error) {
	if err := validateRuntimeDialect(executor.Dialect()); err != nil {
		return 0, wrapStatementError(statement, err)
	}
	result, err := executor.Exec(ctx, statement, bound)
	if err != nil {
		return 0, wrapStatementError(statement, err)
	}
	rows, err := result.RowsAffected()
	return rows, wrapStatementError(statement, err)
}

func ExecuteLastInsertID(
	ctx context.Context,
	executor Executor,
	statement Statement,
	bound BoundSQL,
) (int64, error) {
	if err := validateRuntimeDialect(executor.Dialect()); err != nil {
		return 0, wrapStatementError(statement, err)
	}
	if !executor.Dialect().Supports(FeatureLastInsertID) {
		return 0, wrapStatementError(statement, &UnsupportedFeatureError{
			Dialect:   executor.Dialect(),
			Feature:   FeatureLastInsertID,
			Statement: statement,
		})
	}
	result, err := executor.Exec(ctx, statement, bound)
	if err != nil {
		return 0, wrapStatementError(statement, err)
	}
	id, err := result.LastInsertId()
	return id, wrapStatementError(statement, err)
}

func requireDMLReturning(executor Executor, statement Statement) error {
	if err := validateRuntimeDialect(executor.Dialect()); err != nil {
		return err
	}
	if statement.Kind == StatementSelect {
		return nil
	}
	if executor.Dialect().Supports(FeatureDMLReturning) {
		return nil
	}
	return &UnsupportedFeatureError{
		Dialect:   executor.Dialect(),
		Feature:   FeatureDMLReturning,
		Statement: statement,
	}
}

func wrapStatementError(statement Statement, err error) error {
	if err == nil {
		return nil
	}
	return &StatementError{Statement: statement, Err: err}
}
