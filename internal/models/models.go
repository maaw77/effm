package models

import "time"

type Subscription struct {
	ID          string     `db:"id" json:"id"` // UUID
	ServiceName string     `db:"service_name" json:"service_name"`
	Price       int        `db:"price" json:"price"`     // целые рубли
	UserID      string     `db:"user_id" json:"user_id"` // UUID пользователя
	StartDate   time.Time  `db:"start_date" json:"start_date"`
	EndDate     *time.Time `db:"end_date,omitempty" json:"end_date,omitempty"`
	CreatedAt   time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time  `db:"updated_at" json:"updated_at"`
}
