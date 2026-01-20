package storage

import (
	"context"
	"os"
	"testing"

	"github.com/shatrunoff/yap_metrics/internal/model"
)

func TestNewStorage(t *testing.T) {
	tests := []struct {
		name        string
		config      *Config
		expectError bool
		storageType string
	}{
		{
			name: "memory storage - no config",
			config: &Config{
				DatabaseDSN:     "",
				FileStoragePath: "",
				Restore:         false,
			},
			expectError: false,
			storageType: "memory",
		},
		{
			name: "file storage - with path",
			config: &Config{
				DatabaseDSN:     "",
				FileStoragePath: "/tmp/test.json",
				Restore:         false,
			},
			expectError: false,
			storageType: "memory", // Still memory storage, but with file path
		},
		{
			name: "file storage - with restore from non-existent file",
			config: &Config{
				DatabaseDSN:     "",
				FileStoragePath: "/tmp/nonexistent.json",
				Restore:         true,
			},
			expectError: false, // Memory storage is created even if file doesn't exist
			storageType: "memory",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage, err := NewStorage(tt.config)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if storage == nil {
				t.Error("Expected storage instance but got nil")
				return
			}

			// Test basic storage operations
			ctx := context.Background()

			// Test ping
			if err := storage.Ping(ctx); err != nil {
				t.Errorf("Ping failed: %v", err)
			}

			// Test update gauge
			if err := storage.UpdateGauge(ctx, "test_gauge", 42.0); err != nil {
				t.Errorf("UpdateGauge failed: %v", err)
			}

			// Test update counter
			if err := storage.UpdateCounter(ctx, "test_counter", 10); err != nil {
				t.Errorf("UpdateCounter failed: %v", err)
			}

			// Test get metric
			metric, err := storage.GetMetric(ctx, "gauge", "test_gauge")
			if err != nil {
				t.Errorf("GetMetric failed: %v", err)
			}
			if metric.Value == nil || *metric.Value != 42.0 {
				t.Errorf("Expected gauge value 42.0, got %v", metric.Value)
			}

			// Test get all
			all, err := storage.GetAll(ctx)
			if err != nil {
				t.Errorf("GetAll failed: %v", err)
			}
			if len(all) < 2 {
				t.Errorf("Expected at least 2 metrics, got %d", len(all))
			}

			// Clean up
			if err := storage.Close(); err != nil {
				t.Errorf("Close failed: %v", err)
			}
		})
	}
}

func TestNewStorageWithValidFile(t *testing.T) {
	// Create a temporary file with valid JSON
	tmpFile, err := os.CreateTemp("", "metrics_*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	// Write valid metrics JSON
	validJSON := `{"gauges":{"test_gauge":123.45},"counters":{}}`
	if _, err := tmpFile.WriteString(validJSON); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	tmpFile.Close()

	config := &Config{
		DatabaseDSN:     "",
		FileStoragePath: tmpFile.Name(),
		Restore:         true,
	}

	storage, err := NewStorage(config)
	if err != nil {
		t.Fatalf("NewStorage failed: %v", err)
	}
	defer storage.Close()

	// Verify the metric was loaded
	ctx := context.Background()
	metric, err := storage.GetMetric(ctx, "gauge", "test_gauge")
	if err != nil {
		t.Errorf("GetMetric failed: %v", err)
	}
	if metric.Value == nil || *metric.Value != 123.45 {
		t.Errorf("Expected loaded value 123.45, got %v", metric.Value)
	}
}

func TestNewStorageWithInvalidDSN(t *testing.T) {
	config := &Config{
		DatabaseDSN: "invalid://dsn",
	}
	_, err := NewStorage(config)
	if err == nil {
		t.Error("Expected error for invalid DSN")
	}
}

func TestStorageUpdateMetricsBatch(t *testing.T) {
	storage, err := NewStorage(&Config{})
	if err != nil {
		t.Fatalf("NewStorage failed: %v", err)
	}
	defer storage.Close()

	ctx := context.Background()
	v1 := 10.5
	d1 := int64(100)

	metrics := []model.Metrics{
		{ID: "batch_gauge", MType: model.Gauge, Value: &v1},
		{ID: "batch_counter", MType: model.Counter, Delta: &d1},
	}

	err = storage.UpdateMetricsBatch(ctx, metrics)
	if err != nil {
		t.Errorf("UpdateMetricsBatch failed: %v", err)
	}

	// Verify gauge
	m, err := storage.GetMetric(ctx, model.Gauge, "batch_gauge")
	if err != nil {
		t.Errorf("GetMetric gauge failed: %v", err)
	}
	if m.Value == nil || *m.Value != 10.5 {
		t.Errorf("Expected 10.5, got %v", m.Value)
	}

	// Verify counter
	m, err = storage.GetMetric(ctx, model.Counter, "batch_counter")
	if err != nil {
		t.Errorf("GetMetric counter failed: %v", err)
	}
	if m.Delta == nil || *m.Delta != 100 {
		t.Errorf("Expected 100, got %v", m.Delta)
	}
}

func TestStorageCounterAccumulation(t *testing.T) {
	storage, err := NewStorage(&Config{})
	if err != nil {
		t.Fatalf("NewStorage failed: %v", err)
	}
	defer storage.Close()

	ctx := context.Background()

	// Add counter multiple times
	storage.UpdateCounter(ctx, "acc_counter", 10)
	storage.UpdateCounter(ctx, "acc_counter", 20)
	storage.UpdateCounter(ctx, "acc_counter", 30)

	m, err := storage.GetMetric(ctx, model.Counter, "acc_counter")
	if err != nil {
		t.Errorf("GetMetric failed: %v", err)
	}
	if m.Delta == nil || *m.Delta != 60 {
		t.Errorf("Expected 60, got %v", m.Delta)
	}
}

func TestStorageGaugeOverwrite(t *testing.T) {
	storage, err := NewStorage(&Config{})
	if err != nil {
		t.Fatalf("NewStorage failed: %v", err)
	}
	defer storage.Close()

	ctx := context.Background()

	storage.UpdateGauge(ctx, "ow_gauge", 10.0)
	storage.UpdateGauge(ctx, "ow_gauge", 20.0)
	storage.UpdateGauge(ctx, "ow_gauge", 30.0)

	m, err := storage.GetMetric(ctx, model.Gauge, "ow_gauge")
	if err != nil {
		t.Errorf("GetMetric failed: %v", err)
	}
	if m.Value == nil || *m.Value != 30.0 {
		t.Errorf("Expected 30.0, got %v", m.Value)
	}
}

func TestStorageGetMetricNotFound(t *testing.T) {
	storage, err := NewStorage(&Config{})
	if err != nil {
		t.Fatalf("NewStorage failed: %v", err)
	}
	defer storage.Close()

	ctx := context.Background()
	_, err = storage.GetMetric(ctx, model.Gauge, "nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent metric")
	}
}

func TestStorageGetMetricInvalidType(t *testing.T) {
	storage, err := NewStorage(&Config{})
	if err != nil {
		t.Fatalf("NewStorage failed: %v", err)
	}
	defer storage.Close()

	ctx := context.Background()
	_, err = storage.GetMetric(ctx, "invalid_type", "test")
	if err == nil {
		t.Error("Expected error for invalid metric type")
	}
}

func TestPostgresStorage_SaveLoadFile(t *testing.T) {
	// PostgresStorage doesn't support file operations
	ps := &PostgresStorage{}

	err := ps.SaveToFile("/tmp/test.json")
	if err != nil {
		t.Errorf("SaveToFile should return nil: %v", err)
	}

	err = ps.LoadFromFile("/tmp/test.json")
	if err != nil {
		t.Errorf("LoadFromFile should return nil: %v", err)
	}
}

func TestNewPostgresStorage_InvalidDSN(t *testing.T) {
	_, err := NewPostgresStorage("invalid://dsn")
	if err == nil {
		t.Error("Expected error for invalid DSN")
	}
}

func TestSplitMetricsByType(t *testing.T) {
	v1, v2 := 1.0, 2.0
	d1, d2 := int64(10), int64(20)

	metrics := []model.Metrics{
		{ID: "g1", MType: model.Gauge, Value: &v1},
		{ID: "g2", MType: model.Gauge, Value: &v2},
		{ID: "c1", MType: model.Counter, Delta: &d1},
		{ID: "c2", MType: model.Counter, Delta: &d2},
		{ID: "unknown", MType: "unknown"},
	}

	gauges, counters, unknown := splitMetricsByType(metrics)

	if len(gauges) != 2 {
		t.Errorf("Expected 2 gauges, got %d", len(gauges))
	}
	if len(counters) != 2 {
		t.Errorf("Expected 2 counters, got %d", len(counters))
	}
	if len(unknown) != 1 {
		t.Errorf("Expected 1 unknown, got %d", len(unknown))
	}
}

func TestSplitMetricsByType_Empty(t *testing.T) {
	gauges, counters, unknown := splitMetricsByType([]model.Metrics{})
	if len(gauges) != 0 || len(counters) != 0 || len(unknown) != 0 {
		t.Error("Expected empty slices for empty input")
	}
}

func TestSplitMetricsByType_NilValues(t *testing.T) {
	v := 1.0
	d := int64(10)
	metrics := []model.Metrics{
		{ID: "nil_gauge", MType: model.Gauge, Value: nil},
		{ID: "nil_counter", MType: model.Counter, Delta: nil},
		{ID: "valid_gauge", MType: model.Gauge, Value: &v},
		{ID: "valid_counter", MType: model.Counter, Delta: &d},
	}

	gauges, counters, _ := splitMetricsByType(metrics)

	// Only non-nil values should be included
	if len(gauges) != 1 {
		t.Errorf("Expected 1 gauge (non-nil), got %d", len(gauges))
	}
	if len(counters) != 1 {
		t.Errorf("Expected 1 counter (non-nil), got %d", len(counters))
	}
}

func TestSplitMetricsByType_EmptyID(t *testing.T) {
	v := 1.0
	metrics := []model.Metrics{
		{ID: "", MType: model.Gauge, Value: &v},
		{ID: "valid", MType: model.Gauge, Value: &v},
	}

	gauges, _, _ := splitMetricsByType(metrics)

	// Empty ID should be skipped
	if len(gauges) != 1 {
		t.Errorf("Expected 1 gauge (empty ID skipped), got %d", len(gauges))
	}
}
