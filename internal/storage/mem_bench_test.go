package storage

import (
	"context"
	"fmt"
	"testing"

	"github.com/shatrunoff/yap_metrics/internal/model"
)

// BenchmarkUpdateGauge измеряет производительность обновления gauge-метрики
func BenchmarkUpdateGauge(b *testing.B) {
	ms := NewMemStorage()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ms.UpdateGauge(ctx, "TestMetric", float64(i))
	}
}

// BenchmarkUpdateCounter измеряет производительность обновления counter-метрики
func BenchmarkUpdateCounter(b *testing.B) {
	ms := NewMemStorage()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ms.UpdateCounter(ctx, "TestCounter", 1)
	}
}

// BenchmarkGetMetric измеряет производительность получения метрики
func BenchmarkGetMetric(b *testing.B) {
	ms := NewMemStorage()
	ctx := context.Background()
	_ = ms.UpdateGauge(ctx, "TestMetric", 123.45)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ms.GetMetric(ctx, model.Gauge, "TestMetric")
	}
}

// BenchmarkGetAll измеряет производительность получения всех метрик
func BenchmarkGetAll(b *testing.B) {
	ms := NewMemStorage()
	ctx := context.Background()

	// Заполняем хранилище данными
	for i := 0; i < 100; i++ {
		_ = ms.UpdateGauge(ctx, fmt.Sprintf("Gauge%d", i), float64(i))
		_ = ms.UpdateCounter(ctx, fmt.Sprintf("Counter%d", i), int64(i))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ms.GetAll(ctx)
	}
}

// BenchmarkUpdateMetricsBatch измеряет производительность batch-обновления
func BenchmarkUpdateMetricsBatch(b *testing.B) {
	ms := NewMemStorage()
	ctx := context.Background()

	// Подготавливаем batch метрик
	metrics := make([]model.Metrics, 100)
	for i := 0; i < 100; i++ {
		if i%2 == 0 {
			value := float64(i)
			metrics[i] = model.Metrics{
				ID:    fmt.Sprintf("Gauge%d", i),
				MType: model.Gauge,
				Value: &value,
			}
		} else {
			delta := int64(i)
			metrics[i] = model.Metrics{
				ID:    fmt.Sprintf("Counter%d", i),
				MType: model.Counter,
				Delta: &delta,
			}
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ms.UpdateMetricsBatch(ctx, metrics)
	}
}

// BenchmarkConcurrentUpdates измеряет производительность при параллельных обновлениях
func BenchmarkConcurrentUpdates(b *testing.B) {
	ms := NewMemStorage()
	ctx := context.Background()

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			_ = ms.UpdateGauge(ctx, fmt.Sprintf("Gauge%d", i%10), float64(i))
			i++
		}
	})
}

// BenchmarkConcurrentReads измеряет производительность при параллельных чтениях
func BenchmarkConcurrentReads(b *testing.B) {
	ms := NewMemStorage()
	ctx := context.Background()

	// Подготовка данных
	for i := 0; i < 10; i++ {
		_ = ms.UpdateGauge(ctx, fmt.Sprintf("Gauge%d", i), float64(i))
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			_, _ = ms.GetMetric(ctx, model.Gauge, fmt.Sprintf("Gauge%d", i%10))
			i++
		}
	})
}

// BenchmarkSaveToFile измеряет производительность сохранения в файл
func BenchmarkSaveToFile(b *testing.B) {
	ms := NewMemStorage()
	ctx := context.Background()

	// Заполняем хранилище данными
	for i := 0; i < 100; i++ {
		_ = ms.UpdateGauge(ctx, fmt.Sprintf("Gauge%d", i), float64(i))
		_ = ms.UpdateCounter(ctx, fmt.Sprintf("Counter%d", i), int64(i))
	}

	tmpFile := b.TempDir() + "/metrics.json"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ms.SaveToFile(tmpFile)
	}
}

// BenchmarkLoadFromFile измеряет производительность загрузки из файла
func BenchmarkLoadFromFile(b *testing.B) {
	ms := NewMemStorage()
	ctx := context.Background()

	// Подготавливаем файл с данными
	for i := 0; i < 100; i++ {
		_ = ms.UpdateGauge(ctx, fmt.Sprintf("Gauge%d", i), float64(i))
		_ = ms.UpdateCounter(ctx, fmt.Sprintf("Counter%d", i), int64(i))
	}

	tmpFile := b.TempDir() + "/metrics.json"
	_ = ms.SaveToFile(tmpFile)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		newMs := NewMemStorage()
		_ = newMs.LoadFromFile(tmpFile)
	}
}
