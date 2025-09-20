package storage

import (
	"context"
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

func (ms *MemStorage) Ping(ctx context.Context) error {
	return nil
}

func (ms *MemStorage) Close() error {
	return nil
}

func (ms *MemStorage) UpdateGauge(ctx context.Context, name string, value float64) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	ms.gauges[name] = value
	return nil
}

func (ms *MemStorage) UpdateCounter(ctx context.Context, name string, delta int64) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	ms.counters[name] += delta
	return nil
}

func (ms *MemStorage) GetMetric(ctx context.Context, metricType, name string) (model.Metrics, error) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	switch metricType {
	case model.Gauge:
		if value, ok := ms.gauges[name]; ok {
			return model.Metrics{
				ID:    name,
				MType: model.Gauge,
				Value: &value,
			}, nil
		}
	case model.Counter:
		if value, ok := ms.counters[name]; ok {
			return model.Metrics{
				ID:    name,
				MType: model.Counter,
				Delta: &value,
			}, nil
		}
	}
	return model.Metrics{}, fmt.Errorf("metric %s not found", name)
}

func (ms *MemStorage) GetAll(ctx context.Context) (map[string]model.Metrics, error) {
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

	return result, nil
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
		if os.IsNotExist(err) {
			return nil
		}
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
