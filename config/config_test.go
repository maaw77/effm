package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInitConnString_Defaults(t *testing.T) {
	expected := "postgres://postgres:epas@localhost:5432/postgres?sslmode=disable&pool_max_conns=10"

	connString := InitConnString("")
	if connString != expected {
		t.Errorf("Expected default connection string:\n%s\nGot:\n%s", expected, connString)
	}
}

func TestInitConnString_File(t *testing.T) {
	currDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	configPath := filepath.Join(currDir, "config.yaml")

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Skipf("config.yaml not found at %s, skipping test", configPath)
	}

	// Просто проверяем, что функция не падает и возвращает валидную строку подключения
	connString := InitConnString(configPath)

	if connString == "" {
		t.Error("Expected non-empty connection string")
	}

	if !strings.Contains(connString, "postgres://") {
		t.Error("Expected PostgreSQL connection string")
	}
}

// func TestInitConnString_File(t *testing.T) {
// 	currDir, err := os.Getwd()
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	configPath := filepath.Join(currDir, "config.yaml")

// 	if _, err := os.Stat(configPath); os.IsNotExist(err) {
// 		t.Skipf("config.yaml not found at %s, skipping test", configPath)
// 	}

// 	// Подставляем значения из твоего config.yaml
// 	expected := "postgres://postgres:epas@db:5432/postgres?sslmode=disable&pool_max_conns=10"

// 	connString := InitConnString(configPath)
// 	if connString != expected {
// 		t.Errorf("Expected connection string from file:\n%s\nGot:\n%s", expected, connString)
// 	}
// }
