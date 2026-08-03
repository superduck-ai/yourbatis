# MyBatis test review: remaining core migrations

This document records the final migration set selected after reviewing every
`*Test.java` file under `/Users/arthur/IdeaProjects/mybatis-3/src/test/java`.

## Audit scope and decision rule

- Upstream base revision: `2fa2b38690af8f93e32b40e54f8611707b9d71f1`;
  the current test worktree was reviewed, whose local deltas do not change the
  test contracts selected here.
- Inventory: 367 `*Test.java` files. Of these, 361 directly declare 1,916
  JUnit `@Test` or `@ParameterizedTest` methods; the other six are shared base
  classes or subclasses that inherit tests.
- Existing migrations remain documented in:
  - `docs/mybatis-java-test-migration.md`
  - `docs/mybatis-dynamic-sql-test-migration.md`
  - `docs/mybatis-binding-test-migration.md`
- A method was selected only when its observable contract exists in the
  Go-first framework and was not already covered at equivalent strength.
- Java runtime configuration, annotations/providers, reflection, proxy/lazy
  loading, caches, constructor/discriminator/nested result graphs, custom type
  handlers, stored procedures, batch executors, `selectKey`, `keyProperty`, and
  unsupported databases were not copied. Those are outside the current product
  contract, not missing test coverage for supported behavior.
- CRUD, transactions, DML modes, streaming, cardinality, result mapping, dynamic
  SQL, and parameter binding methods already covered by stronger generated-code
  and real-database tests were not duplicated.

The result is 16 source files and 21 source methods that required new
conformance coverage.

## Required file and function list

| Priority | Upstream test file | Upstream function | Go-first observable contract | Migrated coverage |
| --- | --- | --- | --- | --- |
| P0 | `binding/WrongNamespacesTest.java` | `shouldFailForWrongNamespace` | XML namespace must match the Go mapper interface | `TestMigratedGenerationFailures` |
| P0 | `binding/WrongNamespacesTest.java` | `shouldFailForMissingNamespace` | XML mapper requires a non-empty namespace | `TestMigratedGenerationFailures` |
| P0 | `builder/XmlMapperBuilderTest.java` | `errorResultMapLocation` | A missing `resultMap` reference fails generation with the XML source location | `TestMigratedGenerationFailures` |
| P0 | `submitted/no_result_type_map/NoResultTypeMapTest.java` | `shouldGetAUser` | A single Go return type supplies the scan type when XML omits `resultType` | SQLite + PostgreSQL |
| P0 | `submitted/no_result_type_map/NoResultTypeMapTest.java` | `shouldGetAllUsers` | A slice return supplies element type and many-row cardinality when XML omits `resultType` | SQLite + PostgreSQL |
| P0 | `submitted/no_result_type_map/NoResultTypeMapTest.java` | `shouldFailIfNoMatchingMethod` | An XML statement without a Go mapper method fails generation | `TestMigratedGenerationFailures` |
| P0 | `submitted/mapper_extend/MapperExtendTest.java` | `shouldThrowExceptionIfNoMatchingStatementFound` | A Go mapper method without an XML statement fails generation | `TestMigratedGenerationFailures` |
| P0 | `submitted/refid_resolution/RefidResolutionTest.java` | `includes` | An unresolved, cross-namespace `refid` fails immediately; no cross-file fallback is invented | `TestMigratedGenerationFailures` |
| P0 | `submitted/foreach/ForEachTest.java` | `shouldRemoveItemVariableInTheContext` | A nested loop that shadows `item` restores the outer/root binding afterward | Exact `BoundSQL` + SQLite + PostgreSQL |
| P0 | `submitted/foreach/ForEachTest.java` | `shouldRemoveIndexVariableInTheContext` | A nested loop that shadows `index` restores the outer/root binding afterward | Exact `BoundSQL` + SQLite + PostgreSQL |
| P0 | `builder/annotation/NamedResultMapTest.java` | `namedResultMapWithNoMappingsShouldFail` | An empty XML `resultMap` fails generation | `TestMigratedGenerationFailures` |
| P0 | `submitted/results_id/IdConflictTest.java` | `shouldFailOnDuplicatedId` | Duplicate XML `resultMap` IDs fail generation | `TestMigratedGenerationFailures` |
| P1 | `submitted/camelcase/CamelCaseMappingTest.java` | `list` | An untagged Go field such as `FirstName` maps from an explicit `first_name` projection | SQLite + PostgreSQL |
| P1 | `submitted/quotedcolumnnames/QuotedColumnNamesTest.java` | `it` | Quoted aliases retain their labels and map to statically declared Go fields | SQLite + PostgreSQL |
| P1 | `submitted/array_result_type/ArrayResultTypeTest.java` | `ensureByteArrayBackwardCompatibility` | Go `[]byte` is one BLOB scalar; a multi-row query returns `ErrTooManyRows`, not an aggregated byte collection | SQLite + PostgreSQL |
| P1 | `submitted/blobtest/BlobTest.java` | `insertBlobThenSelectAll` | `[]byte` parameters and fields round-trip without a custom constructor/type handler | SQLite + PostgreSQL |
| P1 | `submitted/empty_row/ReturnInstanceForEmptyRowTest.java` | `shouldSimpleTypeBeNull` | SQL `NULL` scans into a nil pointer scalar | SQLite + PostgreSQL |
| P1 | `submitted/empty_row/ReturnInstanceForEmptyRowTest.java` | `shouldObjectTypeNotBeNull` | An existing all-NULL row returns a struct whose nullable fields are nil | SQLite + PostgreSQL |
| P1 | `submitted/complex_type/ComplexTypeTest.java` | `shouldUpdateProps` | Nested slice elements bind together with an outer struct field in a multi-values DML statement | SQLite + PostgreSQL |
| P1 | `submitted/encoding/EncodingTest.java` | `encoding2` | UTF-8 mapper text survives parse, generation, execution, and scan exactly | SQLite + PostgreSQL |
| P1 | `submitted/literal_string/LiteralStringTest.java` | `shouldRetainControlCharactersInMapperXml` | XML numeric character references survive end to end exactly | SQLite + PostgreSQL |

## Migration layout

- Go mapper contracts: `internal/mybatiscoreconformance/mapper.go`
- Independent positive XML fixtures:
  `internal/mybatiscoreconformance/fixtures/`
- Independent failure fixtures:
  `internal/mybatiscoreconformance/negative/`
- Generation-failure assertions:
  `internal/mybatiscoreconformance/generation_contract_test.go`
- Exact foreach `BoundSQL` assertions:
  `internal/mybatiscoreconformance/mapper_generated_test.go`
- Shared behavior assertions:
  `internal/mybatiscoreconformance/integration_test.go`
- Real database runners:
  `internal/mybatiscoreconformance/sqlite_integration_test.go` and
  `internal/mybatiscoreconformance/postgres_integration_test.go`

## Implementation gap found by the migration

The migrated `shouldSimpleTypeBeNull` test found that generated pointer-scalar
scanners allocated the pointed-to value before `Scan`. Both PostgreSQL and
SQLite then attempted to convert SQL `NULL` into a concrete `string` and
failed. The generator now scans into `**T` for scalar `*T` returns, allowing
`database/sql` to preserve `NULL` as a nil pointer. Pointer-to-struct scanners
remain preallocated because their individual fields are scan destinations.

This behavior is guarded by both the generator unit test
`TestGenerateScansPointerScalarWithoutPreallocation` and the two real-database
integration suites.
