package projectionconformance

import "context"

//go:generate go run github.com/superduck-ai/yourbatis/cmd/sqlmapgen -mapper ProjectionMapper -sql ./projection_mapper.xml -dialect sqlite

type ProjectionMapper interface {
	Get(ctx context.Context, id int64) (OrderProjection, error)
	Exists(ctx context.Context, id int64) (bool, error)
}
