package config

import (
	"fmt"
	"log"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

// InitConnString возвращает строку подключения к PostgreSQL.
// Если файл конфигурации не найден, используются значения по умолчанию.
func InitConnString(pathConfig string) string {
	dir, file := filepath.Split(pathConfig)
	fileName := strings.Split(file, ".")[0]

	// Значения по умолчанию
	viper.SetDefault("db.DB", "postgres")
	viper.SetDefault("db.User", "postgres")
	viper.SetDefault("db.Password", "epas")
	viper.SetDefault("db.Host", "localhost")
	viper.SetDefault("db.Port", 5433)
	viper.SetDefault("db.PoolMaxConns", 10)

	viper.SetConfigName(fileName)
	viper.SetConfigType("yaml")
	viper.AddConfigPath(dir)

	// Чтение конфигурации
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			log.Println("⚠️ Configuration file not found, using default parameters")
		} else {
			log.Fatalf("❌ Fatal error reading config file: %v", err)
		}
	} else {
		log.Printf("✅ Loaded configuration from %s\n", pathConfig)
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
