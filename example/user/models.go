package user

import "time"

type UserStatus string

const (
	UserStatusActive   UserStatus = "active"
	UserStatusDisabled UserStatus = "disabled"
	UserStatusDeleted  UserStatus = "deleted"
)

type User struct {
	ID        int64      `db:"id"`
	Name      string     `db:"name"`
	Email     string     `db:"email"`
	Status    UserStatus `db:"status"`
	CreatedAt time.Time  `db:"created_at"`
}

type UserLabel struct {
	UserID      int64  `db:"user_id"`
	DisplayName string `db:"display_name"`
}

type SearchUserParams struct {
	Keyword        string
	Status         *UserStatus
	IDs            []int64
	IncludeDeleted bool
	Limit          int
	Offset         int
}

func filterPositiveUserIDs(ids []int64) []int64 {
	filtered := make([]int64, 0, len(ids))
	for _, id := range ids {
		if id > 0 {
			filtered = append(filtered, id)
		}
	}
	return filtered
}

type CreateUserParams struct {
	Name   string
	Email  string
	Status UserStatus
}
