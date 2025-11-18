// Пакет server реализует HTTP-слой (REST API) приложения.
// Используется фреймворк Gin. Все ручки работают с новой моделью данных:
// одна запись = оплата подписки за конкретный месяц (year + month).

package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/maaw77/effm/internal/database"
	"github.com/maaw77/effm/internal/dto"
	"github.com/maaw77/effm/internal/models"
)

// Server — основной объект HTTP-сервера.
// Содержит роутер Gin и подключение к базе данных.
type Server struct {
	Router *gin.Engine
	DB     *database.SubscriptionDatabase
}

// NewServer создаёт и настраивает новый экземпляр сервера.
// Регистрирует все HTTP-ручки.
func NewServer(db *database.SubscriptionDatabase) *Server {
	s := &Server{
		Router: gin.Default(),
		DB:     db,
	}

	// Группа API с префиксом /api
	api := s.Router.Group("/api")
	{
		// CRUD операции
		api.POST("/subscriptions", s.createSubscriptionHandler)
		api.GET("/subscriptions", s.listSubscriptionsHandler)
		api.GET("/subscriptions/:id", s.getSubscriptionHandler)
		api.PUT("/subscriptions/:id", s.updateSubscriptionHandler)
		api.DELETE("/subscriptions/:id", s.deleteSubscriptionHandler)

		// Подсчёт общей стоимости
		api.GET("/subscriptions/total", s.sumSubscriptionsHandler)
	}

	return s
}

// parseYearMonth преобразует строку "2025-07" в год и месяц.
// Возвращает ошибку, если формат неправильный или значения вне диапазона.
func parseYearMonth(s string) (int, int, error) {
	var year, month int
	n, err := fmt.Sscanf(s, "%d-%02d", &year, &month)
	if err != nil || n != 2 {
		return 0, 0, fmt.Errorf("неверный формат даты: %s (ожидается YYYY-MM)", s)
	}
	if month < 1 || month > 12 {
		return 0, 0, fmt.Errorf("месяц должен быть от 01 до 12, получено: %02d", month)
	}
	if year < 2000 || year > 2100 {
		return 0, 0, fmt.Errorf("год должен быть от 2000 до 2100, получено: %d", year)
	}
	return year, month, nil
}

// createSubscriptionHandler — POST /api/subscriptions
// Создаёт новую запись о подписке за указанный месяц.
func (s *Server) createSubscriptionHandler(c *gin.Context) {
	var req dto.CreateSubscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "некорректное тело запроса", "details": err.Error()})
		return
	}

	year, month, err := parseYearMonth(req.YearMonth)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	sub := models.Subscription{
		ServiceName: req.ServiceName,
		Price:       req.Price,
		UserID:      req.UserID,
		Year:        year,
		Month:       month,
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	id, err := s.DB.Create(ctx, sub)
	if err != nil {
		if err == database.ErrConflict {
			c.JSON(http.StatusConflict, gin.H{"error": "подписка на этот сервис за указанный месяц уже существует"})
			return
		}
		log.Printf("ошибка создания подписки: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "внутренняя ошибка сервера"})
		return
	}

	log.Printf("подписка успешно создана | id=%s | %s | %d-%02d | %d₽", id, req.ServiceName, year, month, req.Price)
	c.JSON(http.StatusCreated, gin.H{"id": id})
}

// getSubscriptionHandler — GET /api/subscriptions/:id
// Возвращает одну подписку по её UUID.
func (s *Server) getSubscriptionHandler(c *gin.Context) {
	id := c.Param("id")

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	sub, err := s.DB.Get(ctx, id)
	if err != nil {
		if err == database.ErrNotExist {
			c.JSON(http.StatusNotFound, gin.H{"error": "подписка не найдена"})
			return
		}
		log.Printf("ошибка получения подписки id=%s: %v", id, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "внутренняя ошибка сервера"})
		return
	}

	resp := dto.SubscriptionResponse{
		ID:          sub.ID,
		ServiceName: sub.ServiceName,
		Price:       sub.Price,
		UserID:      sub.UserID,
		YearMonth:   fmt.Sprintf("%d-%02d", sub.Year, sub.Month),
		CreatedAt:   sub.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   sub.UpdatedAt.Format(time.RFC3339),
	}

	c.JSON(http.StatusOK, resp)
}

// updateSubscriptionHandler — PUT /api/subscriptions/:id
// Обновляет название сервиса и/или цену существующей подписки.
func (s *Server) updateSubscriptionHandler(c *gin.Context) {
	id := c.Param("id")
	var req dto.UpdateSubscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "некорректное тело запроса"})
		return
	}

	updates := models.Subscription{}
	if req.ServiceName != nil {
		updates.ServiceName = *req.ServiceName
	}
	if req.Price != nil {
		updates.Price = *req.Price
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	if err := s.DB.Update(ctx, id, updates); err != nil {
		if err == database.ErrNotExist {
			c.JSON(http.StatusNotFound, gin.H{"error": "подписка не найдена"})
			return
		}
		log.Printf("ошибка обновления подписки id=%s: %v", id, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "внутренняя ошибка сервера"})
		return
	}

	log.Printf("подписка успешно обновлена | id=%s", id)
	c.JSON(http.StatusOK, gin.H{"status": "updated"})
}

// deleteSubscriptionHandler — DELETE /api/subscriptions/:id
// Удаляет подписку по ID.
func (s *Server) deleteSubscriptionHandler(c *gin.Context) {
	id := c.Param("id")

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	if err := s.DB.Delete(ctx, id); err != nil {
		if err == database.ErrNotExist {
			c.JSON(http.StatusNotFound, gin.H{"error": "подписка не найдена"})
			return
		}
		log.Printf("ошибка удаления подписки id=%s: %v", id, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "внутренняя ошибка сервера"})
		return
	}

	log.Printf("подписка успешно удалена | id=%s", id)
	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}

// listSubscriptionsHandler — GET /api/subscriptions?user_id=...
// Возвращает список всех подписок или только одного пользователя.
func (s *Server) listSubscriptionsHandler(c *gin.Context) {
	userID := c.Query("user_id")

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	subs, err := s.DB.List(ctx, userID)
	if err != nil {
		log.Printf("ошибка получения списка подписок: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "внутренняя ошибка сервера"})
		return
	}

	var resp []dto.SubscriptionResponse
	for _, sub := range subs {
		resp = append(resp, dto.SubscriptionResponse{
			ID:          sub.ID,
			ServiceName: sub.ServiceName,
			Price:       sub.Price,
			UserID:      sub.UserID,
			YearMonth:   fmt.Sprintf("%d-%02d", sub.Year, sub.Month),
			CreatedAt:   sub.CreatedAt.Format(time.RFC3339),
			UpdatedAt:   sub.UpdatedAt.Format(time.RFC3339),
		})
	}

	c.JSON(http.StatusOK, resp)
}

// sumSubscriptionsHandler — GET /api/subscriptions/total
// Подсчитывает общую стоимость всех подписок за указанный период.
// Поддерживает фильтры по пользователю и сервису.
func (s *Server) sumSubscriptionsHandler(c *gin.Context) {
	var req dto.TotalCostRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "некорректные параметры запроса", "details": err.Error()})
		return
	}

	startYear, startMonth, err := parseYearMonth(req.Start)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "неверный формат параметра start: " + err.Error()})
		return
	}
	endYear, endMonth, err := parseYearMonth(req.End)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "неверный формат параметра end: " + err.Error()})
		return
	}

	userID := ""
	if req.UserID != nil {
		userID = *req.UserID
	}
	service := ""
	if req.Service != nil {
		service = *req.Service
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	total, err := s.DB.SumSubscriptionsCost(ctx, userID, service, startYear, startMonth, endYear, endMonth)
	if err != nil {
		log.Printf("ошибка подсчёта общей стоимости: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "внутренняя ошибка сервера"})
		return
	}

	log.Printf("подсчёт общей стоимости завершён | период=%s..%s | пользователь=%s | сервис=%s | итог=%d₽",
		req.Start, req.End, userID, service, total)

	c.JSON(http.StatusOK, dto.TotalCostResponse{Total: total})
}
