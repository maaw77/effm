package models

import "time"

// Subscription представляет запись о подписке пользователя.
// Поля соответствуют таблице subscriptions в базе данных PostgreSQL.
type Subscription struct {
	// ID подписки (UUID)
	ID string `db:"id" json:"id"`

	// Название сервиса, предоставляющего подписку
	ServiceName string `db:"service_name" json:"service_name"`

	// Стоимость месячной подписки в рублях (целое число)
	Price int `db:"price" json:"price"`

	// ID пользователя (UUID)
	UserID string `db:"user_id" json:"user_id"`

	// Дата начала подписки
	StartDate time.Time `db:"start_date" json:"start_date"`

	// Дата окончания подписки, может быть nil для активных подписок
	EndDate *time.Time `db:"end_date,omitempty" json:"end_date,omitempty"`

	// Дата и время создания записи
	CreatedAt time.Time `db:"created_at" json:"created_at"`

	// Дата и время последнего обновления записи
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`

	// Статус подписки, вычисляется в коде (не хранится в базе напрямую)
	Status string `db:"-" json:"status,omitempty"`

	// Дополнительные заметки по подписке (не обязательно)
	Notes *string `db:"-" json:"notes,omitempty"`
}
