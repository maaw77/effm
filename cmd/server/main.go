// Package main запускает HTTP-сервер для работы с подписками.
// Сервер использует Gin и подключается к PostgreSQL через internal/database.
// Конфигурация читается из config/config.yaml.
package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/maaw77/effm/config"
	"github.com/maaw77/effm/internal/database"
	"github.com/maaw77/effm/internal/server"
)

func main() {
	// настройка логирования: дата, время, короткий путь к файлу
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	// загрузка конфигурации базы данных и сервера
	ctx := context.Background()
	connStr := config.InitConnString("config/config.yaml")
	serverCfg := config.InitServerConfig()

	// подключение к базе данных
	db, err := database.NewSubscriptionDatabase(ctx, connStr)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Close()

	// создание сервера с конфигурацией
	srv := server.NewServer(db, serverCfg)

	// запуск сервера в отдельной горутине
	go func() {
		if err := srv.Run(); err != nil {
			log.Fatalf("server exited with error: %v", err)
		}
	}()

	// ожидание сигнала завершения (graceful shutdown)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("received shutdown signal, stopping server...")

	// плавное завершение работы сервера (опционально)
	time.Sleep(1 * time.Second)
	log.Println("server stopped successfully")
}
