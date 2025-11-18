// internal/dto/dto.go
// Пакет dto содержит структуры для приёма и отдачи данных через HTTP API.
// Все даты теперь в формате YYYY-MM (например, "2025-07") — это и удобно, и однозначно.

package dto

// CreateSubscriptionRequest — тело запроса на создание подписки за конкретный месяц
type CreateSubscriptionRequest struct {
	ServiceName string `json:"service_name" binding:"required" example:"Yandex Plus"`
	Price       int    `json:"price" binding:"required,gt=0" example:"499"` // gt=0 вместо min=1
	UserID      string `json:"user_id" binding:"required,uuid" example:"f47ac10b-58cc-4372-a567-0e02b2c3d479"`
	// Формат: YYYY-MM (например, 2025-07)
	YearMonth string `json:"year_month" binding:"required,len=7" example:"2025-07"`
}

// UpdateSubscriptionRequest — частичное обновление подписки (только цена и название)
type UpdateSubscriptionRequest struct {
	ServiceName *string `json:"service_name,omitempty" example:"Yandex Plus Premium"`
	Price       *int    `json:"price,omitempty" binding:"omitempty,gt=0" example:"599"` // gt=0 + omitempty
}

// SubscriptionResponse — ответ сервера при чтении одной или списка подписок
type SubscriptionResponse struct {
	ID          string `json:"id" example:"f47ac10b-58cc-4372-a567-0e02b2c3d479"`
	ServiceName string `json:"service_name" example:"Yandex Plus"`
	Price       int    `json:"price" example:"499"`
	UserID      string `json:"user_id" example:"f47ac10b-58cc-4372-a567-0e02b2c3d479"`
	YearMonth   string `json:"year_month" example:"2025-07"` // YYYY-MM
	CreatedAt   string `json:"created_at" example:"2025-07-15T10:23:45Z"`
	UpdatedAt   string `json:"updated_at" example:"2025-07-15T10:23:45Z"`
}

// TotalCostRequest — параметры для подсчёта общей стоимости за период
type TotalCostRequest struct {
	Start   string  `form:"start" binding:"required,len=7" example:"2025-01"` // YYYY-MM
	End     string  `form:"end" binding:"required,len=7" example:"2025-12"`   // YYYY-MM
	UserID  *string `form:"user_id,omitempty" binding:"omitempty,uuid" example:"f47ac10b-58cc-4372-a567-0e02b2c3d479"`
	Service *string `form:"service,omitempty" example:"Yandex Plus"`
}

// TotalCostResponse — результат подсчёта
type TotalCostResponse struct {
	Total int `json:"total" example:"5999"` // сумма в рублях
}
