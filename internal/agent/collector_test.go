package agent

import (
	"testing"

	"github.com/shatrunoff/yap_metrics/internal/model"
)

func TestNewMetricsCollector(t *testing.T) {
	mc := NewMetricsCollector()
	if mc == nil {
		t.Fatal("expected non-nil MetricsCollector")
	}
	if mc.runtimeMetrics == nil {
		t.Error("expected runtimeMetrics to be initialized")
	}
	if mc.rand == nil {
		t.Error("expected rand to be initialized")
	}
}

func TestMetricsCollector_updateGauge(t *testing.T) {
	mc := NewMetricsCollector()
	mc.updateGauge("test", 42.5)

	metrics := mc.GetMetrics()
	metric, ok := metrics["test"]
	if !ok {
		t.Fatal("expected metric 'test' to exist")
	}

	if metric.MType != model.Gauge {
		t.Errorf("expected type %s, got %s", model.Gauge, metric.MType)
	}
	if metric.Value == nil || *metric.Value != 42.5 {
		t.Errorf("expected value 42.5, got %v", metric.Value)
	}
}

func TestMetricsCollector_updateCounter(t *testing.T) {
	mc := NewMetricsCollector()
	mc.updateCounter("test", 10)
	mc.updateCounter("test", 5)

	metrics := mc.GetMetrics()
	metric, ok := metrics["test"]
	if !ok {
		t.Fatal("expected metric 'test' to exist")
	}

	if metric.MType != model.Counter {
		t.Errorf("expected type %s, got %s", model.Counter, metric.MType)
	}
	if metric.Delta == nil || *metric.Delta != 15 {
		t.Errorf("expected delta 15, got %v", metric.Delta)
	}
}

func TestMetricsCollector_Collect(t *testing.T) {
	mc := NewMetricsCollector()
	mc.Collect()

	metrics := mc.GetMetrics()

	expectedGauges := []string{
		"Alloc", "BuckHashSys", "Frees", "GCCPUFraction", "GCSys",
		"HeapAlloc", "HeapIdle", "HeapInuse", "HeapObjects", "HeapReleased",
		"HeapSys", "LastGC", "Lookups", "MCacheInuse", "MCacheSys",
		"MSpanInuse", "MSpanSys", "Mallocs", "NextGC", "NumForcedGC",
		"NumGC", "OtherSys", "PauseTotalNs", "StackInuse", "StackSys",
		"Sys", "TotalAlloc", "RandomValue",
	}

	for _, name := range expectedGauges {
		metric, ok := metrics[name]
		if !ok {
			t.Errorf("expected gauge metric %s to exist", name)
			continue
		}
		if metric.MType != model.Gauge {
			t.Errorf("expected %s to be gauge, got %s", name, metric.MType)
		}
		if metric.Value == nil {
			t.Errorf("expected %s to have a value", name)
		}
	}

	pollCount, ok := metrics["PollCount"]
	if !ok {
		t.Fatal("expected PollCount metric to exist")
	}
	if pollCount.MType != model.Counter {
		t.Errorf("expected PollCount to be counter, got %s", pollCount.MType)
	}
	if pollCount.Delta == nil || *pollCount.Delta != 1 {
		t.Errorf("expected PollCount delta 1, got %v", pollCount.Delta)
	}
}

func TestMetricsCollector_Collect_Multiple(t *testing.T) {
	mc := NewMetricsCollector()
	mc.Collect()
	mc.Collect()
	mc.Collect()

	metrics := mc.GetMetrics()
	pollCount := metrics["PollCount"]

	if pollCount.Delta == nil || *pollCount.Delta != 3 {
		t.Errorf("expected PollCount delta 3 after 3 collections, got %v", pollCount.Delta)
	}
}

func TestMetricsCollector_GetMetrics(t *testing.T) {
	mc := NewMetricsCollector()
	mc.updateGauge("gauge1", 10.5)
	mc.updateGauge("gauge2", 20.5)
	mc.updateCounter("counter1", 5)

	metrics := mc.GetMetrics()

	if len(metrics) != 3 {
		t.Errorf("expected 3 metrics, got %d", len(metrics))
	}

	if _, ok := metrics["gauge1"]; !ok {
		t.Error("expected gauge1 to exist")
	}
	if _, ok := metrics["gauge2"]; !ok {
		t.Error("expected gauge2 to exist")
	}
	if _, ok := metrics["counter1"]; !ok {
		t.Error("expected counter1 to exist")
	}
}

func TestMetricsCollector_GetMetrics_Copy(t *testing.T) {
	mc := NewMetricsCollector()
	mc.updateGauge("test", 42.5)

	metrics1 := mc.GetMetrics()
	metrics2 := mc.GetMetrics()

	if &metrics1 == &metrics2 {
		t.Error("GetMetrics should return a copy, not the same reference")
	}

	if len(metrics1) != len(metrics2) {
		t.Error("both copies should have the same length")
	}
}

func TestMetricsCollector_ConcurrentAccess(t *testing.T) {
	mc := NewMetricsCollector()

	done := make(chan bool)

	go func() {
		for i := 0; i < 100; i++ {
			mc.Collect()
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 100; i++ {
			mc.GetMetrics()
		}
		done <- true
	}()

	<-done
	<-done
}
