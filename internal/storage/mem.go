package storage

import (
	"encoding/json"
	"log"
	"maps"
	"os"
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

// Загрузка метрик из файла
func (m *MemStorage) LoadFromFile(filename string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	data, err := os.ReadFile(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	var metrics []model.Metrics
	if err := json.Unmarshal(data, &metrics); err != nil {
		return err
	}

	// Очищаем текущие метрики и загружаем новые
	m.metrics = make(map[string]model.Metrics)
	for _, metric := range metrics {
		m.metrics[metric.ID] = metric
	}

	return nil
}

// Сохранение метрик в файл
func (m *MemStorage) SaveToFile(filename string) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	log.Printf("Saving %d metrics to %s", len(m.metrics), filename)

	metrics := make([]model.Metrics, 0, len(m.metrics))
	for _, metric := range m.metrics {
		metrics = append(metrics, metric)
	}

	data, err := json.MarshalIndent(metrics, "", "  ")
	if err != nil {
		log.Printf("ERROR: failed to marshal metrics: %v", err)
		return err
	}

	err = os.WriteFile(filename, data, 0644)
	if err != nil {
		log.Printf("ERROR: failed to write file %s: %v", filename, err)
		return err
	}

	log.Printf("Successfully saved %d metrics to %s", len(metrics), filename)
	return nil
}

// обновление метрики
func (m *MemStorage) UpdateGauge(name string, value float64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.metrics[name] = model.Metrics{
		ID:    name,
		MType: model.Gauge,
		Value: &value,
	}
}

// обновление счетчика
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
