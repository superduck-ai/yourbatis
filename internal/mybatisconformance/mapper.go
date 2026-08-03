package mybatisconformance

import "context"

//go:generate go run github.com/superduck-ai/yourbatis/cmd/sqlmapgen -mapper ConformanceMapper -sql ./mapper.xml

type ConformanceBean struct {
	ID string
}

type ConformanceMapper interface {
	SelectDatabaseID(ctx context.Context) (string, error)
	SelectEscapedDollarLiteral(ctx context.Context) (string, error)
	SelectCustomTrimPreservesTrailingSpace(ctx context.Context, id int64) ([]int64, error)
	SelectWhitespaceSuffixOverride(ctx context.Context, id int64) ([]int64, error)
	SelectForeachRendering(ctx context.Context, ids []int64) ([]int64, error)
	SelectOtherwiseOnly(ctx context.Context, id int64) ([]int64, error)
	SelectEmptyChoose(ctx context.Context) (int64, error)
	InsertMap(ctx context.Context, values map[string]int64) error
	InsertForeachFirst(ctx context.Context, ids []int64) error
	InsertBound(ctx context.Context, values []int64) error
	InsertFiltered(ctx context.Context, ids []*int64) error
	InsertNested(ctx context.Context, groups [][]int64) error
	InsertStrictNames(
		ctx context.Context,
		uuu map[string]string,
		uuuu []ConformanceBean,
	) error
	InsertChosen(ctx context.Context, mode int, id int64) error
	DeleteAdjacent(ctx context.Context, left bool, right bool) error
	DeleteWhereSpacing(
		ctx context.Context,
		whereFirst bool,
		whereSecond bool,
		tailFirst bool,
		tailSecond bool,
	) error
	InsertTrimmed(ctx context.Context, id int64) error
	UpdateOptional(ctx context.Context, name *string, status *string) error
}
