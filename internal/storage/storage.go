package storage

import (
	"context"
	"fmt"

	"github.com/shatrunoff/yap_metrics/internal/model"
	"github.com/shatrunoff/yap_metrics/internal/repository"
)

// интерфейс для всех типов хранилищ
type Storage interface {
	Pinger
	Updater
	Getter
	Saver
	Loader
}

// интерфейс для проверки соединения
type Pinger interface {
	Ping(ctx context.Context) error
}

// интерфейс для обновления метрик
type Updater interface {
	UpdateGauge(ctx context.Context, name string, value float64) error
	UpdateCounter(ctx context.Context, name string, delta int64) error
}

// интерфейс для получения метрик
type Getter interface {
	GetMetric(ctx context.Context, metricType, name string) (model.Metrics, error)
	GetAll(ctx context.Context) (map[string]model.Metrics, error)
}

// интерфейс для сохранения в файл
type Saver interface {
	SaveToFile(path string) error
}

// интерфейс для загрузки из файла
type Loader interface {
	LoadFromFile(filename string) error
}

// тип хранилища
type StorageType int

const (
	StorageTypeMemory StorageType = iota
	StorageTypeFile
	StorageTypePostgres
)

// создает соответствующее хранилище на основе конфигурации
func NewStorage(cfg *Config) (Storage, error) {
	if cfg.DatabaseDSN != "" {
		pgStorage, err := NewPostgresStorage(cfg.DatabaseDSN)
		if err != nil {
			return nil, fmt.Errorf("failed to create PostgreSQL storage: %w", err)
		}

		// Запускаем миграции с путем по умолчанию
		migrator := repository.NewMigratorWithDefaultPath(pgStorage.db)
		if err := migrator.RunMigrations(); err != nil {
			pgStorage.Close()
			return nil, fmt.Errorf("failed to run migrations: %w", err)
		}

		return pgStorage, nil
	}

	if cfg.FileStoragePath != "" {
		memStorage := NewMemStorage()
		if cfg.Restore {
			if err := memStorage.LoadFromFile(cfg.FileStoragePath); err != nil {
				return nil, fmt.Errorf("failed to load from file: %w", err)
			}
		}
		return memStorage, nil
	}

	return NewMemStorage(), nil
}

// конфигурация хранилища
type Config struct {
	DatabaseDSN     string
	FileStoragePath string
	Restore         bool
}
