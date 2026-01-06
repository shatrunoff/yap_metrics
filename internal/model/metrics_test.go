package model

import (
	"encoding/json"
	"testing"
)

func TestMetricsJSON(t *testing.T) {
	tests := []struct {
		name     string
		metrics  Metrics
		expected string
	}{
		{
			name: "counter metric",
			metrics: Metrics{
				ID:    "test_counter",
				MType: Counter,
				Delta: int64Ptr(42),
				Hash:  "test_hash",
			},
			expected: `{"id":"test_counter","type":"counter","delta":42,"hash":"test_hash"}`,
		},
		{
			name: "gauge metric",
			metrics: Metrics{
				ID:    "test_gauge",
				MType: Gauge,
				Value: float64Ptr(3.14),
				Hash:  "test_hash",
			},
			expected: `{"id":"test_gauge","type":"gauge","value":3.14,"hash":"test_hash"}`,
		},
		{
			name: "metric without hash",
			metrics: Metrics{
				ID:    "test_no_hash",
				MType: Counter,
				Delta: int64Ptr(0),
			},
			expected: `{"id":"test_no_hash","type":"counter","delta":0}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test marshaling
			data, err := json.Marshal(tt.metrics)
			if err != nil {
				t.Fatalf("Failed to marshal metrics: %v", err)
			}

			if string(data) != tt.expected {
				t.Errorf("JSON mismatch. Got %s, want %s", string(data), tt.expected)
			}

			// Test unmarshaling
			var unmarshaled Metrics
			if err := json.Unmarshal(data, &unmarshaled); err != nil {
				t.Fatalf("Failed to unmarshal metrics: %v", err)
			}

			if unmarshaled.ID != tt.metrics.ID {
				t.Errorf("ID mismatch. Got %s, want %s", unmarshaled.ID, tt.metrics.ID)
			}
			if unmarshaled.MType != tt.metrics.MType {
				t.Errorf("MType mismatch. Got %s, want %s", unmarshaled.MType, tt.metrics.MType)
			}
			if !equalInt64Ptr(unmarshaled.Delta, tt.metrics.Delta) {
				t.Errorf("Delta mismatch. Got %v, want %v", unmarshaled.Delta, tt.metrics.Delta)
			}
			if !equalFloat64Ptr(unmarshaled.Value, tt.metrics.Value) {
				t.Errorf("Value mismatch. Got %v, want %v", unmarshaled.Value, tt.metrics.Value)
			}
		})
	}
}

func TestMetricsConstants(t *testing.T) {
	if Counter != "counter" {
		t.Errorf("Counter constant mismatch. Got %s, want counter", Counter)
	}
	if Gauge != "gauge" {
		t.Errorf("Gauge constant mismatch. Got %s, want gauge", Gauge)
	}
}

func TestMetricsOmitEmpty(t *testing.T) {
	// Test that nil pointers are omitted
	m := Metrics{
		ID:    "test",
		MType: Counter,
		// Delta and Value are nil
	}

	data, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("Failed to marshal metrics: %v", err)
	}

	expected := `{"id":"test","type":"counter"}`
	if string(data) != expected {
		t.Errorf("Expected omitempty behavior. Got %s, want %s", string(data), expected)
	}
}

// Helper functions
func int64Ptr(v int64) *int64 {
	return &v
}

func float64Ptr(v float64) *float64 {
	return &v
}

func equalInt64Ptr(a, b *int64) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}

func equalFloat64Ptr(a, b *float64) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}
