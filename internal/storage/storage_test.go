package storage

import (
	"context"
	"os"
	"testing"
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
