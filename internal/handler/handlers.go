package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"text/template"

	"github.com/go-chi/chi/v5"
	"github.com/shatrunoff/yap_metrics/internal/middleware"
	"github.com/shatrunoff/yap_metrics/internal/model"
	"go.uber.org/zap"
)

const htmlPage = `
	<!DOCTYPE html>
	<html>
	<head><title>Metrics</title></head>
	<body>
	<h1>Metrics</h1>
	<ul>
		{{range $name, $metric := .}}
			<li>{{$metric.MType}} {{$metric.ID}}:
				{{if eq $metric.MType "` + model.Gauge + `"}}{{$metric.Value}}{{end}}
				{{if eq $metric.MType "` + model.Counter + `"}}{{$metric.Delta}}{{end}}
			</li>
		{{end}}
	</ul>
	</body>
	</html>`

var (
	metricsTemplate *template.Template
	once            sync.Once
)

func initTemplates() {
	once.Do(func() {
		metricsTemplate = template.Must(template.New("metrics").Parse(htmlPage))
	})
}

type Storage interface {
	UpdateGauge(name string, value float64)
	UpdateCounter(name string, delta int64)
	GetMetric(metricType, name string) (model.Metrics, bool)
	GetAll() map[string]model.Metrics
}

type Handler struct {
	storage Storage
	logger  *zap.Logger
	sugar   *zap.SugaredLogger
}

// хэндлер обновления метрики
func (h *Handler) updateMetric(w http.ResponseWriter, r *http.Request) {

	// для отладки
	// log.Printf("Incoming update: %s %s", r.Method, r.URL.Path)

	// проверка content-type
	contentType := r.Header.Get("Content-Type")
	if contentType == "application/json" {
		h.updateMetricJSON(w, r)
		return
	}

	metricType := chi.URLParam(r, "type")
	metricName := chi.URLParam(r, "name")
	metricValue := chi.URLParam(r, "value")

	switch metricType {
	case model.Gauge:
		value, err := strconv.ParseFloat(metricValue, 64)
		if err != nil {
			http.Error(w, "ERROR: invalid Gauge metric", http.StatusBadRequest)
		}
		h.storage.UpdateGauge(metricName, value)

	case model.Counter:
		delta, err := strconv.ParseInt(metricValue, 10, 64)
		if err != nil {
			http.Error(w, "ERROR: invalid Counter metric", http.StatusBadRequest)
		}
		h.storage.UpdateCounter(metricName, delta)

	default:
		http.Error(w, "ERROR: unknow metric type", http.StatusBadRequest)
	}
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) updateMetricJSON(w http.ResponseWriter, r *http.Request) {
	var metric model.Metrics

	// заголовок ответа
	w.Header().Set("Content-Type", "application/json")

	// декодируем JSON
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&metric)
	if err != nil {
		h.sugar.Errorw("Failed to decode JSON: ", "error", err)
		http.Error(w, `{"error": "Invalid JSON"}`, http.StatusBadRequest)
		return
	}

	// валидация полей
	if metric.ID == "" || metric.MType == "" {
		h.sugar.Warnw("Missing required fields in JSON", "metric", metric)
		http.Error(w, `{"error": "Missing required fields: id or type"}`, http.StatusBadRequest)
		return
	}

	// обновляем метрики
	switch metric.MType {
	case model.Gauge:
		if metric.Value == nil {
			h.sugar.Warnw("Missing value for gauge in JSON", "metric", metric)
			http.Error(w, `{"error": "Missing value for gauge metric"}`, http.StatusBadRequest)
			return
		}
		h.storage.UpdateGauge(metric.ID, *metric.Value)

	case model.Counter:
		if metric.Delta == nil {
			h.sugar.Warnw("Missing value for counter in JSON", "metric", metric)
			http.Error(w, `{"error": "Missing value for counter metric"}`, http.StatusBadRequest)
		}
		h.storage.UpdateCounter(metric.ID, *metric.Delta)

	default:
		h.sugar.Warnw("Unknown metric type in JSON", "type", metric.MType)
		http.Error(w, `{"error": "Unknown metric type"}`, http.StatusBadRequest)
		return
	}

	// возвращаем обновленную метрику
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(metric); err != nil {
		h.sugar.Errorw("Failed to encode JSON response", "error", err)
		http.Error(w, `{"error": "Internal server error"}`, http.StatusInternalServerError)
	}
}

// хэндлер получения метрики
func (h *Handler) getMetric(w http.ResponseWriter, r *http.Request) {
	metricType := chi.URLParam(r, "type")
	metricName := chi.URLParam(r, "name")

	metric, ok := h.storage.GetMetric(metricType, metricName)
	if !ok {
		http.NotFound(w, r)
		return
	}
	switch metric.MType {
	case model.Gauge:
		fmt.Fprintf(w, "%g", *metric.Value)
	case model.Counter:
		fmt.Fprintf(w, "%d", *metric.Delta)
	}
}

// получение метрик через JSON
func (h *Handler) getMetricJSON(w http.ResponseWriter, r *http.Request) {
	var requestMetric model.Metrics

	// устанавливаем заголовок Content-Type
	w.Header().Set("Content-Type", "application/json")

	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&requestMetric); err != nil {
		h.sugar.Errorw("Failed to decode JSON", "error", err)
		http.Error(w, `{"error": "Invalid JSON"}`, http.StatusBadRequest)
		return
	}

	// валидация обязательных полей
	if requestMetric.ID == "" || requestMetric.MType == "" {
		h.sugar.Warnw("Missing required fields", "metric", requestMetric)
		http.Error(w, `{"error": "Missing required fields: id or type"}`, http.StatusBadRequest)
		return
	}

	// получаем метрику из хранилища
	metric, ok := h.storage.GetMetric(requestMetric.MType, requestMetric.ID)
	if !ok {
		h.sugar.Warnw("Metric not found", "type", requestMetric.MType, "name", requestMetric.ID)
		http.Error(w, `{"error": "Metric not found"}`, http.StatusNotFound)
		return
	}

	// возвращаем найденную метрику
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(metric); err != nil {
		h.sugar.Errorw("Failed to encode response", "error", err)
		http.Error(w, `{"error": "Internal server error"}`, http.StatusInternalServerError)
	}
}

// хэндлер получения всех метрик
func (h *Handler) listMetrics(w http.ResponseWriter, r *http.Request) {
	metrics := h.storage.GetAll()
	initTemplates()

	w.WriteHeader(http.StatusOK)
	metricsTemplate.Execute(w, metrics)
}

// основной хэндлер
func NewHandler(storage Storage) http.Handler {
	// инициализируем логгер
	err := middleware.InitLogger()
	if err != nil {
		panic(err)
	}

	// получаем логгер
	logger := middleware.GetLogger()
	sugar := middleware.GetSugar()
	defer logger.Sync()

	handler := &Handler{
		storage: storage,
		logger:  logger,
		sugar:   sugar,
	}

	router := chi.NewRouter()
	router.Use(middleware.LoggingMiddleware)

	// старые эндпоинты
	router.Post("/update/{type}/{name}/{value}", handler.updateMetric)
	router.Get("/value/{type}/{name}", handler.getMetric)
	router.Get("/", handler.listMetrics)

	// новые для JSON
	router.Post("/update", handler.updateMetricJSON)
	router.Post("/value", handler.getMetricJSON)

	return router

}
