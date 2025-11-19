//go:build migrate

package main

import (
	"log"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/maaw77/effm/config"
)

func init() {
	log.Println("running database migrations")

	connStr := config.ConnStringForMigrate("config/config.yaml")

	m, err := migrate.New("file://migrations", connStr)
	if err != nil {
		log.Fatalf("failed to initialize migrator: %v", err)
	}
	defer m.Close()

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		log.Fatalf("migration failed: %v", err)
	}

	if err == migrate.ErrNoChange {
		log.Println("migrations are up to date")
	} else {
		log.Println("migrations applied")
	}

	log.Println("migrations completed")
}
