package storage

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

// хранилище
type PostgresStorage struct {
	db *sql.DB
}

// новое подключение к БД
func NewPostgresStorage(dsn string) (*PostgresStorage, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open DB: %w", err)
	}

	// проверка соединения
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping DB: %w", err)
	}

	// Проверяем статус миграций перед применением
	currentVersion, err := goose.GetDBVersion(db)
	if err != nil {
		return nil, err
	}

	// Применяем миграции
	err = goose.Up(db, "migrations")
	if err != nil {
		return nil, err
	}

	// Логируем результат
	newVersion, _ := goose.GetDBVersion(db)
	if currentVersion != newVersion {
		log.Printf("Migrations apply: from %d to %d", currentVersion, newVersion)
	}

	return &PostgresStorage{db: db}, nil
}

// проверка соединения с БД
func (ps *PostgresStorage) Ping(ctx context.Context) error {
	return ps.db.PingContext(ctx)
}

// закрывает соединение с БД
func (ps *PostgresStorage) Close() error {
	return ps.db.Close()
}
