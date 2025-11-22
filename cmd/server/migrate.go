//go:build migrate

// Package main предоставляет утилиту для выполнения миграций базы данных.
//
// Данный пакет собирается только при наличии тега "migrate" и предназначен
// для применения миграций PostgreSQL при развертывании приложения.
//
// Использование:
//
//	go run -tags migrate cmd/migrate/main.go
package main

import (
	"log"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/maaw77/effm/config"
)

// init выполняется при запуске приложения и применяет все pending миграции.
//
// Процесс миграции:
// 1. Загружает строку подключения из конфигурации
// 2. Инициализирует мигратор с указанием пути к файлам миграций
// 3. Применяет все доступные миграции (m.Up())
// 4. Корректно обрабатывает случай, когда миграции уже применены
//
// В случае ошибки подключения или применения миграций процесс завершается с fatal error.
func init() {
	log.Println("running database migrations")

	// Загрузка строки подключения для миграций
	connStr := config.ConnStringForMigrate("config/config.yaml")

	// Инициализация мигратора с файлами из директории migrations
	m, err := migrate.New("file://migrations", connStr)
	if err != nil {
		log.Fatalf("failed to initialize migrator: %v", err)
	}
	defer m.Close()

	// Применение всех pending миграций
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		log.Fatalf("migration failed: %v", err)
	}

	// Логирование результата выполнения миграций
	if err == migrate.ErrNoChange {
		log.Println("migrations are up to date")
	} else {
		log.Println("migrations applied")
	}

	log.Println("migrations completed")
}
