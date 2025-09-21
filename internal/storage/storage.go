package storage

import (
	"context"

	"github.com/shatrunoff/yap_metrics/internal/model"
)

// Storage универсальный интерфейс для всех типов хранилищ
type Storage interface {
	Ping(ctx context.Context) error
	UpdateGauge(ctx context.Context, name string, value float64) error
	UpdateCounter(ctx context.Context, name string, delta int64) error
	UpdateMetricsBatch(ctx context.Context, metrics []model.Metrics) error
	GetMetric(ctx context.Context, metricType, name string) (model.Metrics, error)
	GetAll(ctx context.Context) (map[string]model.Metrics, error)
	Close() error
}

// FileSaver интерфейс для хранилищ, поддерживающих файловые операции
type FileSaver interface {
	SaveToFile(path string) error
	LoadFromFile(filename string) error
}

// Config конфигурация хранилища
type Config struct {
	DatabaseDSN     string
	FileStoragePath string
	Restore         bool
}

// NewStorage создает хранилище с приоритетом:
// PostgreSQL -> файл -> память
func NewStorage(cfg *Config) (Storage, error) {
	// Приоритет 1: PostgreSQL
	if cfg.DatabaseDSN != "" {
		pgStorage, err := NewPostgresStorage(cfg.DatabaseDSN)
		if err != nil {
			return nil, err
		}
		return pgStorage, nil
	}

	// Приоритет 2: Файловое хранилище
	if cfg.FileStoragePath != "" {
		memStorage := NewMemStorage()
		if cfg.Restore {
			if err := memStorage.LoadFromFile(cfg.FileStoragePath); err != nil {
				return nil, err
			}
		}
		return memStorage, nil
	}

	// Приоритет 3: хранилище в памяти
	return NewMemStorage(), nil
}
