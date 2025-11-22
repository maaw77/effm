// Package main является точкой входа для приложения управления подписками.
//
// Приложение предоставляет REST API для работы с подписками, включая:
// - Создание, чтение, обновление и удаление подписки
// - Просмотр списка подписок
// - Автоматическую генерацию Swagger документации
//
// Запуск: go run cmd/server/main.go
// Swagger UI: http://localhost:8080/swagger/index.html
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/maaw77/effm/config"
	"github.com/maaw77/effm/docs"
	"github.com/maaw77/effm/internal/database"
	"github.com/maaw77/effm/internal/server"

	_ "github.com/maaw77/effm/docs"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// main инициализирует и запускает HTTP сервер приложения.
//
// Основные этапы:
// 1. Загрузка конфигурации БД и сервера
// 2. Подключение к PostgreSQL
// 3. Инициализация маршрутов и middleware
// 4. Запуск HTTP сервера
// 5. Обработка graceful shutdown
//
// Конфигурация загружается из config/config.yaml или использует значения по умолчанию.
func main() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	// Настройка Swagger перед запуском сервера
	docs.SwaggerInfo.Host = "localhost:8080"
	docs.SwaggerInfo.BasePath = "/api"
	docs.SwaggerInfo.Schemes = []string{"http"}
	docs.SwaggerInfo.Title = "EFFM API"
	docs.SwaggerInfo.Description = "API для управления подписками"
	docs.SwaggerInfo.Version = "1.0"

	// Инициализация подключения к базе данных
	connStr := config.InitConnString("config/config.yaml")
	serverCfg := config.InitServerConfig()
	log.Printf("server will listen on :%s", serverCfg.Port)

	// Подключение к PostgreSQL
	db, err := database.NewSubscriptionDatabase(context.Background(), connStr)
	if err != nil {
		log.Fatalf("failed to connect to PostgreSQL: %v", err)
	}
	defer db.Close()

	// Инициализация HTTP сервера с роутингом
	srv := server.NewServer(db)

	// Swagger UI — доступен по http://localhost:8080/swagger/index.html
	srv.Router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Настройка HTTP сервера с таймаутами
	httpServer := &http.Server{
		Addr:         ":" + serverCfg.Port,
		Handler:      srv.Router,
		ReadTimeout:  serverCfg.ReadTimeout,
		WriteTimeout: serverCfg.WriteTimeout,
	}

	// Запуск сервера в отдельной goroutine
	go func() {
		log.Printf("server started at http://localhost:%s", serverCfg.Port)
		log.Printf("Swagger UI: http://localhost:%s/swagger/index.html", serverCfg.Port)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	// Обработка graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("shutdown signal received...")

	// Graceful shutdown с таймаутом
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(ctx); err != nil {
		log.Printf("server forced to shutdown: %v", err)
	} else {
		log.Println("server stopped gracefully")
	}
}
