package embeddedresultconformance

import "context"

//go:generate go run github.com/superduck-ai/yourbatis/cmd/sqlmapgen -mapper EmbeddedResultMapper -sql ./embedded_result_mapper.xml

type Base struct {
	ID int64
}

type User struct {
	Base
	Name string
}

type EmbeddedResultMapper interface {
	Get(ctx context.Context, id int64) (User, error)
}
