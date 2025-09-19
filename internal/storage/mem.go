package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"github.com/shatrunoff/yap_metrics/internal/model"
)

type MemStorage struct {
	gauges   map[string]float64
	counters map[string]int64
	mu       sync.RWMutex
}

func NewMemStorage() *MemStorage {
	return &MemStorage{
		gauges:   make(map[string]float64),
		counters: make(map[string]int64),
	}
}

func (ms *MemStorage) UpdateGauge(name string, value float64) {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	ms.gauges[name] = value
}

func (ms *MemStorage) UpdateCounter(name string, delta int64) {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	ms.counters[name] += delta
}

func (ms *MemStorage) GetMetric(metricType, name string) (model.Metrics, bool) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	switch metricType {
	case model.Gauge:
		if value, ok := ms.gauges[name]; ok {
			return model.Metrics{
				ID:    name,
				MType: model.Gauge,
				Value: &value,
			}, true
		}
	case model.Counter:
		if value, ok := ms.counters[name]; ok {
			return model.Metrics{
				ID:    name,
				MType: model.Counter,
				Delta: &value,
			}, true
		}
	}
	return model.Metrics{}, false
}

func (ms *MemStorage) GetAll() map[string]model.Metrics {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	result := make(map[string]model.Metrics)

	for name, value := range ms.gauges {
		v := value
		result[name] = model.Metrics{
			ID:    name,
			MType: model.Gauge,
			Value: &v,
		}
	}

	for name, value := range ms.counters {
		v := value
		result[name] = model.Metrics{
			ID:    name,
			MType: model.Counter,
			Delta: &v,
		}
	}

	return result
}

func (ms *MemStorage) SaveToFile(filePath string) error {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	data := struct {
		Gauges   map[string]float64 `json:"gauges"`
		Counters map[string]int64   `json:"counters"`
	}{
		Gauges:   ms.gauges,
		Counters: ms.counters,
	}

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal metrics: %w", err)
	}

	if err := os.WriteFile(filePath, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

func (ms *MemStorage) LoadFromFile(filePath string) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	var fileData struct {
		Gauges   map[string]float64 `json:"gauges"`
		Counters map[string]int64   `json:"counters"`
	}

	if err := json.Unmarshal(data, &fileData); err != nil {
		return fmt.Errorf("failed to unmarshal metrics: %w", err)
	}

	ms.gauges = fileData.Gauges
	ms.counters = fileData.Counters

	return nil
}
