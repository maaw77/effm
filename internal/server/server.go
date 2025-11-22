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

// Server — основной объект HTTP-сервера
type Server struct {
	Router *gin.Engine
	DB     *database.SubscriptionDatabase
}

// NewServer создаёт и настраивает сервер со всеми роутами
//
//	@title           Subscriptions API
//	@version         1.0
//	@description     REST-сервис для агрегации данных об онлайн-подписках пользователей
//	@contact.name    API Support
//	@contact.email   maaw@mail.ru
//	@license.name    MIT
//	@host            localhost:8080
//	@BasePath        /api
func NewServer(db *database.SubscriptionDatabase) *Server {
	s := &Server{
		Router: gin.Default(),
		DB:     db,
	}

	api := s.Router.Group("/api")
	{
		api.POST("/subscriptions", s.createSubscriptionHandler)
		api.GET("/subscriptions", s.listSubscriptionsHandler)
		api.GET("/subscriptions/:id", s.getSubscriptionHandler)
		api.PUT("/subscriptions/:id", s.updateSubscriptionHandler)
		api.DELETE("/subscriptions/:id", s.deleteSubscriptionHandler)
		api.GET("/subscriptions/total", s.sumSubscriptionsHandler)
	}

	return s
}

// parseYearMonth преобразует строку "2025-07" в год и месяц с валидацией
func parseYearMonth(s string) (int, int, error) {
	var year, month int
	n, err := fmt.Sscanf(s, "%d-%02d", &year, &month)
	if err != nil || n != 2 {
		return 0, 0, fmt.Errorf("invalid date format: %s (expected YYYY-MM)", s)
	}
	if month < 1 || month > 12 {
		return 0, 0, fmt.Errorf("month must be between 01 and 12, got: %02d", month)
	}
	if year < 2000 || year > 2100 {
		return 0, 0, fmt.Errorf("year must be between 2000 and 2100, got: %d", year)
	}
	return year, month, nil
}

// createSubscriptionHandler — создаёт запись об оплате подписки за конкретный месяц
// @Summary      Создать запись о подписке за конкретный месяц
// @Description  Одна запись = оплата за один месяц. Дубли по пользователю + сервису + месяцу запрещены
// @Tags         subscriptions
// @Accept       json
// @Produce      json
// @Param        request body     dto.CreateSubscriptionRequest true "Данные подписки"
// @Success      201    {object} map[string]string{id=string} "UUID созданной записи"
// @Failure      400    {object} map[string]string
// @Failure      409    {object} map[string]string "подписка уже существует"
// @Failure      500    {object} map[string]string
// @Router       /subscriptions [post]
func (s *Server) createSubscriptionHandler(c *gin.Context) {
	log.Printf("HTTP: creating subscription | method=POST | path=/api/subscriptions")

	var req dto.CreateSubscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("HTTP: invalid request body | error=%v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body", "details": err.Error()})
		return
	}

	year, month, err := parseYearMonth(req.YearMonth)
	if err != nil {
		log.Printf("HTTP: invalid date format | year_month=%s | error=%v", req.YearMonth, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	log.Printf("HTTP: creating subscription | user=%s | service=%s | period=%d-%02d | price=%d",
		req.UserID, req.ServiceName, year, month, req.Price)

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
			log.Printf("HTTP: subscription conflict | user=%s | service=%s | period=%d-%02d",
				req.UserID, req.ServiceName, year, month)
			c.JSON(http.StatusConflict, gin.H{"error": "subscription for this service in specified month already exists"})
			return
		}
		log.Printf("HTTP: subscription creation error | error=%v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	log.Printf("HTTP: subscription successfully created | id=%s | user=%s | service=%s | period=%d-%02d | price=%d",
		id, req.UserID, req.ServiceName, year, month, req.Price)
	c.JSON(http.StatusCreated, gin.H{"id": id})
}

// getSubscriptionHandler — возвращает одну подписку по её UUID
// @Summary      Получить подписку по ID
// @Tags         subscriptions
// @Produce      json
// @Param        id  path     string true "UUID подписки"
// @Success      200 {object} dto.SubscriptionResponse
// @Failure      404 {object} map[string]string
// @Failure      500 {object} map[string]string
// @Router       /subscriptions/{id} [get]
func (s *Server) getSubscriptionHandler(c *gin.Context) {
	id := c.Param("id")
	log.Printf("HTTP: retrieving subscription | method=GET | path=/api/subscriptions/%s", id)

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	sub, err := s.DB.Get(ctx, id)
	if err != nil {
		if err == database.ErrNotExist {
			log.Printf("HTTP: subscription not found | id=%s", id)
			c.JSON(http.StatusNotFound, gin.H{"error": "subscription not found"})
			return
		}
		log.Printf("HTTP: subscription retrieval error | id=%s | error=%v", id, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	log.Printf("HTTP: subscription retrieved | id=%s | service=%s | user=%s", id, sub.ServiceName, sub.UserID)

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

// updateSubscriptionHandler — частично обновляет подписку (только название и/или цену)
// @Summary      Частично обновить подписку
// @Description  Можно менять только название сервиса и/или цену. Месяц и год — неизменяемые
// @Tags         subscriptions
// @Accept       json
// @Produce      json
// @Param        id      path     string                        true  "UUID подписки"
// @Param        request  body     dto.UpdateSubscriptionRequest true  "Поля для обновления"
// @Success      200 {object} map[string]string{status=string}
// @Failure      400,404,500 {object} map[string]string
// @Router       /subscriptions/{id} [put]
func (s *Server) updateSubscriptionHandler(c *gin.Context) {
	id := c.Param("id")
	log.Printf("HTTP: updating subscription | method=PUT | path=/api/subscriptions/%s", id)

	var req dto.UpdateSubscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("HTTP: invalid request body | id=%s | error=%v", id, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body", "details": err.Error()})
		return
	}

	log.Printf("HTTP: updating subscription fields | id=%s | service=%v | price=%v",
		id, req.ServiceName, req.Price)

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
			log.Printf("HTTP: subscription not found for update | id=%s", id)
			c.JSON(http.StatusNotFound, gin.H{"error": "subscription not found"})
			return
		}
		log.Printf("HTTP: subscription update error | id=%s | error=%v", id, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	log.Printf("HTTP: subscription successfully updated | id=%s", id)
	c.JSON(http.StatusOK, gin.H{"status": "updated"})
}

// deleteSubscriptionHandler — удаляет подписку по UUID
// @Summary      Удалить подписку
// @Tags         subscriptions
// @Param        id  path     string true "UUID подписки"
// @Success      200 {object} map[string]string{status=string}
// @Failure      404,500 {object} map[string]string
// @Router       /subscriptions/{id} [delete]
func (s *Server) deleteSubscriptionHandler(c *gin.Context) {
	id := c.Param("id")
	log.Printf("HTTP: deleting subscription | method=DELETE | path=/api/subscriptions/%s", id)

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	if err := s.DB.Delete(ctx, id); err != nil {
		if err == database.ErrNotExist {
			log.Printf("HTTP: subscription not found for deletion | id=%s", id)
			c.JSON(http.StatusNotFound, gin.H{"error": "subscription not found"})
			return
		}
		log.Printf("HTTP: subscription deletion error | id=%s | error=%v", id, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	log.Printf("HTTP: subscription successfully deleted | id=%s", id)
	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}

// listSubscriptionsHandler — возвращает список всех подписок (с фильтром по пользователю)
// @Summary      Список всех подписок
// @Description  Если передать user_id — вернёт только подписки этого пользователя
// @Tags         subscriptions
// @Produce      json
// @Param        user_id query string false "UUID пользователя для фильтрации"
// @Success      200 {array} dto.SubscriptionResponse
// @Failure      500 {object} map[string]string
// @Router       /subscriptions [get]
func (s *Server) listSubscriptionsHandler(c *gin.Context) {
	userID := c.Query("user_id")

	if userID != "" {
		log.Printf("HTTP: listing subscriptions for user | method=GET | path=/api/subscriptions | user=%s", userID)
	} else {
		log.Printf("HTTP: listing all subscriptions | method=GET | path=/api/subscriptions")
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	subs, err := s.DB.List(ctx, userID)
	if err != nil {
		log.Printf("HTTP: subscription list retrieval error | error=%v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
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

	if userID != "" {
		log.Printf("HTTP: returned %d subscriptions for user | user=%s", len(resp), userID)
	} else {
		log.Printf("HTTP: returned %d total subscriptions", len(resp))
	}

	c.JSON(http.StatusOK, resp)
}

// sumSubscriptionsHandler — считает суммарную стоимость подписок за период с фильтрами
// @Summary      Подсчёт общей стоимости подписок за период
// @Description  Период задаётся включительно. Поддерживает фильтры по пользователю и сервису
// @Tags         subscriptions
// @Produce      json
// @Param        start   query string true "Начало периода (YYYY-MM)" example(2025-01)
// @Param        end     query string true "Конец периода (YYYY-MM)" example(2025-12)
// @Param        user_id query string false "UUID пользователя" example(f47ac10b-58cc-4372-a567-0e02b2c3d479)
// @Param        service  query string false "Название сервиса"
// @Success      200 {object} dto.TotalCostResponse
// @Failure      400,500 {object} map[string]string
// @Router       /subscriptions/total [get]
func (s *Server) sumSubscriptionsHandler(c *gin.Context) {
	log.Printf("HTTP: calculating total cost | method=GET | path=/api/subscriptions/total")

	var req dto.TotalCostRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		log.Printf("HTTP: invalid query parameters | error=%v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid query parameters", "details": err.Error()})
		return
	}

	startYear, startMonth, err := parseYearMonth(req.Start)
	if err != nil {
		log.Printf("HTTP: invalid start parameter | start=%s | error=%v", req.Start, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid start parameter format: " + err.Error()})
		return
	}
	endYear, endMonth, err := parseYearMonth(req.End)
	if err != nil {
		log.Printf("HTTP: invalid end parameter | end=%s | error=%v", req.End, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid end parameter format: " + err.Error()})
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

	log.Printf("HTTP: calculating total cost | period=%s..%s | user=%s | service=%s",
		req.Start, req.End, userID, service)

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	total, err := s.DB.SumSubscriptionsCost(ctx, userID, service, startYear, startMonth, endYear, endMonth)
	if err != nil {
		log.Printf("HTTP: total cost calculation error | error=%v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	log.Printf("HTTP: total subscription cost calculated | period=%s..%s | user=%s | service=%s | total=%d",
		req.Start, req.End, userID, service, total)
	c.JSON(http.StatusOK, dto.TotalCostResponse{Total: total})
}
