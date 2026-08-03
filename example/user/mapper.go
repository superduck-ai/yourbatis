package user

import (
	"context"

	"github.com/superduck-ai/yourbatis"
)

//go:generate go run github.com/superduck-ai/yourbatis/cmd/sqlmapgen -mapper UserMapper -sql ./user_mapper.xml -dialect sqlite

type UserMapper interface {
	GetByID(
		ctx context.Context,
		id int64,
	) (User, error)

	FindByEmail(
		ctx context.Context,
		email string,
	) (User, bool, error)

	GetLabel(
		ctx context.Context,
		id int64,
	) (UserLabel, error)

	Search(
		ctx context.Context,
		params SearchUserParams,
	) ([]User, error)

	SearchEach(
		ctx context.Context,
		params SearchUserParams,
		yield func(User) error,
	) error

	CountByStatus(
		ctx context.Context,
		status UserStatus,
	) (int64, error)

	ListNamesOrderedBy(
		ctx context.Context,
		column yourbatis.Identifier,
	) ([]string, error)

	Create(
		ctx context.Context,
		params CreateUserParams,
	) (int64, error)

	CreateMany(
		ctx context.Context,
		params []CreateUserParams,
	) (int64, error)

	UpdateStatus(
		ctx context.Context,
		id int64,
		status UserStatus,
	) (int64, error)

	Delete(
		ctx context.Context,
		id int64,
	) error
}
