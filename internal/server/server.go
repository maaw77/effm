// Package server реализует HTTP-сервер для работы с подписками.
// Сервер использует Gin и предоставляет REST API для CRUD операций с подписками.
// Подключение к базе данных выполняется через internal/database.
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
	c.JSON(http.StatusOK, sub)
}

// createSubscriptionHandler создаёт новую подписку.
// POST /api/subscriptions
func (s *Server) createSubscriptionHandler(c *gin.Context) {
	var sub models.Subscription
	if err := c.ShouldBindJSON(&sub); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	if sub.UserID == "" {
		sub.UserID = uuid.New().String()
	}
	if sub.StartDate.IsZero() {
		sub.StartDate = time.Now()
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
func (s *Server) updateSubscriptionHandler(c *gin.Context) {
	id := c.Param("id")
	var sub models.Subscription
	if err := c.ShouldBindJSON(&sub); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
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
	c.JSON(http.StatusOK, subs)
}
