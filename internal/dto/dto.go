// Пакет dto содержит структуры для приёма и отдачи данных через HTTP API.
// Все даты теперь в формате YYYY-MM (например, "2025-07") — это и удобно, и однозначно.

package dto

// CreateSubscriptionRequest — тело запроса на создание подписки за конкретный месяц
// @Description Данные для создания записи об оплате подписки. Дата в формате YYYY-MM (например, 2025-07)
//
//	@Example {
//	  "service_name": "Yandex Plus",
//	  "price": 499,
//	  "user_id": "f47ac10b-58cc-4372-a567-0e02b2c3d479",
//	  "year_month": "2025-07"
//	}
type CreateSubscriptionRequest struct {
	ServiceName string `json:"service_name" binding:"required" example:"Yandex Plus" validate:"required"`                               // Название сервиса
	Price       int    `json:"price" binding:"required,gt=0" example:"499" validate:"gt=0"`                                             // Стоимость за месяц в рублях
	UserID      string `json:"user_id" binding:"required,uuid" example:"f47ac10b-58cc-4372-a567-0e02b2c3d479" validate:"required,uuid"` // UUID пользователя
	YearMonth   string `json:"year_month" binding:"required,len=7" example:"2025-07" validate:"required"`                               // Год и месяц в формате YYYY-MM
}

// UpdateSubscriptionRequest — частичное обновление подписки (только то, что можно менять)
// @Description Можно обновить название сервиса и/или цену. Поля опциональные
type UpdateSubscriptionRequest struct {
	ServiceName *string `json:"service_name,omitempty" example:"Yandex Plus Premium"`   // Новое название сервиса
	Price       *int    `json:"price,omitempty" binding:"omitempty,gt=0" example:"599"` // Новая цена (если 0 — будет проигнорировано)
}

// SubscriptionResponse — ответ при чтении одной подписки или списка
// @Description Полная информация о записи подписки с датой создания/обновления
type SubscriptionResponse struct {
	ID          string `json:"id" example:"f47ac10b-58cc-4372-a567-0e02b2c3d479"`      // UUID записи
	ServiceName string `json:"service_name" example:"Yandex Plus"`                     // Название сервиса
	Price       int    `json:"price" example:"499"`                                    // Стоимость за месяц
	UserID      string `json:"user_id" example:"f47ac10b-58cc-4372-a567-0e02b2c3d479"` // UUID пользователя
	YearMonth   string `json:"year_month" example:"2025-07"`                           // Год и месяц в формате YYYY-MM
	CreatedAt   string `json:"created_at" example:"2025-07-15T10:23:45Z"`              // Время создания
	UpdatedAt   string `json:"updated_at" example:"2025-07-15T10:23:45Z"`              // Время последнего обновления
}

// TotalCostRequest — параметры для подсчёта общей стоимости
type TotalCostRequest struct {
	Start   string  `form:"start" binding:"required,len=7" example:"2025-01"`                                          // Начало периода YYYY-MM
	End     string  `form:"end" binding:"required,len=7" example:"2025-12"`                                            // Конец периода YYYY-MM
	UserID  *string `form:"user_id,omitempty" binding:"omitempty,uuid" example:"f47ac10b-58cc-4372-a567-0e02b2c3d479"` // Опционально: фильтр по пользователю
	Service *string `form:"service,omitempty" example:"Yandex Plus"`                                                   // Опционально: фильтр по сервису
}

// TotalCostResponse — результат подсчёта
// @Description Суммарная стоимость всех подходящих подписок за выбранный период
type TotalCostResponse struct {
	Total int `json:"total" example:"8599"` // Итоговая сумма в рублях
}
