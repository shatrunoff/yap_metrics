package repository

import (
	"database/sql"
	"fmt"
	"os"
	"strconv"
	"strings"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

type Migrator struct {
	db         *sql.DB
	migrations string
}

func NewMigrator(db *sql.DB, migrationsPath string) *Migrator {
	return &Migrator{
		db:         db,
		migrations: migrationsPath,
	}
}

// Путь по умолчанию - migrations
func NewMigratorWithDefaultPath(db *sql.DB) *Migrator {
	return NewMigrator(db, "migrations")
}

func (m *Migrator) RunMigrations() error {
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("failed to set dialect: %w", err)
	}

	// Проверяем существование директории с миграциями
	if _, err := os.Stat(m.migrations); os.IsNotExist(err) {
		return fmt.Errorf("migrations directory does not exist: %s", m.migrations)
	}

	if err := goose.Up(m.db, m.migrations); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	return nil
}

func (m *Migrator) Status() (string, error) {
	if err := goose.SetDialect("postgres"); err != nil {
		return "", fmt.Errorf("failed to set dialect: %w", err)
	}

	migrations, err := goose.CollectMigrations(m.migrations, 0, goose.MaxVersion)
	if err != nil {
		return "", fmt.Errorf("failed to collect migrations: %w", err)
	}

	var status strings.Builder
	status.WriteString("Migration Status:\n")

	for _, migration := range migrations {
		exists, err := goose.GetDBVersion(m.db)
		if err != nil {
			return "", fmt.Errorf("failed to get DB version: %w", err)
		}

		applied := exists >= migration.Version
		status.WriteString(fmt.Sprintf("  %s: %s\n",
			strconv.FormatInt(migration.Version, 10),
			map[bool]string{true: "APPLIED", false: "PENDING"}[applied]))
	}

	return status.String(), nil
}
