package storage

import (
	"maps"
	"sync"

	"github.com/shatrunoff/yap_metrics/internal/model"
)

type MemStorage struct {
	metrics map[string]model.Metrics
	mu      sync.RWMutex
}

func NewMemStorage() *MemStorage {
	return &MemStorage{
		metrics: make(map[string]model.Metrics),
	}
}

// обнновление метрики
func (m *MemStorage) UpdateGauge(name string, value float64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.metrics[name] = model.Metrics{
		ID:    name,
		MType: model.Gauge,
		Value: &value,
	}
}

// обнновление счетчика
func (m *MemStorage) UpdateCounter(name string, delta int64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	exist, ok := m.metrics[name]
	if ok && exist.Delta != nil {
		*exist.Delta += delta
		m.metrics[name] = exist
	} else {
		m.metrics[name] = model.Metrics{
			ID:    name,
			MType: model.Counter,
			Delta: &delta,
		}
	}
}

// получение 1й метрики
func (m *MemStorage) GetMetric(metricType, name string) (model.Metrics, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	metric, ok := m.metrics[name]
	if !ok || metric.MType != metricType {
		return model.Metrics{}, false
	}
	return metric, true
}

// получение всех метрик
func (m *MemStorage) GetAll() map[string]model.Metrics {
	m.mu.RLock()
	defer m.mu.RUnlock()

	res := make(map[string]model.Metrics, len(m.metrics))
	maps.Copy(res, m.metrics)
	return res
}
