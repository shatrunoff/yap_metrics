package storage

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/shatrunoff/yap_metrics/internal/model"
)

func TestMemStorage_UpdateGauge(t *testing.T) {
	ms := NewMemStorage()
	ctx := context.Background()

	err := ms.UpdateGauge(ctx, "temperature", 25.5)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	metric, err := ms.GetMetric(ctx, model.Gauge, "temperature")
	if err != nil {
		t.Errorf("failed to get metric: %v", err)
	}

	if metric.Value == nil || *metric.Value != 25.5 {
		t.Errorf("expected value 25.5, got %v", metric.Value)
	}
}

func TestMemStorage_UpdateCounter(t *testing.T) {
	ms := NewMemStorage()
	ctx := context.Background()

	err := ms.UpdateCounter(ctx, "requests", 10)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	err = ms.UpdateCounter(ctx, "requests", 5)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	metric, err := ms.GetMetric(ctx, model.Counter, "requests")
	if err != nil {
		t.Errorf("failed to get metric: %v", err)
	}

	if metric.Delta == nil || *metric.Delta != 15 {
		t.Errorf("expected delta 15, got %v", metric.Delta)
	}
}

func TestMemStorage_GetMetric(t *testing.T) {
	ms := NewMemStorage()
	ctx := context.Background()

	ms.UpdateGauge(ctx, "cpu", 75.5)
	ms.UpdateCounter(ctx, "requests", 100)

	tests := []struct {
		name        string
		metricType  string
		metricName  string
		expectError bool
	}{
		{
			name:        "get gauge",
			metricType:  model.Gauge,
			metricName:  "cpu",
			expectError: false,
		},
		{
			name:        "get counter",
			metricType:  model.Counter,
			metricName:  "requests",
			expectError: false,
		},
		{
			name:        "get non-existing gauge",
			metricType:  model.Gauge,
			metricName:  "nonexistent",
			expectError: true,
		},
		{
			name:        "get non-existing counter",
			metricType:  model.Counter,
			metricName:  "nonexistent",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ms.GetMetric(ctx, tt.metricType, tt.metricName)
			if tt.expectError && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestMemStorage_GetAll(t *testing.T) {
	ms := NewMemStorage()
	ctx := context.Background()

	ms.UpdateGauge(ctx, "cpu", 75.5)
	ms.UpdateGauge(ctx, "memory", 80.0)
	ms.UpdateCounter(ctx, "requests", 100)

	metrics, err := ms.GetAll(ctx)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if len(metrics) != 3 {
		t.Errorf("expected 3 metrics, got %d", len(metrics))
	}

	if _, ok := metrics["cpu"]; !ok {
		t.Error("expected cpu metric")
	}
	if _, ok := metrics["memory"]; !ok {
		t.Error("expected memory metric")
	}
	if _, ok := metrics["requests"]; !ok {
		t.Error("expected requests metric")
	}
}

func TestMemStorage_SaveToFile(t *testing.T) {
	ms := NewMemStorage()
	ctx := context.Background()

	ms.UpdateGauge(ctx, "cpu", 75.5)
	ms.UpdateCounter(ctx, "requests", 100)

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "metrics.json")

	err := ms.SaveToFile(filePath)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Error("file was not created")
	}
}

func TestMemStorage_LoadFromFile(t *testing.T) {
	// Создаем хранилище с данными
	ms1 := NewMemStorage()
	ctx := context.Background()

	ms1.UpdateGauge(ctx, "cpu", 75.5)
	ms1.UpdateCounter(ctx, "requests", 100)

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "metrics.json")

	// Сохраняем в файл
	err := ms1.SaveToFile(filePath)
	if err != nil {
		t.Fatalf("failed to save to file: %v", err)
	}

	// Создаем новое хранилище и загружаем из файла
	ms2 := NewMemStorage()
	err = ms2.LoadFromFile(filePath)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Проверяем gauge
	metric, err := ms2.GetMetric(ctx, model.Gauge, "cpu")
	if err != nil {
		t.Errorf("failed to get gauge: %v", err)
	}
	if metric.Value == nil || *metric.Value != 75.5 {
		t.Errorf("expected value 75.5, got %v", metric.Value)
	}

	// Проверяем counter
	metric, err = ms2.GetMetric(ctx, model.Counter, "requests")
	if err != nil {
		t.Errorf("failed to get counter: %v", err)
	}
	if metric.Delta == nil || *metric.Delta != 100 {
		t.Errorf("expected delta 100, got %v", metric.Delta)
	}
}

func TestMemStorage_LoadFromFile_NonExistent(t *testing.T) {
	ms := NewMemStorage()
	err := ms.LoadFromFile("/nonexistent/path/to/file.json")
	if err != nil {
		t.Errorf("expected no error for non-existent file, got %v", err)
	}
}

func TestMemStorage_UpdateMetricsBatch(t *testing.T) {
	ms := NewMemStorage()
	ctx := context.Background()

	value1 := 75.5
	value2 := 80.0
	delta := int64(100)

	metrics := []model.Metrics{
		{
			ID:    "cpu",
			MType: model.Gauge,
			Value: &value1,
		},
		{
			ID:    "memory",
			MType: model.Gauge,
			Value: &value2,
		},
		{
			ID:    "requests",
			MType: model.Counter,
			Delta: &delta,
		},
	}

	err := ms.UpdateMetricsBatch(ctx, metrics)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Проверяем gauge
	metric, err := ms.GetMetric(ctx, model.Gauge, "cpu")
	if err != nil {
		t.Errorf("failed to get cpu: %v", err)
	}
	if metric.Value == nil || *metric.Value != 75.5 {
		t.Errorf("expected cpu value 75.5, got %v", metric.Value)
	}

	// Проверяем counter
	metric, err = ms.GetMetric(ctx, model.Counter, "requests")
	if err != nil {
		t.Errorf("failed to get requests: %v", err)
	}
	if metric.Delta == nil || *metric.Delta != 100 {
		t.Errorf("expected requests delta 100, got %v", metric.Delta)
	}
}

func TestMemStorage_UpdateMetricsBatch_Empty(t *testing.T) {
	ms := NewMemStorage()
	ctx := context.Background()

	err := ms.UpdateMetricsBatch(ctx, []model.Metrics{})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestMemStorage_UpdateMetricsBatch_EmptyID(t *testing.T) {
	ms := NewMemStorage()
	ctx := context.Background()

	value := 75.5
	metrics := []model.Metrics{
		{
			ID:    "",
			MType: model.Gauge,
			Value: &value,
		},
	}

	err := ms.UpdateMetricsBatch(ctx, metrics)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Не должно быть метрик с пустым ID
	allMetrics, _ := ms.GetAll(ctx)
	if len(allMetrics) != 0 {
		t.Errorf("expected 0 metrics, got %d", len(allMetrics))
	}
}

func TestMemStorage_UpdateMetricsBatch_NilValues(t *testing.T) {
	ms := NewMemStorage()
	ctx := context.Background()

	metrics := []model.Metrics{
		{
			ID:    "cpu",
			MType: model.Gauge,
			Value: nil,
		},
		{
			ID:    "requests",
			MType: model.Counter,
			Delta: nil,
		},
	}

	err := ms.UpdateMetricsBatch(ctx, metrics)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Метрики с nil значениями не должны быть добавлены
	allMetrics, _ := ms.GetAll(ctx)
	if len(allMetrics) != 0 {
		t.Errorf("expected 0 metrics, got %d", len(allMetrics))
	}
}

func TestMemStorage_Ping(t *testing.T) {
	ms := NewMemStorage()
	ctx := context.Background()

	err := ms.Ping(ctx)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestMemStorage_Close(t *testing.T) {
	ms := NewMemStorage()
	err := ms.Close()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}
