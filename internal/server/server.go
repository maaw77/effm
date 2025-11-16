// Package server реализует HTTP-сервер для работы с подписками.
// Сервер использует Gin и предоставляет REST API для CRUD операций с подписками.
// Подключение к базе данных выполняется через internal/database.
// Обработчики используют DTO из internal/dto и модель Subscription из internal/models.
package server

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/maaw77/effm/config"
	"github.com/maaw77/effm/internal/database"
	"github.com/maaw77/effm/internal/dto"
	"github.com/maaw77/effm/internal/models"
)

// Server хранит объект Gin Engine и ссылку на базу данных.
type Server struct {
	DB     *database.SubscriptionDatabase
	Router *gin.Engine
	Cfg    config.ServerConfig
}

// NewServer создаёт новый HTTP-сервер с настройками из config.ServerConfig и подключением к базе данных.
func NewServer(db *database.SubscriptionDatabase, cfg config.ServerConfig) *Server {
	r := gin.Default()
	s := &Server{
		DB:     db,
		Router: r,
		Cfg:    cfg,
	}
	s.registerRoutes()
	return s
}

// Run запускает HTTP-сервер с настройками таймаутов и порта.
func (s *Server) Run() error {
	log.Printf("server running on port %s", s.Cfg.Port)

	httpServer := &http.Server{
		Addr:         ":" + s.Cfg.Port,
		Handler:      s.Router,
		ReadTimeout:  s.Cfg.ReadTimeout,
		WriteTimeout: s.Cfg.WriteTimeout,
	}

	return httpServer.ListenAndServe()
}

// registerRoutes регистрирует маршруты для API подписок.
func (s *Server) registerRoutes() {
	api := s.Router.Group("/api")
	{
		api.GET("/subscriptions/:id", s.getSubscriptionHandler)
		api.POST("/subscriptions", s.createSubscriptionHandler)
		api.PUT("/subscriptions/:id", s.updateSubscriptionHandler)
		api.DELETE("/subscriptions/:id", s.deleteSubscriptionHandler)
		api.GET("/subscriptions", s.listSubscriptionsHandler)
	}
}

// getSubscriptionHandler возвращает подписку по ID.
// GET /api/subscriptions/:id
// Параметры:
//   - id (path) : UUID подписки
//
// Ответ:
//   - 200 OK : SubscriptionResponse
//   - 404 Not Found : если подписка не найдена
//   - 500 Internal Server Error : при ошибке сервера
func (s *Server) getSubscriptionHandler(c *gin.Context) {
	id := c.Param("id")
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	sub, err := s.DB.GetSubscription(ctx, id)
	if err != nil {
		if err == database.ErrNotExist {
			c.JSON(http.StatusNotFound, gin.H{"error": "subscription not found"})
			return
		}
		log.Printf("error getting subscription: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	resp := dto.SubscriptionResponse{
		ID:          sub.ID,
		ServiceName: sub.ServiceName,
		Price:       sub.Price,
		UserID:      sub.UserID,
		StartDate:   sub.StartDate.Format("01-2006"),
		EndDate:     formatTimePtr(sub.EndDate),
		CreatedAt:   sub.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   sub.UpdatedAt.Format(time.RFC3339),
		Status:      "active",
	}
	c.JSON(http.StatusOK, resp)
}

// createSubscriptionHandler создаёт новую подписку.
// POST /api/subscriptions
// Тело запроса: CreateSubscriptionRequest
// Ответ:
//   - 201 Created : { "id": <UUID> }
//   - 400 Bad Request : некорректное тело запроса
//   - 409 Conflict : если подписка уже существует
//   - 500 Internal Server Error : ошибка сервера
func (s *Server) createSubscriptionHandler(c *gin.Context) {
	var req dto.CreateSubscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	userID := req.UserID
	if userID == "" {
		userID = uuid.New().String()
	}

	start, err := parseMonthYear(req.StartDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid start_date format"})
		return
	}

	var endPtr *time.Time
	if req.EndDate != nil {
		end, err := parseMonthYear(*req.EndDate)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid end_date format"})
			return
		}
		endPtr = &end
	}

	sub := models.Subscription{
		ServiceName: req.ServiceName,
		Price:       req.Price,
		UserID:      userID,
		StartDate:   start,
		EndDate:     endPtr,
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	id, err := s.DB.CreateIfNotExist(ctx, sub)
	if err != nil {
		if err == database.ErrExist {
			c.JSON(http.StatusConflict, gin.H{"error": "subscription already exists"})
			return
		}
		log.Printf("error creating subscription: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": id})
}

// updateSubscriptionHandler обновляет подписку по ID.
// PUT /api/subscriptions/:id
// Тело запроса: UpdateSubscriptionRequest
// Ответ:
//   - 200 OK : { "status": "updated" }
//   - 404 Not Found : если подписка не найдена
//   - 400 Bad Request : некорректное тело запроса
//   - 500 Internal Server Error : ошибка сервера
func (s *Server) updateSubscriptionHandler(c *gin.Context) {
	id := c.Param("id")
	var req dto.UpdateSubscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	sub := models.Subscription{}
	if req.ServiceName != nil {
		sub.ServiceName = *req.ServiceName
	}
	if req.Price != nil {
		sub.Price = *req.Price
	}
	if req.StartDate != nil {
		start, err := parseMonthYear(*req.StartDate)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid start_date format"})
			return
		}
		sub.StartDate = start
	}
	if req.EndDate != nil {
		end, err := parseMonthYear(*req.EndDate)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid end_date format"})
			return
		}
		sub.EndDate = &end
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	err := s.DB.UpdateSubscription(ctx, id, sub)
	if err != nil {
		if err == database.ErrNotExist {
			c.JSON(http.StatusNotFound, gin.H{"error": "subscription not found"})
			return
		}
		log.Printf("error updating subscription: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "updated"})
}

// deleteSubscriptionHandler удаляет подписку по ID.
// DELETE /api/subscriptions/:id
// Ответ:
//   - 200 OK : { "status": "deleted" }
//   - 404 Not Found : если подписка не найдена
//   - 500 Internal Server Error : ошибка сервера
func (s *Server) deleteSubscriptionHandler(c *gin.Context) {
	id := c.Param("id")
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	err := s.DB.DeleteSubscription(ctx, id)
	if err != nil {
		if err == database.ErrNotExist {
			c.JSON(http.StatusNotFound, gin.H{"error": "subscription not found"})
			return
		}
		log.Printf("error deleting subscription: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}

// listSubscriptionsHandler возвращает список подписок.
// GET /api/subscriptions?user_id=...
// Параметры:
//   - user_id (query, optional) : фильтрация по пользователю
//
// Ответ:
//   - 200 OK : []SubscriptionResponse
//   - 500 Internal Server Error : ошибка сервера
func (s *Server) listSubscriptionsHandler(c *gin.Context) {
	userID := c.Query("user_id")
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	subs, err := s.DB.ListSubscriptions(ctx, userID)
	if err != nil {
		log.Printf("error listing subscriptions: %v", err)
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
			StartDate:   sub.StartDate.Format("01-2006"),
			EndDate:     formatTimePtr(sub.EndDate),
			CreatedAt:   sub.CreatedAt.Format(time.RFC3339),
			UpdatedAt:   sub.UpdatedAt.Format(time.RFC3339),
			Status:      "active",
		})
	}

	c.JSON(http.StatusOK, resp)
}

// --- вспомогательные функции ---

func parseMonthYear(s string) (time.Time, error) {
	return time.Parse("01-2006", s)
}

func formatTimePtr(t *time.Time) *string {
	if t == nil {
		return nil
	}
	str := t.Format("01-2006")
	return &str
}
