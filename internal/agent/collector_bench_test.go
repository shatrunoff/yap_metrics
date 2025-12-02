package agent

import (
	"testing"
)

// BenchmarkCollect измеряет производительность сбора метрик
func BenchmarkCollect(b *testing.B) {
	mc := NewMetricsCollector()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		mc.Collect()
	}
}

// BenchmarkGetMetrics измеряет производительность получения метрик
func BenchmarkGetMetrics(b *testing.B) {
	mc := NewMetricsCollector()
	mc.Collect() // Заполняем данными

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = mc.GetMetrics()
	}
}

// BenchmarkConcurrentCollect измеряет производительность при параллельном сборе
func BenchmarkConcurrentCollect(b *testing.B) {
	mc := NewMetricsCollector()

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			mc.Collect()
		}
	})
}

// BenchmarkConcurrentGetMetrics измеряет производительность при параллельном чтении
func BenchmarkConcurrentGetMetrics(b *testing.B) {
	mc := NewMetricsCollector()
	mc.Collect() // Заполняем данными

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = mc.GetMetrics()
		}
	})
}

// BenchmarkUpdateGauge измеряет производительность обновления gauge
func BenchmarkUpdateGauge(b *testing.B) {
	mc := NewMetricsCollector()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		mc.updateGauge("TestMetric", 123.45)
	}
}

// BenchmarkUpdateCounter измеряет производительность обновления counter
func BenchmarkUpdateCounter(b *testing.B) {
	mc := NewMetricsCollector()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		mc.updateCounter("TestCounter", 1)
	}
}
