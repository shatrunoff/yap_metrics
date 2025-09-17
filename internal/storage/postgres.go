package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// хранилище
type PostgresStorage struct {
	db *sql.DB
}

// новое подключение к БД
func NewPostgresStorage(dsn string) (*PostgresStorage, error) {
	//
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
