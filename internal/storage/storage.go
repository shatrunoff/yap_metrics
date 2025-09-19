package storage

import (
	"context"

	"github.com/shatrunoff/yap_metrics/internal/model"
)

// интерфейс для всех типов хранилищ
type Storage interface {
	Pinger
	Updater
	Getter
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
