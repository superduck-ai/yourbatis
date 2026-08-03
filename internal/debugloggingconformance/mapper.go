package debugloggingconformance

import (
	"context"
	"database/sql/driver"
)

//go:generate go run github.com/superduck-ai/yourbatis/cmd/sqlmapgen -mapper DebugLoggingMapper -sql ./debug_logging_mapper.xml

type DebugLoggingMapper interface {
	EchoUUID(ctx context.Context, id UUID) (string, error)
}

type UUID string

func (value UUID) String() string {
	return string(value)
}

func (value UUID) Value() (driver.Value, error) {
	return string(value), nil
}
