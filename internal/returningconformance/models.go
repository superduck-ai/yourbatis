package returningconformance

import "time"

type CreatedUser struct {
	ID        int64     `db:"id"`
	Name      string    `db:"name"`
	Status    string    `db:"status"`
	CreatedAt time.Time `db:"created_at"`
}
