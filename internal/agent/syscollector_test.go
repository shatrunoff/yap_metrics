package agent

import (
	"testing"
)

func TestMetricsCollector_CollectSys(t *testing.T) {
	mc := NewMetricsCollector()

	// Collect system metrics
	mc.CollectSys()

	metrics := mc.GetMetrics()

	// Should have TotalMemory and FreeMemory
	foundTotal := false
	foundFree := false
	foundCPU := false

	for id := range metrics {
		if id == "TotalMemory" {
			foundTotal = true
		}
		if id == "FreeMemory" {
			foundFree = true
		}
		if len(id) > 14 && id[:14] == "CPUutilization" {
			foundCPU = true
		}
	}

	if !foundTotal {
		t.Error("Expected TotalMemory metric")
	}
	if !foundFree {
		t.Error("Expected FreeMemory metric")
	}
	// CPU metrics may not be available in all environments
	t.Logf("Found CPU metrics: %v", foundCPU)
}

func TestMetricsCollector_CollectSys_Multiple(t *testing.T) {
	mc := NewMetricsCollector()

	// Collect multiple times
	for i := 0; i < 3; i++ {
		mc.CollectSys()
	}

	metrics := mc.GetMetrics()

	// Should still have metrics
	if len(metrics) == 0 {
		t.Error("Expected some metrics after multiple CollectSys calls")
	}
}
