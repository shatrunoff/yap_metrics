package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type Pinger interface {
	Ping(ctx context.Context) error
}

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

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping DB: %w", err)
	}

	// if err := createTables(db); err != nil {
	// 	db.Close()
	// 	return nil, fmt.Errorf("failed to create tables: %w", err)
	// }

	return &PostgresStorage{db: db}, nil
}

// func createTables(db *sql.DB) error {
// 	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
// 	defer cancel()

// 	_, err := db.ExecContext(ctx, `
// 		CREATE TABLE IF NOT EXISTS gauges (
// 			id SERIAL PRIMARY KEY,
// 			name VARCHAR(255) NOT NULL UNIQUE,
// 			value DOUBLE PRECISION NOT NULL,
// 			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
// 		)
// 	`)
// 	if err != nil {
// 		return fmt.Errorf("failed to create gauges table: %w", err)
// 	}

// 	_, err = db.ExecContext(ctx, `
// 		CREATE TABLE IF NOT EXISTS counters (
// 			id SERIAL PRIMARY KEY,
// 			name VARCHAR(255) NOT NULL UNIQUE,
// 			value BIGINT NOT NULL,
// 			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
// 		)
// 	`)
// 	if err != nil {
// 		return fmt.Errorf("failed to create counters table: %w", err)
// 	}

//		return nil
//	}
//
// проверка соединения с БД
func (ps *PostgresStorage) Ping(ctx context.Context) error {
	return ps.db.PingContext(ctx)
}

// закрывает соединение с БДs
func (ps *PostgresStorage) Close() error {
	return ps.db.Close()
}
