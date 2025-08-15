package agent

import (
	"maps"
	"runtime"
	"sync/atomic"

	model "github.com/shatrunoff/yap_metrics/internal/model"
)

type MetricsCollector struct {
	runtimeMetrics map[string]model.Metrics
	PollCount      int64
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
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	// потокобезопасный ++
	atomic.AddInt64(&mc.PollCount, 1)

	// gauge
	mc.updateGauge("Alloc", float64(memStats.Alloc))
	mc.updateGauge("BuckHashSys", float64(memStats.BuckHashSys))
	mc.updateGauge("Frees", float64(memStats.Frees))
	mc.updateGauge("GCCPUFraction", float64(memStats.GCCPUFraction))
	mc.updateGauge("GCSys", float64(memStats.GCSys))
	mc.updateGauge("HeapAlloc", float64(memStats.HeapAlloc))
	mc.updateGauge("HeapIdle", float64(memStats.HeapIdle))
	mc.updateGauge("HeapInuse", float64(memStats.HeapInuse))
	mc.updateGauge("HeapObjects", float64(memStats.HeapObjects))
	mc.updateGauge("HeapReleased", float64(memStats.HeapReleased))
	mc.updateGauge("HeapSys", float64(memStats.HeapSys))
	mc.updateGauge("LastGC", float64(memStats.LastGC))
	mc.updateGauge("Lookups", float64(memStats.Lookups))
	mc.updateGauge("MCacheInuse", float64(memStats.MCacheInuse))
	mc.updateGauge("MCacheSys", float64(memStats.MCacheSys))
	mc.updateGauge("MSpanInuse", float64(memStats.MSpanInuse))
	mc.updateGauge("MSpanSys", float64(memStats.MSpanSys))
	mc.updateGauge("Mallocs", float64(memStats.Mallocs))
	mc.updateGauge("NextGC", float64(memStats.NextGC))
	mc.updateGauge("NumForcedGC", float64(memStats.NumForcedGC))
	mc.updateGauge("NumGC", float64(memStats.NumGC))
	mc.updateGauge("OtherSys", float64(memStats.OtherSys))
	mc.updateGauge("PauseTotalNs", float64(memStats.PauseTotalNs))
	mc.updateGauge("StackInuse", float64(memStats.StackInuse))
	mc.updateGauge("StackSys", float64(memStats.StackSys))
	mc.updateGauge("Sys", float64(memStats.Sys))
	mc.updateGauge("TotalAlloc", float64(memStats.TotalAlloc))

	// counter
	mc.updateCounter("PollCount", 1)

}

// получение текущих метрик
func (mc *MetricsCollector) GetMetrics() map[string]model.Metrics {

	res := make(map[string]model.Metrics, len(mc.runtimeMetrics))
	maps.Copy(res, mc.runtimeMetrics)

	return res
}
