// Пакет models содержит структуры данных для работы с подписками.
// Структура Subscription полностью соответствует таблице subscriptions в PostgreSQL.
// Одна запись = оплата за конкретный месяц (модель "monthly billing").

package models

import "time"

// Subscription представляет запись о подписке пользователя за конкретный месяц.
// Поля соответствуют таблице subscriptions в БД.
type Subscription struct {
	// ID подписки (UUID, PRIMARY KEY)
	ID string `db:"id" json:"id"`

	// Название сервиса (например, "Yandex Plus")
	ServiceName string `db:"service_name" json:"service_name"`

	// Стоимость за месяц в рублях (целое число, >= 0)
	Price int `db:"price" json:"price"`

	// ID пользователя (UUID)
	UserID string `db:"user_id" json:"user_id"`

	// Год подписки (2000-2100)
	Year int `db:"year" json:"year"`

	// Месяц подписки (1-12)
	Month int `db:"month" json:"month"`

	// Дата и время создания записи
	CreatedAt time.Time `db:"created_at" json:"created_at"`

	// Дата и время последнего обновления
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}
