package agent

import (
	"maps"
	"runtime"
	"sync"
	"sync/atomic"

	model "github.com/shatrunoff/yap_metrics/internal/model"
)

type MetricsCollector struct {
	runtimeMetrics map[string]model.Metrics
	PollCount      int64
	mu             sync.RWMutex
}

func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		runtimeMetrics: make(map[string]model.Metrics),
	}
}

// обновление gauge
func (mc *MetricsCollector) updateGauge(name string, value float64) {

	mc.runtimeMetrics[name] = model.Metrics{
		ID:    name,
		MType: model.Gauge,
		Value: &value,
	}
}

// обновление counter
func (mc *MetricsCollector) updateCounter(name string, delta int64) {

	if exist, ok := mc.runtimeMetrics[name]; ok && exist.Delta != nil {
		*exist.Delta += delta
		mc.runtimeMetrics[name] = exist
	} else {
		d := delta
		mc.runtimeMetrics[name] = model.Metrics{
			ID:    name,
			MType: model.Counter,
			Delta: &d,
		}
	}
}

// сбор метрик
func (mc *MetricsCollector) Collect() {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	// потокобезопасный ++
	atomic.AddInt64(&mc.PollCount, 1)

	// Создаем мапу метрик gauge
	gaugeMetrics := map[string]float64{
		"Alloc":         float64(memStats.Alloc),
		"BuckHashSys":   float64(memStats.BuckHashSys),
		"Frees":         float64(memStats.Frees),
		"GCCPUFraction": memStats.GCCPUFraction,
		"GCSys":         float64(memStats.GCSys),
		"HeapAlloc":     float64(memStats.HeapAlloc),
		"HeapIdle":      float64(memStats.HeapIdle),
		"HeapInuse":     float64(memStats.HeapInuse),
		"HeapObjects":   float64(memStats.HeapObjects),
		"HeapReleased":  float64(memStats.HeapReleased),
		"HeapSys":       float64(memStats.HeapSys),
		"LastGC":        float64(memStats.LastGC),
		"Lookups":       float64(memStats.Lookups),
		"MCacheInuse":   float64(memStats.MCacheInuse),
		"MCacheSys":     float64(memStats.MCacheSys),
		"MSpanInuse":    float64(memStats.MSpanInuse),
		"MSpanSys":      float64(memStats.MSpanSys),
		"Mallocs":       float64(memStats.Mallocs),
		"NextGC":        float64(memStats.NextGC),
		"NumForcedGC":   float64(memStats.NumForcedGC),
		"NumGC":         float64(memStats.NumGC),
		"OtherSys":      float64(memStats.OtherSys),
		"PauseTotalNs":  float64(memStats.PauseTotalNs),
		"StackInuse":    float64(memStats.StackInuse),
		"StackSys":      float64(memStats.StackSys),
		"Sys":           float64(memStats.Sys),
		"TotalAlloc":    float64(memStats.TotalAlloc),
	}

	// Обновляем метрики gauge в цикле
	for name, value := range gaugeMetrics {
		mc.updateGauge(name, value)
	}

	// counter
	mc.updateCounter("PollCount", 1)

}

// получение текущих метрик
func (mc *MetricsCollector) GetMetrics() map[string]model.Metrics {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	res := make(map[string]model.Metrics, len(mc.runtimeMetrics))
	maps.Copy(res, mc.runtimeMetrics)

	return res
}
