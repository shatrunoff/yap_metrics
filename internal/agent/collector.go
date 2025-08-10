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

// обновление gaude
func (mc *MetricsCollector) updateGaude(name string, value float64) {

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

	// gaude
	mc.updateGaude("Alloc", float64(memStats.Alloc))
	mc.updateGaude("BuckHashSys", float64(memStats.BuckHashSys))
	mc.updateGaude("Frees", float64(memStats.Frees))
	mc.updateGaude("GCCPUFraction", float64(memStats.GCCPUFraction))
	mc.updateGaude("GCSys", float64(memStats.GCSys))
	mc.updateGaude("HeapAlloc", float64(memStats.HeapAlloc))
	mc.updateGaude("HeapIdle", float64(memStats.HeapIdle))
	mc.updateGaude("HeapInuse", float64(memStats.HeapInuse))
	mc.updateGaude("HeapObjects", float64(memStats.HeapObjects))
	mc.updateGaude("HeapReleased", float64(memStats.HeapReleased))
	mc.updateGaude("HeapSys", float64(memStats.HeapSys))
	mc.updateGaude("LastGC", float64(memStats.LastGC))
	mc.updateGaude("Lookups", float64(memStats.Lookups))
	mc.updateGaude("MCacheInuse", float64(memStats.MCacheInuse))
	mc.updateGaude("MCacheSys", float64(memStats.MCacheSys))
	mc.updateGaude("MSpanInuse", float64(memStats.MSpanInuse))
	mc.updateGaude("MSpanSys", float64(memStats.MSpanSys))
	mc.updateGaude("Mallocs", float64(memStats.Mallocs))
	mc.updateGaude("NextGC", float64(memStats.NextGC))
	mc.updateGaude("NumForcedGC", float64(memStats.NumForcedGC))
	mc.updateGaude("NumGC", float64(memStats.NumGC))
	mc.updateGaude("OtherSys", float64(memStats.OtherSys))
	mc.updateGaude("PauseTotalNs", float64(memStats.PauseTotalNs))
	mc.updateGaude("StackInuse", float64(memStats.StackInuse))
	mc.updateGaude("StackSys", float64(memStats.StackSys))
	mc.updateGaude("Sys", float64(memStats.Sys))
	mc.updateGaude("TotalAlloc", float64(memStats.TotalAlloc))

	// counter
	mc.updateCounter("PollCount", 1)

}

// получение текущих метрик
func (mc *MetricsCollector) GetMetrics() map[string]model.Metrics {

	res := make(map[string]model.Metrics, len(mc.runtimeMetrics))
	maps.Copy(res, mc.runtimeMetrics)

	return res
}
