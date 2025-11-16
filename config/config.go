package config

import (
	"fmt"
	"log"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// InitConnString возвращает строку подключения к PostgreSQL.
// Если файл конфигурации не найден, используются значения по умолчанию.
func InitConnString(pathConfig string) string {
	dir, file := filepath.Split(pathConfig)
	fileName := strings.Split(file, ".")[0]

	// значения по умолчанию
	viper.SetDefault("db.DB", "postgres")
	viper.SetDefault("db.User", "postgres")
	viper.SetDefault("db.Password", "epas")
	viper.SetDefault("db.Host", "localhost")
	viper.SetDefault("db.Port", 5433)
	viper.SetDefault("db.PoolMaxConns", 10)

	viper.SetConfigName(fileName)
	viper.SetConfigType("yaml")
	viper.AddConfigPath(dir)

	// чтение конфигурации
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			log.Println("configuration file not found, using default parameters")
		} else {
			log.Fatalf("fatal error reading config file: %v", err)
		}
	} else {
		log.Printf("configuration loaded from %s", pathConfig)
	}

	// формирование строки подключения
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
	Port         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

// InitServerConfig читает конфиг сервера из YAML
func InitServerConfig() ServerConfig {
	return ServerConfig{
		Port:         viper.GetString("server.Port"),
		ReadTimeout:  viper.GetDuration("server.ReadTimeout"),
		WriteTimeout: viper.GetDuration("server.WriteTimeout"),
	}
}
