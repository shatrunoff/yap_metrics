package repository

import (
	"database/sql"
	"fmt"
	"log"

	"github.com/pressly/goose/v3"
)

type Migrator struct {
	db *sql.DB
}

func NewMigrator(db *sql.DB) *Migrator {
	return &Migrator{db: db}
}

func (m *Migrator) RunMigrations() error {
	log.Printf("Running database migrations...")

	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("failed to set dialect: %w", err)
	}

	// Получаем текущую версию миграций
	currentVersion, err := goose.GetDBVersion(m.db)
	if err != nil {
		return fmt.Errorf("failed to get current migration version: %w", err)
	}
	log.Printf("Current migration version: %d", currentVersion)

	// Запускаем миграции
	if err := goose.Up(m.db, "migrations"); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	// Проверяем результат
	newVersion, err := goose.GetDBVersion(m.db)
	if err != nil {
		return fmt.Errorf("failed to get new migration version: %w", err)
	}
	log.Printf("New migration version: %d", newVersion)

	if newVersion > currentVersion {
		log.Printf("Applied %d migration(s)", newVersion-currentVersion)
	} else {
		log.Printf("No new migrations to apply")
	}

	return nil
}
