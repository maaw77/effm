// Package main запускает HTTP-сервер для работы с подписками.
// Сервер использует Gin и подключается к PostgreSQL через internal/database.
// Конфигурация читается из config/config.yaml.

// cmd/server/main.go
// Точка входа в приложение.
// Использует реальный config-пакет с InitConnString и InitServerConfig.

// cmd/server/main.go
// Точка входа в приложение.
// Полностью совместима с твоим текущим database.Close() → void

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
	"github.com/maaw77/effm/internal/database"
	"github.com/maaw77/effm/internal/server"
)

func main() {
	// Инициализируем конфигурацию из config/config.yaml
	// Если файла нет — используются значения по умолчанию
	connStr := config.InitConnString("config/config.yaml")
	serverCfg := config.InitServerConfig()

	log.Printf("server will listen on :%s", serverCfg.Port)

	// Подключаемся к PostgreSQL
	db, err := database.NewSubscriptionDatabase(context.Background(), connStr)
	if err != nil {
		log.Fatalf("failed to connect to PostgreSQL: %v", err)
	}

	// При завершении процесса корректно закрываем пул соединений

	defer db.Close()

	// Создаём HTTP-сервер на базе Gin
	srv := server.NewServer(db)

	// Оборачиваем Gin-роутер в net/http.Server

	httpServer := &http.Server{
		Addr:         ":" + serverCfg.Port,
		Handler:      srv.Router,
		ReadTimeout:  serverCfg.ReadTimeout,
		WriteTimeout: serverCfg.WriteTimeout,
	}

	// Запускаем сервер в отдельной горутине
	go func() {
		log.Printf("server started at http://localhost:%s", serverCfg.Port)
		log.Printf("API docs: http://localhost:%s/swagger/index.html (если swag включён)", serverCfg.Port)

		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	// Ожидаем сигнал завершения работы (Ctrl+C)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("shutdown signal received, stopping server...")

	// Даём активным запросам до 10 секунд на завершение
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		log.Printf("server forced to shutdown: %v", err)
	} else {
		log.Println("server stopped gracefully")
	}
}
