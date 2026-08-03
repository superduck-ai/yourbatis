package yourbatis

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"sync"
	"testing"

	_ "modernc.org/sqlite"
)

type capabilityTestExecutor struct {
	dialect     Dialect
	queryCalled bool
	execCalled  bool
	result      sql.Result
	execErr     error
}

func (executor *capabilityTestExecutor) Dialect() Dialect {
	return executor.dialect
}

func (executor *capabilityTestExecutor) Query(
	context.Context,
	Statement,
	BoundSQL,
) (*sql.Rows, error) {
	executor.queryCalled = true
	return nil, errors.New("unexpected query")
}

func (executor *capabilityTestExecutor) Exec(
	context.Context,
	Statement,
	BoundSQL,
) (sql.Result, error) {
	executor.execCalled = true
	return executor.result, executor.execErr
}

type capabilityTestResult struct {
	lastID int64
	rows   int64
}

func (result capabilityTestResult) LastInsertId() (int64, error) {
	return result.lastID, nil
}

func (result capabilityTestResult) RowsAffected() (int64, error) {
	return result.rows, nil
}

var (
	selectEachRowsError           = errors.New("forced rows iteration error")
	registerSelectEachErrorDriver sync.Once
)

type selectEachErrorDriver struct{}

func (selectEachErrorDriver) Open(string) (driver.Conn, error) {
	return selectEachErrorConn{}, nil
}

type selectEachErrorConn struct{}

func (selectEachErrorConn) Prepare(string) (driver.Stmt, error) {
	return nil, driver.ErrSkip
}

func (selectEachErrorConn) Close() error {
	return nil
}

func (selectEachErrorConn) Begin() (driver.Tx, error) {
	return nil, driver.ErrSkip
}

func (selectEachErrorConn) QueryContext(
	context.Context,
	string,
	[]driver.NamedValue,
) (driver.Rows, error) {
	return &selectEachErrorRows{}, nil
}

type selectEachErrorRows struct {
	returned bool
}

func (*selectEachErrorRows) Columns() []string {
	return []string{"value"}
}

func (*selectEachErrorRows) Close() error {
	return nil
}

func (rows *selectEachErrorRows) Next(dest []driver.Value) error {
	if rows.returned {
		return selectEachRowsError
	}
	rows.returned = true
	dest[0] = int64(7)
	return nil
}

func TestExecuteLastInsertIDRejectsPostgresBeforeExec(t *testing.T) {
	executor := &capabilityTestExecutor{dialect: DialectPostgres}
	statement := Statement{ID: "UserMapper.Create", Kind: StatementInsert}

	_, err := ExecuteLastInsertID(context.Background(), executor, statement, BoundSQL{})
	if err == nil {
		t.Fatal("ExecuteLastInsertID() error = nil")
	}
	if executor.execCalled {
		t.Fatal("Exec was called before rejecting unsupported LastInsertId")
	}

	var featureError *UnsupportedFeatureError
	if !errors.As(err, &featureError) {
		t.Fatalf("error = %T %v, want UnsupportedFeatureError", err, err)
	}
	if featureError.Feature != FeatureLastInsertID ||
		featureError.Dialect != DialectPostgres {
		t.Fatalf("feature error = %#v", featureError)
	}
}

func TestQueryOneRejectsMySQLDMLReturningBeforeQuery(t *testing.T) {
	executor := &capabilityTestExecutor{dialect: DialectMySQL}
	statement := Statement{ID: "UserMapper.Create", Kind: StatementInsert}

	_, err := QueryOne(
		context.Background(),
		executor,
		statement,
		BoundSQL{},
		func(Scanner) (int64, error) { return 0, nil },
	)
	if err == nil {
		t.Fatal("QueryOne() error = nil")
	}
	if executor.queryCalled {
		t.Fatal("Query was called before rejecting unsupported DML returning")
	}

	var featureError *UnsupportedFeatureError
	if !errors.As(err, &featureError) {
		t.Fatalf("error = %T %v, want UnsupportedFeatureError", err, err)
	}
	if featureError.Feature != FeatureDMLReturning ||
		featureError.Dialect != DialectMySQL {
		t.Fatalf("feature error = %#v", featureError)
	}
}

func TestSelectOptionalDistinguishesZeroOneAndManyRows(t *testing.T) {
	ctx := context.Background()
	raw, err := sql.Open("sqlite", "file:select_optional?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := raw.Close(); err != nil {
			t.Errorf("close SQLite: %v", err)
		}
	})
	raw.SetMaxOpenConns(1)

	if _, err := raw.ExecContext(ctx, `
		CREATE TABLE users (id INTEGER PRIMARY KEY);
		INSERT INTO users (id) VALUES (1), (2);
	`); err != nil {
		t.Fatal(err)
	}

	db := NewDB(raw, DialectSQLite)
	statement := Statement{
		ID:     "UserMapper.Find",
		Source: "user_mapper.xml:10",
		Kind:   StatementSelect,
	}
	scanID := func(row Scanner) (int64, error) {
		var id int64
		err := row.Scan(&id)
		return id, err
	}

	value, found, err := SelectOptional(
		ctx,
		db,
		statement,
		BoundSQL{SQL: "SELECT id FROM users WHERE id = -1"},
		scanID,
	)
	if err != nil || found || value != 0 {
		t.Fatalf("zero rows = (%d, %t, %v), want (0, false, nil)", value, found, err)
	}

	value, found, err = SelectOptional(
		ctx,
		db,
		statement,
		BoundSQL{SQL: "SELECT id FROM users WHERE id = 1"},
		scanID,
	)
	if err != nil || !found || value != 1 {
		t.Fatalf("one row = (%d, %t, %v), want (1, true, nil)", value, found, err)
	}

	value, found, err = SelectOptional(
		ctx,
		db,
		statement,
		BoundSQL{SQL: "SELECT id FROM users ORDER BY id"},
		scanID,
	)
	if !errors.Is(err, ErrTooManyRows) || found || value != 0 {
		t.Fatalf(
			"many rows = (%d, %t, %v), want (0, false, ErrTooManyRows)",
			value,
			found,
			err,
		)
	}
	var statementError *StatementError
	if !errors.As(err, &statementError) || statementError.Statement.ID != statement.ID {
		t.Fatalf("many-row error = %#v, want StatementError for %s", err, statement.ID)
	}
}

func TestSelectEachRejectsNilYieldBeforeQuery(t *testing.T) {
	executor := &capabilityTestExecutor{dialect: DialectSQLite}
	statement := Statement{ID: "UserMapper.SearchEach", Kind: StatementSelect}

	err := SelectEach[int64](
		context.Background(),
		executor,
		statement,
		BoundSQL{},
		func(Scanner) (int64, error) { return 0, nil },
		nil,
	)
	if !errors.Is(err, ErrNilYield) {
		t.Fatalf("SelectEach() error = %v, want ErrNilYield", err)
	}
	if executor.queryCalled {
		t.Fatal("Query was called before rejecting a nil yield callback")
	}
	var statementError *StatementError
	if !errors.As(err, &statementError) || statementError.Statement.ID != statement.ID {
		t.Fatalf("error = %#v, want StatementError for %s", err, statement.ID)
	}
}

func TestSelectEachVisitsRowsAndReleasesConnection(t *testing.T) {
	ctx := context.Background()
	raw, err := sql.Open("sqlite", "file:select_each?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := raw.Close(); err != nil {
			t.Errorf("close SQLite: %v", err)
		}
	})
	raw.SetMaxOpenConns(1)

	if _, err := raw.ExecContext(ctx, `
		CREATE TABLE users (id INTEGER PRIMARY KEY);
		INSERT INTO users (id) VALUES (1), (2), (3);
	`); err != nil {
		t.Fatal(err)
	}

	db := NewDB(raw, DialectSQLite)
	statement := Statement{
		ID:     "UserMapper.SearchEach",
		Source: "user_mapper.xml:10",
		Kind:   StatementSelect,
	}
	scanID := func(row Scanner) (int64, error) {
		var id int64
		err := row.Scan(&id)
		return id, err
	}

	var visited []int64
	err = SelectEach(
		ctx,
		db,
		statement,
		BoundSQL{SQL: "SELECT id FROM users ORDER BY id"},
		scanID,
		func(id int64) error {
			visited = append(visited, id)
			return nil
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(visited) != 3 || visited[0] != 1 || visited[1] != 2 || visited[2] != 3 {
		t.Fatalf("visited IDs = %v, want [1 2 3]", visited)
	}

	yieldErr := errors.New("stop processing")
	calls := 0
	err = SelectEach(
		ctx,
		db,
		statement,
		BoundSQL{SQL: "SELECT id FROM users ORDER BY id"},
		scanID,
		func(int64) error {
			calls++
			return yieldErr
		},
	)
	if err != yieldErr {
		t.Fatalf("yield error = %#v, want original error %#v", err, yieldErr)
	}
	if calls != 1 {
		t.Fatalf("yield calls = %d, want 1", calls)
	}
	if err := raw.PingContext(ctx); err != nil {
		t.Fatalf("connection was not released after yield error: %v", err)
	}

	var recovered any
	func() {
		defer func() {
			recovered = recover()
		}()
		_ = SelectEach(
			ctx,
			db,
			statement,
			BoundSQL{SQL: "SELECT id FROM users ORDER BY id"},
			scanID,
			func(int64) error {
				panic("yield panic")
			},
		)
	}()
	if recovered != "yield panic" {
		t.Fatalf("recovered panic = %#v, want %q", recovered, "yield panic")
	}
	if err := raw.PingContext(ctx); err != nil {
		t.Fatalf("connection was not released after yield panic: %v", err)
	}
}

func TestSelectEachWrapsScanAndRowsErrors(t *testing.T) {
	ctx := context.Background()
	statement := Statement{
		ID:     "UserMapper.SearchEach",
		Source: "user_mapper.xml:10",
		Kind:   StatementSelect,
	}

	raw, err := sql.Open("sqlite", "file:select_each_scan_error?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := raw.Close(); err != nil {
			t.Errorf("close SQLite: %v", err)
		}
	})
	db := NewDB(raw, DialectSQLite)
	err = SelectEach(
		ctx,
		db,
		statement,
		BoundSQL{SQL: "SELECT 'not-an-integer'"},
		func(row Scanner) (int64, error) {
			var value int64
			err := row.Scan(&value)
			return value, err
		},
		func(int64) error { return nil },
	)
	var statementError *StatementError
	if !errors.As(err, &statementError) {
		t.Fatalf("scan error = %T %v, want StatementError", err, err)
	}

	const driverName = "sqlmap_select_each_rows_error"
	registerSelectEachErrorDriver.Do(func() {
		sql.Register(driverName, selectEachErrorDriver{})
	})
	errorRaw, err := sql.Open(driverName, "")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := errorRaw.Close(); err != nil {
			t.Errorf("close error driver: %v", err)
		}
	})
	errorDB := NewDB(errorRaw, DialectSQLite)
	var values []int64
	err = SelectEach(
		ctx,
		errorDB,
		statement,
		BoundSQL{SQL: "SELECT value"},
		func(row Scanner) (int64, error) {
			var value int64
			err := row.Scan(&value)
			return value, err
		},
		func(value int64) error {
			values = append(values, value)
			return nil
		},
	)
	if !errors.Is(err, selectEachRowsError) {
		t.Fatalf("rows error = %v, want %v", err, selectEachRowsError)
	}
	if !errors.As(err, &statementError) {
		t.Fatalf("rows error = %T %v, want StatementError", err, err)
	}
	if len(values) != 1 || values[0] != 7 {
		t.Fatalf("values before rows error = %v, want [7]", values)
	}
}

func TestQueryOneStillReturnsErrNoRows(t *testing.T) {
	ctx := context.Background()
	raw, err := sql.Open("sqlite", "file:query_one_no_rows?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := raw.Close(); err != nil {
			t.Errorf("close SQLite: %v", err)
		}
	})
	raw.SetMaxOpenConns(1)

	db := NewDB(raw, DialectSQLite)
	statement := Statement{
		ID:     "UserMapper.Get",
		Source: "user_mapper.xml:20",
		Kind:   StatementSelect,
	}
	_, err = QueryOne(
		ctx,
		db,
		statement,
		BoundSQL{SQL: "SELECT 1 WHERE FALSE"},
		func(row Scanner) (int64, error) {
			var value int64
			err := row.Scan(&value)
			return value, err
		},
	)
	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("QueryOne() error = %v, want sql.ErrNoRows", err)
	}
}

func TestExecuteLastInsertIDSupportsMySQL(t *testing.T) {
	executor := &capabilityTestExecutor{
		dialect: DialectMySQL,
		result:  capabilityTestResult{lastID: 42},
	}
	statement := Statement{ID: "UserMapper.Create", Kind: StatementInsert}

	id, err := ExecuteLastInsertID(context.Background(), executor, statement, BoundSQL{})
	if err != nil {
		t.Fatal(err)
	}
	if !executor.execCalled {
		t.Fatal("Exec was not called")
	}
	if id != 42 {
		t.Fatalf("last insert ID = %d, want 42", id)
	}
}

func TestExecuteRejectsUnsupportedDialectBeforeExec(t *testing.T) {
	executor := &capabilityTestExecutor{dialect: DialectQuestion}
	statement := Statement{ID: "UserMapper.Delete", Kind: StatementDelete}

	err := Execute(context.Background(), executor, statement, BoundSQL{})
	if err == nil {
		t.Fatal("Execute() error = nil")
	}
	if executor.execCalled {
		t.Fatal("Exec was called before rejecting unsupported dialect")
	}

	var dialectError *UnsupportedDialectError
	if !errors.As(err, &dialectError) {
		t.Fatalf("error = %T %v, want UnsupportedDialectError", err, err)
	}
}

func TestSupportedDialectCapabilities(t *testing.T) {
	testCases := []struct {
		dialect   Dialect
		returning bool
		lastID    bool
		supported bool
	}{
		{dialect: DialectPostgres, returning: true, supported: true},
		{dialect: DialectSQLite, returning: true, lastID: true, supported: true},
		{dialect: DialectMySQL, lastID: true, supported: true},
		{dialect: DialectQuestion},
		{dialect: Dialect("oracle")},
	}

	for _, testCase := range testCases {
		t.Run(string(testCase.dialect), func(t *testing.T) {
			if got := testCase.dialect.Supports(FeatureDMLReturning); got != testCase.returning {
				t.Fatalf("Supports(FeatureDMLReturning) = %t, want %t", got, testCase.returning)
			}
			if got := testCase.dialect.Supports(FeatureLastInsertID); got != testCase.lastID {
				t.Fatalf("Supports(FeatureLastInsertID) = %t, want %t", got, testCase.lastID)
			}
			if got := testCase.dialect.Supported(); got != testCase.supported {
				t.Fatalf("Supported() = %t, want %t", got, testCase.supported)
			}
		})
	}
}
