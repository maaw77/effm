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
//	@contact.email   support@example.com
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

	var req dto.UpdateSubscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "некорректное тело запроса", "details": err.Error()})
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

// deleteSubscriptionHandler — удаляет подписку по UUID
// @Summary      Удалить подписку
// @Tags         subscriptions
// @Param        id  path     string true "UUID подписки"
// @Success      200 {object} map[string]string{status=string}
// @Failure      404,500 {object} map[string]string
// @Router       /subscriptions/{id} [delete]
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

	log.Printf("подсчёт общей стоимости | %s..%s | user=%s | service=%s | total=%d₽", req.Start, req.End, userID, service, total)
	c.JSON(http.StatusOK, dto.TotalCostResponse{Total: total})
}
