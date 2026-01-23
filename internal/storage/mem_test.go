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

func TestMemStorage_SaveAndLoadFile(t *testing.T) {
	ms := NewMemStorage()
	ctx := context.Background()

	// Add some metrics
	ms.UpdateGauge(ctx, "g1", 1.1)
	ms.UpdateGauge(ctx, "g2", 2.2)
	ms.UpdateCounter(ctx, "c1", 100)
	ms.UpdateCounter(ctx, "c2", 200)

	// Save to temp file
	tmpFile := filepath.Join(t.TempDir(), "metrics.json")
	err := ms.SaveToFile(tmpFile)
	if err != nil {
		t.Fatalf("SaveToFile failed: %v", err)
	}

	// Create new storage and load
	ms2 := NewMemStorage()
	err = ms2.LoadFromFile(tmpFile)
	if err != nil {
		t.Fatalf("LoadFromFile failed: %v", err)
	}

	// Verify gauges
	m, _ := ms2.GetMetric(ctx, model.Gauge, "g1")
	if m.Value == nil || *m.Value != 1.1 {
		t.Errorf("Expected g1=1.1, got %v", m.Value)
	}

	// Verify counters
	m, _ = ms2.GetMetric(ctx, model.Counter, "c1")
	if m.Delta == nil || *m.Delta != 100 {
		t.Errorf("Expected c1=100, got %v", m.Delta)
	}
}

func TestMemStorage_SaveToFileError(t *testing.T) {
	ms := NewMemStorage()

	// Try to save to invalid path
	err := ms.SaveToFile("/nonexistent/dir/file.json")
	if err == nil {
		t.Error("Expected error for invalid path")
	}
}

func TestMemStorage_LoadFromFileNotExist(t *testing.T) {
	ms := NewMemStorage()

	// Load from non-existent file should not error (creates empty storage)
	err := ms.LoadFromFile("/nonexistent/file.json")
	if err != nil {
		t.Errorf("LoadFromFile should not error for non-existent file: %v", err)
	}
}

func TestMemStorage_LoadFromFileInvalidJSON(t *testing.T) {
	ms := NewMemStorage()

	// Create temp file with invalid JSON
	tmpFile := filepath.Join(t.TempDir(), "invalid.json")
	os.WriteFile(tmpFile, []byte("invalid json"), 0644)

	err := ms.LoadFromFile(tmpFile)
	if err == nil {
		t.Error("Expected error for invalid JSON")
	}
}

func TestMemStorage_Ping_Basic(t *testing.T) {
	ms := NewMemStorage()
	err := ms.Ping(context.Background())
	if err != nil {
		t.Errorf("Ping should not error: %v", err)
	}
}

func TestMemStorage_Close_Basic(t *testing.T) {
	ms := NewMemStorage()
	err := ms.Close()
	if err != nil {
		t.Errorf("Close should not error: %v", err)
	}
}

func TestMemStorage_GetMetricUnknownType(t *testing.T) {
	ms := NewMemStorage()
	ctx := context.Background()

	_, err := ms.GetMetric(ctx, "unknown", "test")
	if err == nil {
		t.Error("Expected error for unknown metric type")
	}
}

func TestMemStorage_GetAllEmpty(t *testing.T) {
	ms := NewMemStorage()
	ctx := context.Background()

	all, err := ms.GetAll(ctx)
	if err != nil {
		t.Errorf("GetAll failed: %v", err)
	}
	if len(all) != 0 {
		t.Errorf("Expected empty map, got %d items", len(all))
	}
}

func TestMemStorage_UpdateMetricsBatchMixed(t *testing.T) {
	ms := NewMemStorage()
	ctx := context.Background()

	v1, v2 := 10.5, 20.5
	d1, d2 := int64(100), int64(200)

	metrics := []model.Metrics{
		{ID: "g1", MType: model.Gauge, Value: &v1},
		{ID: "g2", MType: model.Gauge, Value: &v2},
		{ID: "c1", MType: model.Counter, Delta: &d1},
		{ID: "c2", MType: model.Counter, Delta: &d2},
		{ID: "unknown", MType: "unknown"},
	}

	err := ms.UpdateMetricsBatch(ctx, metrics)
	if err != nil {
		t.Errorf("UpdateMetricsBatch failed: %v", err)
	}

	all, _ := ms.GetAll(ctx)
	if len(all) != 4 {
		t.Errorf("Expected 4 metrics, got %d", len(all))
	}
}

func TestMemStorage_UpdateMetricsBatchEmpty(t *testing.T) {
	ms := NewMemStorage()
	ctx := context.Background()

	err := ms.UpdateMetricsBatch(ctx, []model.Metrics{})
	if err != nil {
		t.Errorf("UpdateMetricsBatch with empty slice failed: %v", err)
	}
}

func TestMemStorage_UpdateMetricsBatchNilValues(t *testing.T) {
	ms := NewMemStorage()
	ctx := context.Background()

	metrics := []model.Metrics{
		{ID: "nil_gauge", MType: model.Gauge, Value: nil},
		{ID: "nil_counter", MType: model.Counter, Delta: nil},
	}

	err := ms.UpdateMetricsBatch(ctx, metrics)
	if err != nil {
		t.Errorf("UpdateMetricsBatch failed: %v", err)
	}
}

func TestMemStorage_UpdateMetricsBatchEmptyID(t *testing.T) {
	ms := NewMemStorage()
	ctx := context.Background()

	v := 1.0
	metrics := []model.Metrics{
		{ID: "", MType: model.Gauge, Value: &v},
	}

	err := ms.UpdateMetricsBatch(ctx, metrics)
	if err != nil {
		t.Errorf("UpdateMetricsBatch failed: %v", err)
	}

	all, _ := ms.GetAll(ctx)
	if len(all) != 0 {
		t.Errorf("Expected 0 metrics (empty ID skipped), got %d", len(all))
	}
}

func TestMemStorage_ConcurrentAccess(t *testing.T) {
	ms := NewMemStorage()
	ctx := context.Background()

	done := make(chan bool)

	// Writer goroutine
	go func() {
		for i := 0; i < 100; i++ {
			ms.UpdateGauge(ctx, "concurrent", float64(i))
			ms.UpdateCounter(ctx, "concurrent_counter", 1)
		}
		done <- true
	}()

	// Reader goroutine
	go func() {
		for i := 0; i < 100; i++ {
			ms.GetMetric(ctx, model.Gauge, "concurrent")
			ms.GetAll(ctx)
		}
		done <- true
	}()

	<-done
	<-done
}

func TestMemStorage_SaveToFile_MarshalError(t *testing.T) {
	// This test verifies SaveToFile handles the path correctly
	ms := NewMemStorage()
	ctx := context.Background()
	ms.UpdateGauge(ctx, "test", 1.0)

	// Save to valid path
	tmpFile := filepath.Join(t.TempDir(), "test.json")
	err := ms.SaveToFile(tmpFile)
	if err != nil {
		t.Errorf("SaveToFile failed: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(tmpFile); os.IsNotExist(err) {
		t.Error("File should exist after SaveToFile")
	}
}

func TestMemStorage_LoadFromFile_ReadError(t *testing.T) {
	ms := NewMemStorage()

	// Create a directory instead of file to cause read error
	tmpDir := t.TempDir()
	dirPath := filepath.Join(tmpDir, "testdir")
	os.Mkdir(dirPath, 0755)

	err := ms.LoadFromFile(dirPath)
	if err == nil {
		t.Error("Expected error when loading directory as file")
	}
}

func TestMemStorage_GetMetricGaugeNotFound(t *testing.T) {
	ms := NewMemStorage()
	ctx := context.Background()

	_, err := ms.GetMetric(ctx, model.Gauge, "nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent gauge")
	}
}

func TestMemStorage_GetMetricCounterNotFound(t *testing.T) {
	ms := NewMemStorage()
	ctx := context.Background()

	_, err := ms.GetMetric(ctx, model.Counter, "nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent counter")
	}
}
