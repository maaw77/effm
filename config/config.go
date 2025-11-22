// Package config предоставляет функционал для работы с конфигурацией приложения.
//
// Пакет включает:
// - Загрузку конфигурации из YAML файлов с использованием Viper
// - Формирование строк подключения к PostgreSQL
// - Управление настройками HTTP сервера
// - Поддержку значений по умолчанию
package config

import (
	"fmt"
	"log"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// InitConnString возвращает строку подключения к PostgreSQL.
//
// Функция пытается загрузить конфигурацию из указанного YAML файла. Если файл
// не найден или произошла ошибка чтения, используются значения по умолчанию.
//
// Параметры подключения (в порядке приоритета):
//   - Из конфигурационного файла
//   - Значения по умолчанию:
//   - DB: "postgres"
//   - User: "postgres"
//   - Password: "epas"
//   - Host: "localhost"
//   - Port: 5433
//   - PoolMaxConns: 10
//
// Пример строки подключения:
//
//	"postgres://user:pass@host:port/db?sslmode=disable&pool_max_conns=10"
//
// Аргументы:
//
//	pathConfig - путь к конфигурационному файлу (например, "config/config.yaml")
//
// Возвращает:
//
//	Строку подключения к PostgreSQL в формате URL
func InitConnString(pathConfig string) string {
	dir, file := filepath.Split(pathConfig)
	fileName := strings.Split(file, ".")[0]

	// Установка значений по умолчанию для параметров базы данных
	viper.SetDefault("db.DB", "postgres")
	viper.SetDefault("db.User", "postgres")
	viper.SetDefault("db.Password", "epas")
	viper.SetDefault("db.Host", "localhost")
	viper.SetDefault("db.Port", 5433)
	viper.SetDefault("db.PoolMaxConns", 10)

	viper.SetConfigName(fileName)
	viper.SetConfigType("yaml")
	viper.AddConfigPath(dir)

	// Попытка загрузки конфигурации из файла
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			log.Println("configuration file not found, using default parameters")
		} else {
			log.Fatalf("fatal error reading config file: %v", err)
		}
	} else {
		log.Printf("configuration loaded from %s", pathConfig)
	}

	// Формирование строки подключения
	connString := fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=disable&pool_max_conns=%d",
		viper.GetString("db.User"),
		viper.GetString("db.Password"),
		viper.GetString("db.Host"),
		viper.GetInt("db.Port"),
		viper.GetString("db.DB"),
		viper.GetInt("db.PoolMaxConns"),
	)

	return connString
}

// ServerConfig хранит параметры конфигурации HTTP-сервера.
type ServerConfig struct {
	Port         string        // Порт для прослушивания (например, "8080")
	ReadTimeout  time.Duration // Таймаут чтения запроса
	WriteTimeout time.Duration // Таймаут записи ответа
}

// InitServerConfig читает конфигурацию сервера из настроек Viper.
//
// Ожидаемые ключи в конфигурации:
//
//	server.Port - порт сервера (string)
//	server.ReadTimeout - таймаут чтения (duration)
//	server.WriteTimeout - таймаут записи (duration)
//
// Возвращает:
//
//	ServerConfig с загруженными настройками сервера
func InitServerConfig() ServerConfig {
	return ServerConfig{
		Port:         viper.GetString("server.Port"),
		ReadTimeout:  viper.GetDuration("server.ReadTimeout"),
		WriteTimeout: viper.GetDuration("server.WriteTimeout"),
	}
}

// ConnStringForMigrate возвращает строку подключения без pgx-специфичных параметров.
//
// Функция используется для утилит миграции (golang-migrate), которые могут
// не поддерживать параметры, специфичные для драйвера pgx (например, pool_max_conns).
//
// Процесс:
//  1. Получает полную строку подключения через InitConnString
//  2. Удаляет параметр pool_max_conns из query string
//  3. Возвращает очищенную строку подключения
//
// Аргументы:
//
//	pathConfig - путь к конфигурационному файлу
//
// Возвращает:
//
//	Строку подключения, совместимую с golang-migrate
func ConnStringForMigrate(pathConfig string) string {
	// Получение полной строки подключения
	full := InitConnString(pathConfig)

	// Удаление pgx-специфичных параметров (pool_max_conns)
	if strings.Contains(full, "pool_max_conns") {
		u, err := url.Parse(full)
		if err != nil {
			return full // возвращаем исходную строку в случае ошибки парсинга
		}
		q := u.Query()
		q.Del("pool_max_conns")
		u.RawQuery = q.Encode()
		return u.String()
	}
	return full
}
