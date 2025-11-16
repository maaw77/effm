package models

import "time"

// Subscription представляет запись о подписке пользователя.
// Поля полностью соответствуют таблице subscriptions в базе данных PostgreSQL.
type Subscription struct {
	// ID подписки (UUID, PRIMARY KEY)
	ID string `db:"id" json:"id"`

	// Название сервиса, предоставляющего подписку
	ServiceName string `db:"service_name" json:"service_name"`

	// Стоимость месячной подписки в рублях (целое число, >= 0)
	Price int `db:"price" json:"price"`

	// ID пользователя (UUID)
	UserID string `db:"user_id" json:"user_id"`

	// Дата начала подписки
	StartDate time.Time `db:"start_date" json:"start_date"`

	// Дата окончания подписки (nullable)
	EndDate *time.Time `db:"end_date,omitempty" json:"end_date,omitempty"`

	// Дата и время создания записи (по умолчанию NOW())
	CreatedAt time.Time `db:"created_at" json:"created_at"`

	// Дата и время последнего обновления записи (по умолчанию NOW())
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}
