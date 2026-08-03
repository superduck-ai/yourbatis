package mybatiscoreconformance

import "context"

type InferredUser struct {
	ID   int64
	Name string
}

type CamelCaseRow struct {
	ID        int64
	FirstName string
}

type QuotedAliasRow struct {
	ID        int64  `db:"id"`
	FirstName string `db:"firstName"`
	LastName  string `db:"lastName"`
}

type BlobRow struct {
	ID      int64  `db:"id"`
	Payload []byte `db:"payload"`
}

type NullableRow struct {
	First  *string `db:"first_value"`
	Second *string `db:"second_value"`
}

type NestedProperty struct {
	ID    int64
	Value string
}

type NestedItem struct {
	ID         int64
	Properties []NestedProperty
}

// Each positive mapper corresponds to one independently reviewed upstream
// test method. Where Java-specific APIs do not exist, the XML preserves only
// the observable contract in a Go-first form.

//go:generate go run github.com/superduck-ai/yourbatis/cmd/sqlmapgen -mapper NoResultTypeOneMapper -sql ./fixtures/no_result_type_one.xml
type NoResultTypeOneMapper interface {
	Run(ctx context.Context, id int64) (InferredUser, error)
}

//go:generate go run github.com/superduck-ai/yourbatis/cmd/sqlmapgen -mapper NoResultTypeManyMapper -sql ./fixtures/no_result_type_many.xml
type NoResultTypeManyMapper interface {
	Run(ctx context.Context) ([]InferredUser, error)
}

//go:generate go run github.com/superduck-ai/yourbatis/cmd/sqlmapgen -mapper ForeachItemScopeMapper -sql ./fixtures/foreach_item_scope.xml
type ForeachItemScopeMapper interface {
	Run(ctx context.Context, id int64, ids []int64, ids2 []int64) (int64, error)
}

//go:generate go run github.com/superduck-ai/yourbatis/cmd/sqlmapgen -mapper ForeachIndexScopeMapper -sql ./fixtures/foreach_index_scope.xml
type ForeachIndexScopeMapper interface {
	Run(ctx context.Context, idx int64, idxs []int64, idxs2 []int64) (int64, error)
}

//go:generate go run github.com/superduck-ai/yourbatis/cmd/sqlmapgen -mapper CamelCaseMapper -sql ./fixtures/camel_case.xml
type CamelCaseMapper interface {
	Run(ctx context.Context) ([]CamelCaseRow, error)
}

//go:generate go run github.com/superduck-ai/yourbatis/cmd/sqlmapgen -mapper QuotedAliasMapper -sql ./fixtures/quoted_alias.xml
type QuotedAliasMapper interface {
	Run(ctx context.Context) ([]QuotedAliasRow, error)
}

// Go deliberately treats []byte as one scalar BLOB value, not as a collection
// result. The second method makes the multi-row cardinality contract explicit.
//
//go:generate go run github.com/superduck-ai/yourbatis/cmd/sqlmapgen -mapper ByteScalarMapper -sql ./fixtures/byte_scalar.xml
type ByteScalarMapper interface {
	One(ctx context.Context, id int64) ([]byte, error)
	Many(ctx context.Context) ([]byte, error)
}

//go:generate go run github.com/superduck-ai/yourbatis/cmd/sqlmapgen -mapper BlobMapper -sql ./fixtures/blob.xml
type BlobMapper interface {
	Insert(ctx context.Context, row BlobRow) (int64, error)
	SelectAll(ctx context.Context) ([]BlobRow, error)
}

//go:generate go run github.com/superduck-ai/yourbatis/cmd/sqlmapgen -mapper NullScalarMapper -sql ./fixtures/null_scalar.xml
type NullScalarMapper interface {
	Run(ctx context.Context, id int64) (*string, error)
}

//go:generate go run github.com/superduck-ai/yourbatis/cmd/sqlmapgen -mapper NullStructMapper -sql ./fixtures/null_struct.xml
type NullStructMapper interface {
	Run(ctx context.Context, id int64) (NullableRow, error)
}

//go:generate go run github.com/superduck-ai/yourbatis/cmd/sqlmapgen -mapper NestedValuesMapper -sql ./fixtures/nested_values.xml
type NestedValuesMapper interface {
	Insert(ctx context.Context, item NestedItem) (int64, error)
	Count(ctx context.Context, itemID int64) (int64, error)
}

//go:generate go run github.com/superduck-ai/yourbatis/cmd/sqlmapgen -mapper UTF8LiteralMapper -sql ./fixtures/utf8_literal.xml
type UTF8LiteralMapper interface {
	Run(ctx context.Context) (string, error)
}

//go:generate go run github.com/superduck-ai/yourbatis/cmd/sqlmapgen -mapper ControlCharacterMapper -sql ./fixtures/control_characters.xml
type ControlCharacterMapper interface {
	Insert(ctx context.Context, id int64) (int64, error)
	Get(ctx context.Context, id int64) (string, error)
}

//go:generate go run github.com/superduck-ai/yourbatis/cmd/sqlmapgen -mapper CanonicalIncludeMapper -sql ./fixtures/canonical_include.xml
type CanonicalIncludeMapper interface {
	Bare(ctx context.Context, id int64) (InferredUser, error)
	Qualified(ctx context.Context, id int64) (InferredUser, error)
}

//go:generate go run github.com/superduck-ai/yourbatis/cmd/sqlmapgen -mapper CanonicalResultMapMapper -sql ./fixtures/canonical_result_map.xml
type CanonicalResultMapMapper interface {
	FromBareDefinition(ctx context.Context, id int64) (InferredUser, error)
	FromQualifiedDefinition(ctx context.Context, id int64) (InferredUser, error)
}

// The interfaces below are generation-failure fixtures and deliberately have
// no go:generate directives.

type NegativeRow struct {
	ID int64
}

type WrongNamespaceMapper interface {
	Run(ctx context.Context) (int64, error)
}

type MissingNamespaceMapper interface {
	Run(ctx context.Context) (int64, error)
}

type MissingResultMapMapper interface {
	Run(ctx context.Context) (NegativeRow, error)
}

type ExtraStatementMapper interface {
	Run(ctx context.Context) (int64, error)
}

type MissingStatementMapper interface {
	Run(ctx context.Context) (int64, error)
	Unmapped(ctx context.Context) (int64, error)
}

type MissingIncludeMapper interface {
	Run(ctx context.Context) (int64, error)
}

type EmptyResultMapMapper interface {
	Run(ctx context.Context) (NegativeRow, error)
}

type DuplicateResultMapMapper interface {
	Run(ctx context.Context) (NegativeRow, error)
}

type DuplicateFragmentQualifiedThenBareMapper interface {
	Run(ctx context.Context) (NegativeRow, error)
}

type DuplicateFragmentBareThenQualifiedMapper interface {
	Run(ctx context.Context) (NegativeRow, error)
}

type DuplicateResultMapQualifiedThenBareMapper interface {
	Run(ctx context.Context) (NegativeRow, error)
}

type DuplicateResultMapBareThenQualifiedMapper interface {
	Run(ctx context.Context) (NegativeRow, error)
}

type ForeignResultMapDefinitionMapper interface {
	Run(ctx context.Context) (NegativeRow, error)
}

type ForeignResultMapReferenceMapper interface {
	Run(ctx context.Context) (NegativeRow, error)
}
