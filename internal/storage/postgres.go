package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/shatrunoff/yap_metrics/internal/model"
)

type PostgresStorage struct {
	db *sql.DB
}

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

	// Устанавливаем настройки пула соединений
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(5 * time.Minute)

	return &PostgresStorage{db: db}, nil
}

func (ps *PostgresStorage) Ping(ctx context.Context) error {
	return ps.db.PingContext(ctx)
}

func (ps *PostgresStorage) Close() error {
	return ps.db.Close()
}

func (ps *PostgresStorage) UpdateGauge(ctx context.Context, name string, value float64) error {
	query := `
		INSERT INTO gauges (name, value) 
		VALUES ($1, $2)
		ON CONFLICT (name) 
		DO UPDATE SET value = EXCLUDED.value, updated_at = CURRENT_TIMESTAMP
	`

	_, err := ps.db.ExecContext(ctx, query, name, value)
	if err != nil {
		return fmt.Errorf("failed to update gauge %s: %w", name, err)
	}
	return nil
}

func (ps *PostgresStorage) UpdateCounter(ctx context.Context, name string, delta int64) error {
	query := `
		INSERT INTO counters (name, value) 
		VALUES ($1, $2)
		ON CONFLICT (name) 
		DO UPDATE SET value = counters.value + EXCLUDED.value, updated_at = CURRENT_TIMESTAMP
	`

	_, err := ps.db.ExecContext(ctx, query, name, delta)
	if err != nil {
		return fmt.Errorf("failed to update counter %s: %w", name, err)
	}
	return nil
}

func (ps *PostgresStorage) GetMetric(ctx context.Context, metricType, name string) (model.Metrics, error) {
	switch metricType {
	case model.Gauge:
		return ps.getGauge(ctx, name)
	case model.Counter:
		return ps.getCounter(ctx, name)
	default:
		return model.Metrics{}, fmt.Errorf("unknown metric type: %s", metricType)
	}
}

func (ps *PostgresStorage) getGauge(ctx context.Context, name string) (model.Metrics, error) {
	query := `SELECT name, value FROM gauges WHERE name = $1`

	var metricName string
	var value float64

	err := ps.db.QueryRowContext(ctx, query, name).Scan(&metricName, &value)
	if err != nil {
		if err == sql.ErrNoRows {
			return model.Metrics{}, fmt.Errorf("gauge %s not found", name)
		}
		return model.Metrics{}, fmt.Errorf("failed to get gauge %s: %w", name, err)
	}

	return model.Metrics{
		ID:    metricName,
		MType: model.Gauge,
		Value: &value,
	}, nil
}

func (ps *PostgresStorage) getCounter(ctx context.Context, name string) (model.Metrics, error) {
	query := `SELECT name, value FROM counters WHERE name = $1`

	var metricName string
	var value int64

	err := ps.db.QueryRowContext(ctx, query, name).Scan(&metricName, &value)
	if err != nil {
		if err == sql.ErrNoRows {
			return model.Metrics{}, fmt.Errorf("counter %s not found", name)
		}
		return model.Metrics{}, fmt.Errorf("failed to get counter %s: %w", name, err)
	}

	return model.Metrics{
		ID:    metricName,
		MType: model.Counter,
		Delta: &value,
	}, nil
}

func (ps *PostgresStorage) GetAll(ctx context.Context) (map[string]model.Metrics, error) {
	metrics := make(map[string]model.Metrics)

	// Получаем все gauge метрики
	gauges, err := ps.getAllGauges(ctx)
	if err != nil {
		return nil, err
	}
	for k, v := range gauges {
		metrics[k] = v
	}

	// Получаем все counter метрики
	counters, err := ps.getAllCounters(ctx)
	if err != nil {
		return nil, err
	}
	for k, v := range counters {
		metrics[k] = v
	}

	return metrics, nil
}

func (ps *PostgresStorage) getAllGauges(ctx context.Context) (map[string]model.Metrics, error) {
	query := `SELECT name, value FROM gauges`

	rows, err := ps.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query gauges: %w", err)
	}
	defer rows.Close()

	metrics := make(map[string]model.Metrics)
	for rows.Next() {
		var name string
		var value float64

		if err := rows.Scan(&name, &value); err != nil {
			return nil, fmt.Errorf("failed to scan gauge: %w", err)
		}

		v := value
		metrics[name] = model.Metrics{
			ID:    name,
			MType: model.Gauge,
			Value: &v,
		}
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating gauges: %w", err)
	}

	return metrics, nil
}

func (ps *PostgresStorage) getAllCounters(ctx context.Context) (map[string]model.Metrics, error) {
	query := `SELECT name, value FROM counters`

	rows, err := ps.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query counters: %w", err)
	}
	defer rows.Close()

	metrics := make(map[string]model.Metrics)
	for rows.Next() {
		var name string
		var value int64

		if err := rows.Scan(&name, &value); err != nil {
			return nil, fmt.Errorf("failed to scan counter: %w", err)
		}

		v := value
		metrics[name] = model.Metrics{
			ID:    name,
			MType: model.Counter,
			Delta: &v,
		}
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating counters: %w", err)
	}

	return metrics, nil
}

// Методы для совместимости с файловым хранилищем
func (ps *PostgresStorage) SaveToFile(path string) error {
	// Для PostgreSQL сохранение в файл не требуется
	return nil
}

// Для PostgreSQL загрузка из файла не требуется
func (ps *PostgresStorage) LoadFromFile(filename string) error {
	return nil
}
