package dto

// CreateSubscriptionRequest используется для создания новой подписки.
// Все поля обязательны, кроме EndDate и Notes. Даты задаются в формате "MM-YYYY".
type CreateSubscriptionRequest struct {
	ServiceName string  `json:"service_name" binding:"required" example:"Yandex Plus"`
	Price       int     `json:"price" binding:"required,min=0" example:"400"`
	UserID      string  `json:"user_id" binding:"required,uuid" example:"60601fee-2bf1-4721-ae6f-7636e79a0cba"`
	StartDate   string  `json:"start_date" binding:"required" example:"07-2025"`
	EndDate     *string `json:"end_date,omitempty" example:"12-2025"`
	Notes       *string `json:"notes,omitempty" example:"Пробная подписка"`
}

// UpdateSubscriptionRequest используется для обновления существующей подписки.
// Все поля опциональны. Nil означает отсутствие изменения.
type UpdateSubscriptionRequest struct {
	ServiceName *string `json:"service_name,omitempty" example:"Yandex Plus Premium"`
	Price       *int    `json:"price,omitempty" example:"500"`
	StartDate   *string `json:"start_date,omitempty" example:"08-2025"`
	EndDate     *string `json:"end_date,omitempty" example:"12-2025"`
	Notes       *string `json:"notes,omitempty" example:"С продлением на год"`
}

// SubscriptionResponse представляет данные подписки,
// возвращаемые клиенту API.
type SubscriptionResponse struct {
	ID          string  `json:"id" example:"60601fee-2bf1-4721-ae6f-7636e79a0cba"`
	ServiceName string  `json:"service_name" example:"Yandex Plus"`
	Price       int     `json:"price" example:"400"`
	UserID      string  `json:"user_id" example:"60601fee-2bf1-4721-ae6f-7636e79a0cba"`
	StartDate   string  `json:"start_date" example:"07-2025"`
	EndDate     *string `json:"end_date,omitempty" example:"12-2025"`
	CreatedAt   string  `json:"created_at" example:"2025-07-01T12:00:00Z"`
	UpdatedAt   string  `json:"updated_at" example:"2025-07-01T12:00:00Z"`
	Status      string  `json:"status" example:"active"`
	Notes       *string `json:"notes,omitempty" example:"Пробная подписка"`
}

// TotalCostRequest используется для получения суммарной стоимости подписок
// за указанный период. Даты — обязательны. Фильтры — опциональны.
type TotalCostRequest struct {
	Start       string  `form:"start" binding:"required" example:"01-2025"`
	End         string  `form:"end" binding:"required" example:"12-2025"`
	UserID      *string `form:"user_id,omitempty" example:"60601fee-2bf1-4721-ae6f-7636e79a0cba"`
	ServiceName *string `form:"service,omitempty" example:"Yandex Plus"`
}

// TotalCostResponse представляет результат расчёта суммарной стоимости.
type TotalCostResponse struct {
	Total int `json:"total" example:"3200"`
}
