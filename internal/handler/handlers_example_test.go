package handler_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"

	"github.com/shatrunoff/yap_metrics/internal/handler"
	"github.com/shatrunoff/yap_metrics/internal/model"
	"github.com/shatrunoff/yap_metrics/internal/storage"
)

// Example_updateMetricGauge демонстрирует обновление gauge метрики через URL параметры
func Example_updateMetricGauge() {
	// Создаем хранилище и handler
	st := storage.NewMemStorage()
	h := handler.NewHandler(st, nil, false, "", nil)

	// Создаем тестовый запрос
	req := httptest.NewRequest(http.MethodPost, "/update/gauge/temperature/25.5", nil)
	w := httptest.NewRecorder()

	// Выполняем запрос
	h.ServeHTTP(w, req)

	// Проверяем результат
	fmt.Printf("Status: %d\n", w.Code)
	// Output:
	// Status: 200
}

// Example_updateMetricCounter демонстрирует обновление counter метрики через URL параметры
func Example_updateMetricCounter() {
	// Создаем хранилище и handler
	st := storage.NewMemStorage()
	h := handler.NewHandler(st, nil, false, "", nil)

	// Создаем тестовый запрос
	req := httptest.NewRequest(http.MethodPost, "/update/counter/requests/100", nil)
	w := httptest.NewRecorder()

	// Выполняем запрос
	h.ServeHTTP(w, req)

	fmt.Printf("Status: %d\n", w.Code)
	// Output:
	// Status: 200
}

// Example_getMetric демонстрирует получение значения метрики через URL параметры
func Example_getMetric() {
	// Создаем хранилище и handler
	st := storage.NewMemStorage()
	h := handler.NewHandler(st, nil, false, "", nil)

	// Сначала добавляем метрику
	updateReq := httptest.NewRequest(http.MethodPost, "/update/gauge/temperature/25.5", nil)
	updateW := httptest.NewRecorder()
	h.ServeHTTP(updateW, updateReq)

	// Затем получаем ее значение
	getReq := httptest.NewRequest(http.MethodGet, "/value/gauge/temperature", nil)
	getW := httptest.NewRecorder()
	h.ServeHTTP(getW, getReq)

	fmt.Printf("Status: %d\n", getW.Code)
	fmt.Printf("Value: %s\n", getW.Body.String())
	// Output:
	// Status: 200
	// Value: 25.5
}

// Example_updateMetricJSON демонстрирует обновление метрики через JSON API
func Example_updateMetricJSON() {
	// Создаем хранилище и handler
	st := storage.NewMemStorage()
	h := handler.NewHandler(st, nil, false, "", nil)

	// Создаем метрику
	value := 42.5
	metric := model.Metrics{
		ID:    "cpu_usage",
		MType: "gauge",
		Value: &value,
	}

	// Преобразуем в JSON
	body, _ := json.Marshal(metric)

	// Создаем запрос
	req := httptest.NewRequest(http.MethodPost, "/update/", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Выполняем запрос
	h.ServeHTTP(w, req)

	fmt.Printf("Status: %d\n", w.Code)

	// Парсим ответ
	var response model.Metrics
	json.NewDecoder(w.Body).Decode(&response)
	fmt.Printf("Metric ID: %s\n", response.ID)
	fmt.Printf("Metric Type: %s\n", response.MType)
	fmt.Printf("Metric Value: %.1f\n", *response.Value)
	// Output:
	// Status: 200
	// Metric ID: cpu_usage
	// Metric Type: gauge
	// Metric Value: 42.5
}

// Example_getMetricJSON демонстрирует получение метрики через JSON API
func Example_getMetricJSON() {
	// Создаем хранилище и handler
	st := storage.NewMemStorage()
	h := handler.NewHandler(st, nil, false, "", nil)

	// Сначала добавляем метрику
	value := 100.0
	updateMetric := model.Metrics{
		ID:    "memory_usage",
		MType: "gauge",
		Value: &value,
	}
	updateBody, _ := json.Marshal(updateMetric)
	updateReq := httptest.NewRequest(http.MethodPost, "/update/", bytes.NewReader(updateBody))
	updateReq.Header.Set("Content-Type", "application/json")
	updateW := httptest.NewRecorder()
	h.ServeHTTP(updateW, updateReq)

	// Теперь получаем метрику
	getMetric := model.Metrics{
		ID:    "memory_usage",
		MType: "gauge",
	}
	getBody, _ := json.Marshal(getMetric)
	getReq := httptest.NewRequest(http.MethodPost, "/value/", bytes.NewReader(getBody))
	getReq.Header.Set("Content-Type", "application/json")
	getW := httptest.NewRecorder()
	h.ServeHTTP(getW, getReq)

	fmt.Printf("Status: %d\n", getW.Code)

	var response model.Metrics
	json.NewDecoder(getW.Body).Decode(&response)
	fmt.Printf("Metric ID: %s\n", response.ID)
	fmt.Printf("Metric Value: %.0f\n", *response.Value)
	// Output:
	// Status: 200
	// Metric ID: memory_usage
	// Metric Value: 100
}

// Example_updateMetricsBatch демонстрирует батч-обновление метрик
func Example_updateMetricsBatch() {
	// Создаем хранилище и handler
	st := storage.NewMemStorage()
	h := handler.NewHandler(st, nil, false, "", nil)

	// Создаем набор метрик
	value1 := 75.5
	value2 := 50.0
	delta := int64(10)

	metrics := []model.Metrics{
		{
			ID:    "cpu",
			MType: "gauge",
			Value: &value1,
		},
		{
			ID:    "memory",
			MType: "gauge",
			Value: &value2,
		},
		{
			ID:    "requests",
			MType: "counter",
			Delta: &delta,
		},
	}

	// Преобразуем в JSON
	body, _ := json.Marshal(metrics)

	// Создаем запрос
	req := httptest.NewRequest(http.MethodPost, "/updates/", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Выполняем запрос
	h.ServeHTTP(w, req)

	fmt.Printf("Status: %d\n", w.Code)
	// Output:
	// Status: 200
}

// Example_pingDB демонстрирует проверку соединения с БД
func Example_pingDB() {
	// Создаем хранилище и handler
	st := storage.NewMemStorage()
	h := handler.NewHandler(st, nil, false, "", nil)

	// Создаем запрос
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	w := httptest.NewRecorder()

	// Выполняем запрос
	h.ServeHTTP(w, req)

	fmt.Printf("Status: %d\n", w.Code)
	fmt.Printf("Response: %s\n", w.Body.String())
	// Output:
	// Status: 200
	// Response: DB connection OK
}

// Example_listMetrics демонстрирует получение списка всех метрик в HTML формате
func Example_listMetrics() {
	// Создаем хранилище и handler
	st := storage.NewMemStorage()
	h := handler.NewHandler(st, nil, false, "", nil)

	// Добавляем несколько метрик
	value := 42.0
	updateMetric := model.Metrics{
		ID:    "test_metric",
		MType: "gauge",
		Value: &value,
	}
	updateBody, _ := json.Marshal(updateMetric)
	updateReq := httptest.NewRequest(http.MethodPost, "/update/", bytes.NewReader(updateBody))
	updateReq.Header.Set("Content-Type", "application/json")
	updateW := httptest.NewRecorder()
	h.ServeHTTP(updateW, updateReq)

	// Получаем список метрик
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	fmt.Printf("Status: %d\n", w.Code)
	fmt.Printf("Content-Type: %s\n", w.Header().Get("Content-Type"))

	// Проверяем, что ответ содержит HTML
	body, _ := io.ReadAll(w.Body)
	containsHTML := bytes.Contains(body, []byte("<!DOCTYPE html>"))
	fmt.Printf("Contains HTML: %t\n", containsHTML)
	// Output:
	// Status: 200
	// Content-Type: text/html
	// Contains HTML: true
}
