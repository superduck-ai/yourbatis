# Yourbatis Project Guidance

## Database support contract

Database support is intentionally limited. Do not add abstractions or compatibility
code for databases outside this contract unless the user explicitly changes the
project scope.

### Support levels

- PostgreSQL is a primary, first-class target.
- SQLite is a primary, first-class target.
- MySQL is a secondary target for ordinary CRUD, transactions, dynamic SQL,
  result mapping, affected-row counts, and auto-increment IDs.
- Oracle, SQL Server, MariaDB, and all other databases are unsupported.
- Do not treat MariaDB as MySQL automatically; their DML return capabilities
  differ.

Primary targets require generated-code tests and real database integration
coverage for supported features. Secondary targets require generation and core
behavior coverage, but do not need feature parity with primary targets.

### DML result capabilities

| Capability | PostgreSQL | SQLite | MySQL |
| --- | --- | --- | --- |
| `result="returning"` | supported | supported | unsupported |
| INSERT/UPDATE/DELETE returning rows | supported | supported | unsupported |
| `result="lastid"` | unsupported | supported | supported |
| `result="rows"` | supported | supported | supported |
| `result="exec"` | supported | supported | supported |
| hidden `selectKey` BEFORE/AFTER execution | unsupported | unsupported | unsupported |
| reflection-based `keyProperty` mutation | unsupported | unsupported | unsupported |

`result="returning"` means that a DML statement produces a result set and must
use the query-and-scan path. PostgreSQL and SQLite SQL normally express this
with a `RETURNING` clause. Support single-column scalars, multiple mapped
columns, one returned row, and multiple returned rows.

`result="lastid"` means `ExecContext` followed by `sql.Result.LastInsertId`.
It is only valid when the selected dialect and driver provide that capability.
Do not emulate it with an extra query.

When the database does not provide a generated value, users must either supply
the ID in Go, generate it directly in their SQL, or use a separate explicit
Mapper method and transaction. The framework must not manufacture portability
through hidden multi-statement execution.

### Go-first constraints

- Do not implement MyBatis-style `selectKey` orchestration in the generated
  Mapper method.
- Do not use reflection to write generated keys into arbitrary parameter
  properties.
- Do not introduce an implicit `Atomic` transaction solely to emulate
  `selectKey`.
- Do not automatically fall back between `RETURNING`, `LastInsertId`, and
  additional key queries. They have different SQL, driver, and transaction
  semantics.
- Keep the result mode explicit in XML: `returning`, `lastid`, `rows`, or
  `exec`.
- Prefer statically generated scanners and ordinary Go return values over
  parameter mutation.

### Dialects and capability checks

Production code must distinguish these dialects explicitly:

```go
const (
	DialectPostgres Dialect = "postgres"
	DialectSQLite   Dialect = "sqlite"
	DialectMySQL    Dialect = "mysql"
)
```

PostgreSQL uses `$n` placeholders. SQLite and MySQL use `?` placeholders.
`DialectQuestion` may remain temporarily for low-level builder/conformance
tests, but it must not represent a production database because SQLite and
MySQL have different capabilities.

Validate dialect-level capabilities before executing a statement with side
effects:

- Reject `result="lastid"` on PostgreSQL before running the INSERT. Executing
  first and failing at `LastInsertId` could report an error after data was
  already written.
- Reject DML query/scan execution on MySQL before running a statement declared
  with `result="returning"`.
- Treat the real database and driver as the final authority for server-version
  or driver-specific limitations. Do not probe capabilities using mutating SQL.

When `sqlmapgen` is given a target dialect, unsupported result modes must fail
during generation. When no target dialect is supplied, generated code must
perform the capability check at runtime before execution.

One Mapper XML file should normally target one database dialect. `_databaseId`
may handle small SQL syntax variations, but the generator must not silently
switch a Mapper method between query-returning and `LastInsertId` execution.

### Required test coverage

PostgreSQL and SQLite coverage for `result="returning"` must include:

- an independent XML fixture;
- `go generate`;
- assertions against generated Go code;
- scalar and struct results;
- single-row and multi-row results;
- INSERT, UPDATE, and DELETE returning rows;
- result-map and automatic result mapping;
- zero-row and too-many-row behavior where applicable;
- successful commit and rollback in a real database transaction;
- logging and statement-error wrapping.

SQLite integration tests may use the repository's selected
`modernc.org/sqlite` driver. PostgreSQL integration tests should use a
`database/sql`-compatible PostgreSQL driver.

MySQL coverage must include SQL generation, `lastid`, `rows`, `exec`, basic
CRUD, transactions, dynamic SQL, and result mapping. MySQL does not need
`result="returning"` coverage.

Do not add compatibility tests or implementation work for unsupported
databases.

### Definition of done for subsequent features

Every subsequent framework feature or behavior change must have real database
integration coverage on both SQLite and PostgreSQL before it is considered
complete. Unit tests, generated-code assertions, or SQLite-only coverage are
not sufficient substitutes for this requirement.

- Run the same supported behavior against SQLite and PostgreSQL wherever their
  SQL syntax and capabilities overlap.
- When syntax differs, use dialect-specific fixtures while asserting equivalent
  framework semantics.
- Exercise generated Mapper code in both database tests; do not validate only
  hand-written runtime calls.
- PostgreSQL tests must actually run during development and verification. An
  environment-gated PostgreSQL test that is merely compiled and skipped does
  not satisfy the completion requirement.
- Use isolated temporary tables or schemas and transactions so tests do not
  modify persistent application data.
- If PostgreSQL and SQLite intentionally differ in capability, add an explicit
  positive test for the supported dialect and an explicit rejection test for
  the unsupported dialect, then document the difference.
