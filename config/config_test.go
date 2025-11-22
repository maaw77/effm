package config

import (
	"os"
	"path/filepath"
	"testing"
)

// TestInitConnString_Defaults проверяет формирование строки подключения
// с параметрами по умолчанию, когда конфигурационный файл не указан.
//
// Тест проверяет:
// - Корректность формирования строки подключения
// - Соответствие значений по умолчанию (localhost:5433)
// - Наличие всех необходимных параметров (sslmode, pool_max_conns)
func TestInitConnString_Defaults(t *testing.T) {
	expected := "postgres://postgres:epas@localhost:5433/postgres?sslmode=disable&pool_max_conns=10"

	connString := InitConnString("")
	if connString != expected {
		t.Errorf("Expected default connection string:\n%s\nGot:\n%s", expected, connString)
	}
}

// TestInitConnString_File проверяет загрузку строки подключения из конфигурационного файла.
//
// Тест:
// 1. Проверяет наличие config.yaml в текущей директории
// 2. Если файл отсутствует - пропускает тест
// 3. Если файл присутствует - проверяет корректность загрузки параметров
//
// Примечание: тест ожидает конкретные значения из config.yaml (db:5432)
// и может потребовать обновления при изменении конфигурации.
func TestInitConnString_File(t *testing.T) {
	currDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	configPath := filepath.Join(currDir, "config.yaml")

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Skipf("config.yaml not found at %s, skipping test", configPath)
	}

	// Ожидаемые значения должны соответствовать config.yaml
	expected := "postgres://postgres:epas@db:5432/postgres?sslmode=disable&pool_max_conns=10"

	connString := InitConnString(configPath)
	if connString != expected {
		t.Errorf("Expected connection string from file:\n%s\nGot:\n%s", expected, connString)
	}
}
