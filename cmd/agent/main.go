package main

import (
	"fmt"
	"math/rand"
	"runtime"
	"time"
)

const (
	pollInterval   = time.Second * 2
	reportInterval = time.Second * 10
	serverURL      = "http://localhost:8080"
)

type Metrics struct {
	gauge   map[string]float64
	counter map[string]int64
}

// создание новых метрики
func NewMetrics() *Metrics {
	return &Metrics{
		gauge:   make(map[string]float64),
		counter: make(map[string]int64),
	}
}

// для печати и отладки
func (m *Metrics) String() string {
	str := "gaude метрики:\n"
	for key, value := range m.gauge {
		str += fmt.Sprintf("%s: %.2f\n", key, value)
	}
	str += "\ncounter метрики:\n"
	for key, value := range m.counter {
		str += fmt.Sprintf("%s: %d\n", key, value)
	}
	return str
}

// сбор метрик
func (m *Metrics) CollectMetrics() {

	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	m.gauge = map[string]float64{
		"Alloc":         float64(memStats.Alloc),
		"BuckHashSys":   float64(memStats.BuckHashSys),
		"Frees":         float64(memStats.Frees),
		"GCCPUFraction": float64(memStats.GCCPUFraction),
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
		"RandomValue":   rand.Float64(),
	}

	m.counter["PollCount"]++
}

func main() {
	metrics := NewMetrics()
	// metrics.CollectMetrics()

	done := make(chan struct{})

	go func() {
		ticker := time.NewTicker(pollInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				metrics.CollectMetrics()
				fmt.Println("Метрики собраны в:", time.Now().Format("15:04:05"))
				fmt.Println(metrics.String())
			case <-done:
				return
			}
		}
	}()

	time.Sleep(1000 * time.Second)

}
