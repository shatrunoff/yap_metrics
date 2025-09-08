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

// хэндлер получения всех метрик
func (h *Handler) listMetrics(w http.ResponseWriter, r *http.Request) {
	metrics := h.storage.GetAll()
	initTemplates()

	w.WriteHeader(http.StatusOK)
	metricsTemplate.Execute(w, metrics)
}

// хэндлер обновления метрики через JSON
func (h *Handler) updateMetricJSON(w http.ResponseWriter, r *http.Request) {
	var metric model.Metrics

	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&metric); err != nil {
		http.Error(w, "ERROR: invalid JSON", http.StatusBadRequest)
		return
	}

	if metric.ID == "" {
		http.Error(w, "ERROR: metric ID is required", http.StatusBadRequest)
		return
	}

	switch metric.MType {
	case model.Gauge:
		if metric.Value == nil {
			http.Error(w, "ERROR: value is required for gauge", http.StatusBadRequest)
			return
		}
		h.storage.UpdateGauge(metric.ID, *metric.Value)

	case model.Counter:
		if metric.Delta == nil {
			http.Error(w, "ERROR: delta is required for counter", http.StatusBadRequest)
			return
		}
		h.storage.UpdateCounter(metric.ID, *metric.Delta)

	default:
		http.Error(w, "ERROR: unknown metric type", http.StatusBadRequest)
		return
	}

	// Возвращаем обновленную метрику
	updatedMetric, ok := h.storage.GetMetric(metric.MType, metric.ID)
	if !ok {
		http.Error(w, "ERROR: failed to get updated metric", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(updatedMetric)
}

// хэндлер получения метрики через JSON
func (h *Handler) getMetricJSON(w http.ResponseWriter, r *http.Request) {
	var metric model.Metrics

	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&metric); err != nil {
		http.Error(w, "ERROR: invalid JSON", http.StatusBadRequest)
		return
	}

	if metric.ID == "" || metric.MType == "" {
		http.Error(w, "ERROR: metric ID and type are required", http.StatusBadRequest)
		return
	}

	foundMetric, ok := h.storage.GetMetric(metric.MType, metric.ID)
	if !ok {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(foundMetric)
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

	// Старые эндпоинты
	router.Post("/update/{type}/{name}/{value}", handler.updateMetric)
	router.Get("/value/{type}/{name}", handler.getMetric)
	router.Get("/", handler.listMetrics)

	// Новые JSON эндпоинты
	router.Post("/update/", handler.updateMetricJSON)
	router.Post("/value/", handler.getMetricJSON)

	return router
}
