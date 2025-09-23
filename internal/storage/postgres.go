package storage

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/shatrunoff/yap_metrics/internal/model"
	"github.com/shatrunoff/yap_metrics/internal/utils"
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

	// Настройка пула соединений
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(30 * time.Minute)

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

func (ps *PostgresStorage) UpdateGauge(ctx context.Context, name string, value float64) error {
	// Retry на уровне всей функции
	return utils.RetrySendWithArgs(ctx, "UpdateGauge", func(ctx context.Context, args struct {
		Name  string
		Value float64
	}) error {
		// Вся логика функции внутри retry
		updateQuery := `UPDATE gauges SET value = $1 WHERE name = $2`
		result, err := ps.db.ExecContext(ctx, updateQuery, args.Value, args.Name)
		if err != nil {
			return fmt.Errorf("failed to update gauge %s: %w", args.Name, err)
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("failed to get rows affected for gauge %s: %w", args.Name, err)
		}

		if rowsAffected == 0 {
			insertQuery := `INSERT INTO gauges (name, value) VALUES ($1, $2)`
			_, err := ps.db.ExecContext(ctx, insertQuery, args.Name, args.Value)
			if err != nil {
				return fmt.Errorf("failed to insert gauge %s: %w", args.Name, err)
			}
		}

		return nil
	}, struct {
		Name  string
		Value float64
	}{Name: name, Value: value})
}

func (ps *PostgresStorage) UpdateCounter(ctx context.Context, name string, delta int64) error {
	// Retry на уровне функции операции
	return utils.RetrySendWithArgs(ctx, "UpdateCounter", func(ctx context.Context, args struct {
		Name  string
		Delta int64
	}) error {
		// Вся логика функции внутри retry
		updateQuery := `UPDATE counters SET value = value + $1 WHERE name = $2`
		result, err := ps.db.ExecContext(ctx, updateQuery, args.Delta, args.Name)
		if err != nil {
			return fmt.Errorf("failed to update counter %s: %w", args.Name, err)
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("failed to get rows affected for counter %s: %w", args.Name, err)
		}

		if rowsAffected == 0 {
			insertQuery := `INSERT INTO counters (name, value) VALUES ($1, $2)`
			_, err := ps.db.ExecContext(ctx, insertQuery, args.Name, args.Delta)
			if err != nil {
				return fmt.Errorf("failed to insert counter %s: %w", args.Name, err)
			}
		}

		return nil
	}, struct {
		Name  string
		Delta int64
	}{Name: name, Delta: delta})
}

// получение метрики из БД
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

// получение gauge из БД с retry
func (ps *PostgresStorage) getGauge(ctx context.Context, name string) (model.Metrics, error) {
	var result model.Metrics
	var lastErr error

	// Retry на уровне всей операции получения
	err := utils.RetrySendWithArgs(ctx, "getGauge", func(ctx context.Context, name string) error {
		query := `SELECT name, value FROM gauges WHERE name = $1`

		var metricName string
		var value float64

		err := ps.db.QueryRowContext(ctx, query, name).Scan(&metricName, &value)
		if err != nil {
			if err == sql.ErrNoRows {
				lastErr = fmt.Errorf("gauge %s not found", name)
				return lastErr
			}
			lastErr = fmt.Errorf("failed to get gauge %s: %w", name, err)
			return lastErr
		}

		result = model.Metrics{
			ID:    metricName,
			MType: model.Gauge,
			Value: &value,
		}
		lastErr = nil
		return nil
	}, name)

	if err != nil {
		return model.Metrics{}, lastErr
	}

	return result, nil
}

// получение counter из БД с retry
func (ps *PostgresStorage) getCounter(ctx context.Context, name string) (model.Metrics, error) {
	var result model.Metrics
	var lastErr error

	// Retry на уровне всей операции получения
	err := utils.RetrySendWithArgs(ctx, "getCounter", func(ctx context.Context, name string) error {
		query := `SELECT name, value FROM counters WHERE name = $1`

		var metricName string
		var value int64

		err := ps.db.QueryRowContext(ctx, query, name).Scan(&metricName, &value)
		if err != nil {
			if err == sql.ErrNoRows {
				lastErr = fmt.Errorf("counter %s not found", name)
				return lastErr
			}
			lastErr = fmt.Errorf("failed to get counter %s: %w", name, err)
			return lastErr
		}

		result = model.Metrics{
			ID:    metricName,
			MType: model.Counter,
			Delta: &value,
		}
		lastErr = nil
		return nil
	}, name)

	if err != nil {
		return model.Metrics{}, lastErr
	}

	return result, nil
}

// получение всех метрик из БД
func (ps *PostgresStorage) GetAll(ctx context.Context) (map[string]model.Metrics, error) {
	metrics := make(map[string]model.Metrics)

	gauges, err := ps.getAllGauges(ctx)
	if err != nil {
		return nil, err
	}
	for k, v := range gauges {
		metrics[k] = v
	}

	counters, err := ps.getAllCounters(ctx)
	if err != nil {
		return nil, err
	}
	for k, v := range counters {
		metrics[k] = v
	}

	return metrics, nil
}

// получение всех gauge из БД с retry
func (ps *PostgresStorage) getAllGauges(ctx context.Context) (map[string]model.Metrics, error) {
	var result map[string]model.Metrics

	// Используем пустую структуру как аргумент
	err := utils.RetrySendWithArgs(ctx, "getAllGauges", func(ctx context.Context, _ struct{}) error {
		query := `SELECT name, value FROM gauges`

		rows, err := ps.db.QueryContext(ctx, query)
		if err != nil {
			return fmt.Errorf("failed to query gauges: %w", err)
		}
		defer rows.Close()

		metrics := make(map[string]model.Metrics)
		for rows.Next() {
			var name string
			var value float64

			if err := rows.Scan(&name, &value); err != nil {
				return fmt.Errorf("failed to scan gauge: %w", err)
			}

			v := value
			metrics[name] = model.Metrics{
				ID:    name,
				MType: model.Gauge,
				Value: &v,
			}
		}

		if err := rows.Err(); err != nil {
			return fmt.Errorf("error iterating gauges: %w", err)
		}

		result = metrics
		return nil
	}, struct{}{})

	if err != nil {
		return nil, err
	}

	return result, nil
}

// получение всех counter из БД с retry
func (ps *PostgresStorage) getAllCounters(ctx context.Context) (map[string]model.Metrics, error) {
	var result map[string]model.Metrics

	err := utils.RetrySendWithArgs(ctx, "getAllCounters", func(ctx context.Context, _ struct{}) error {
		query := `SELECT name, value FROM counters`

		rows, err := ps.db.QueryContext(ctx, query)
		if err != nil {
			return fmt.Errorf("failed to query counters: %w", err)
		}
		defer rows.Close()

		metrics := make(map[string]model.Metrics)
		for rows.Next() {
			var name string
			var value int64

			if err := rows.Scan(&name, &value); err != nil {
				return fmt.Errorf("failed to scan counter: %w", err)
			}

			v := value
			metrics[name] = model.Metrics{
				ID:    name,
				MType: model.Counter,
				Delta: &v,
			}
		}

		if err := rows.Err(); err != nil {
			return fmt.Errorf("error iterating counters: %w", err)
		}

		result = metrics
		return nil
	}, struct{}{})

	if err != nil {
		return nil, err
	}

	return result, nil
}

// Методы для совместимости с файловым интерфейсом (не реализованы для PostgreSQL)
func (ps *PostgresStorage) SaveToFile(path string) error {
	return nil
}
func (ps *PostgresStorage) LoadFromFile(filename string) error {
	return nil
}

// обновление метрик в БД батчами
func (ps *PostgresStorage) UpdateMetricsBatch(ctx context.Context, metrics []model.Metrics) error {
	if len(metrics) == 0 {
		return nil
	}

	log.Printf("Processing batch of %d metrics", len(metrics))

	// Обрабатываем каждую метрику отдельно
	for i, metric := range metrics {
		if metric.ID == "" {
			continue
		}

		log.Printf("Processing metric %d: %s (%s)", i+1, metric.ID, metric.MType)

		var err error
		switch metric.MType {
		case model.Gauge:
			if metric.Value != nil {
				log.Printf("Updating gauge %s with value %f", metric.ID, *metric.Value)
				err = ps.UpdateGauge(ctx, metric.ID, *metric.Value)
			}

		case model.Counter:
			if metric.Delta != nil {
				log.Printf("Updating counter %s with delta %d", metric.ID, *metric.Delta)
				err = ps.UpdateCounter(ctx, metric.ID, *metric.Delta)
			}

		default:
			log.Printf("Skipping unknown metric type: %s", metric.MType)
			continue
		}

		if err != nil {
			log.Printf("ERROR updating metric %s: %v", metric.ID, err)
		}
	}

	log.Printf("Batch processing completed")
	return nil
}
