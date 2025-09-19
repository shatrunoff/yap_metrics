package repository

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"

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

	// Получаем абсолютный путь к миграциям
	migrationsDir, err := getMigrationsDir()
	if err != nil {
		return fmt.Errorf("failed to get migrations directory: %w", err)
	}

	// Проверяем существование директории
	if _, err := os.Stat(migrationsDir); os.IsNotExist(err) {
		return fmt.Errorf("migrations directory does not exist: %s", migrationsDir)
	}

	log.Printf("Using migrations from: %s", migrationsDir)

	// Запускаем миграции
	if err := goose.Up(m.db, migrationsDir); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	log.Printf("Database migrations completed successfully")
	return nil
}

func getMigrationsDir() (string, error) {
	// Пытаемся найти migrations директорию
	possiblePaths := []string{
		"migrations",
		"./migrations",
		"../migrations",
	}

	for _, path := range possiblePaths {
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}

	return "", fmt.Errorf("migrations directory not found")
}

func (m *Migrator) Status() (string, error) {
	if err := goose.SetDialect("postgres"); err != nil {
		return "", fmt.Errorf("failed to set dialect: %w", err)
	}

	migrationsDir, err := getMigrationsDir()
	if err != nil {
		return "", fmt.Errorf("failed to get migrations directory: %w", err)
	}

	migrations, err := goose.CollectMigrations(migrationsDir, 0, goose.MaxVersion)
	if err != nil {
		return "", fmt.Errorf("failed to collect migrations: %w", err)
	}

	var status strings.Builder
	status.WriteString("Migration Status:\n")

	currentVersion, err := goose.GetDBVersion(m.db)
	if err != nil {
		return "", fmt.Errorf("failed to get DB version: %w", err)
	}

	for _, migration := range migrations {
		applied := currentVersion >= migration.Version
		statusText := "PENDING"
		if applied {
			statusText = "APPLIED"
		}
		status.WriteString(fmt.Sprintf(" %d: %s\n",
			migration.Version,
			statusText))
	}

	return status.String(), nil
}
