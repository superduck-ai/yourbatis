package returningconformance

import "context"

//go:generate go run github.com/superduck-ai/yourbatis/cmd/sqlmapgen -mapper ReturningMapper -sql ./returning_mapper.xml -dialect sqlite

type ReturningMapper interface {
	CreateID(
		ctx context.Context,
		name string,
	) (int64, error)

	Create(
		ctx context.Context,
		name string,
		status string,
	) (CreatedUser, error)

	CreateMany(
		ctx context.Context,
		names []string,
	) ([]CreatedUser, error)

	CreateManyAsOne(
		ctx context.Context,
		names []string,
	) (CreatedUser, error)

	Rename(
		ctx context.Context,
		id int64,
		name string,
	) (CreatedUser, error)

	Disable(
		ctx context.Context,
		ids []int64,
	) ([]CreatedUser, error)

	Delete(
		ctx context.Context,
		id int64,
	) (CreatedUser, error)

	Get(
		ctx context.Context,
		id int64,
	) (CreatedUser, error)

	FindByStatus(
		ctx context.Context,
		status string,
	) (CreatedUser, bool, error)

	Count(
		ctx context.Context,
	) (int64, error)
}
