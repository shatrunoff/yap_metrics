package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/shatrunoff/yap_metrics/internal/model"
	"github.com/shatrunoff/yap_metrics/internal/service"
	"github.com/shatrunoff/yap_metrics/internal/storage"
)

// BenchmarkUpdateMetricJSON измеряет производительность обновления метрики через JSON
func BenchmarkUpdateMetricJSON(b *testing.B) {
	st := storage.NewMemStorage()
	fileService := service.NewFileStorageService(nil, "", 0)
	handler := NewHandler(st, fileService, false, "", nil, "")

	value := 123.45
	metric := model.Metrics{
		ID:    "TestMetric",
		MType: model.Gauge,
		Value: &value,
	}

	body, _ := json.Marshal(metric)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/update/", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
	}
}

// BenchmarkGetMetricJSON измеряет производительность получения метрики через JSON
func BenchmarkGetMetricJSON(b *testing.B) {
	st := storage.NewMemStorage()
	ctx := context.Background()
	_ = st.UpdateGauge(ctx, "TestMetric", 123.45)

	fileService := service.NewFileStorageService(nil, "", 0)
	handler := NewHandler(st, fileService, false, "", nil, "")

	metric := model.Metrics{
		ID:    "TestMetric",
		MType: model.Gauge,
	}

	body, _ := json.Marshal(metric)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/value/", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
	}
}

// BenchmarkUpdateMetricsBatch измеряет производительность batch-обновления
func BenchmarkUpdateMetricsBatch(b *testing.B) {
	st := storage.NewMemStorage()
	fileService := service.NewFileStorageService(nil, "", 0)
	handler := NewHandler(st, fileService, false, "", nil, "")

	// Подготавливаем batch из 100 метрик
	metrics := make([]model.Metrics, 100)
	for i := 0; i < 100; i++ {
		value := float64(i)
		metrics[i] = model.Metrics{
			ID:    fmt.Sprintf("Metric%d", i),
			MType: model.Gauge,
			Value: &value,
		}
	}

	body, _ := json.Marshal(metrics)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/updates/", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
	}
}

// BenchmarkListMetrics измеряет производительность получения всех метрик
func BenchmarkListMetrics(b *testing.B) {
	st := storage.NewMemStorage()
	ctx := context.Background()

	// Заполняем хранилище данными
	for i := 0; i < 100; i++ {
		_ = st.UpdateGauge(ctx, fmt.Sprintf("Gauge%d", i), float64(i))
		_ = st.UpdateCounter(ctx, fmt.Sprintf("Counter%d", i), int64(i))
	}

	fileService := service.NewFileStorageService(nil, "", 0)
	handler := NewHandler(st, fileService, false, "", nil, "")

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
	}
}

// BenchmarkJSONEncodeDecode измеряет производительность кодирования/декодирования JSON
func BenchmarkJSONEncodeDecode(b *testing.B) {
	value := 123.45
	metric := model.Metrics{
		ID:    "TestMetric",
		MType: model.Gauge,
		Value: &value,
	}

	b.Run("Encode", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			var buf bytes.Buffer
			_ = json.NewEncoder(&buf).Encode(metric)
		}
	})

	b.Run("Decode", func(b *testing.B) {
		body, _ := json.Marshal(metric)
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			var m model.Metrics
			_ = json.NewDecoder(bytes.NewReader(body)).Decode(&m)
		}
	})
}

// BenchmarkGetClientIP измеряет производительность извлечения IP клиента
func BenchmarkGetClientIP(b *testing.B) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Forwarded-For", "192.168.1.1, 10.0.0.1")
	req.Header.Set("X-Real-IP", "192.168.1.2")
	req.RemoteAddr = "192.168.1.3:12345"

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = getClientIP(req)
	}
}
