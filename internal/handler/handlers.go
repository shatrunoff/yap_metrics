package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"text/template"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/shatrunoff/yap_metrics/internal/middleware"
	"github.com/shatrunoff/yap_metrics/internal/model"
	"github.com/shatrunoff/yap_metrics/internal/service"
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

// интерфейс для всех типов хранилищ
type Storage interface {
	Ping(ctx context.Context) error
	UpdateGauge(ctx context.Context, name string, value float64) error
	UpdateCounter(ctx context.Context, name string, delta int64) error
	GetMetric(ctx context.Context, metricType, name string) (model.Metrics, error)
	GetAll(ctx context.Context) (map[string]model.Metrics, error)
}

// интерфейс только для файлового хранилища
type FileSaver interface {
	SaveToFile(path string) error
	LoadFromFile(filename string) error
}

type Handler struct {
	storage     Storage
	fileService *service.FileStorageService
	syncSave    bool
	logger      *zap.Logger
	sugar       *zap.SugaredLogger
	fileSaver   FileSaver
}

func (h *Handler) pingDB(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if err := h.storage.Ping(ctx); err != nil {
		h.logger.Error("DB ping failed", zap.Error(err))
		http.Error(w, "DB connection failed", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("DB connection OK"))
}

func (h *Handler) updateMetric(w http.ResponseWriter, r *http.Request) {
	metricType := chi.URLParam(r, "type")
	metricName := chi.URLParam(r, "name")
	metricValue := chi.URLParam(r, "value")

	ctx := r.Context()

	switch metricType {
	case model.Gauge:
		value, err := strconv.ParseFloat(metricValue, 64)
		if err != nil {
			http.Error(w, "ERROR: invalid Gauge metric", http.StatusBadRequest)
			return
		}
		if err := h.storage.UpdateGauge(ctx, metricName, value); err != nil {
			h.logger.Error("Failed to update gauge", zap.Error(err))
			http.Error(w, "ERROR: failed to update gauge", http.StatusInternalServerError)
			return
		}

	case model.Counter:
		delta, err := strconv.ParseInt(metricValue, 10, 64)
		if err != nil {
			http.Error(w, "ERROR: invalid Counter metric", http.StatusBadRequest)
			return
		}
		if err := h.storage.UpdateCounter(ctx, metricName, delta); err != nil {
			h.logger.Error("Failed to update counter", zap.Error(err))
			http.Error(w, "ERROR: failed to update counter", http.StatusInternalServerError)
			return
		}

	default:
		http.Error(w, "ERROR: unknown metric type", http.StatusBadRequest)
		return
	}

	// Синхронное сохранение только для файлового хранилища
	if h.syncSave && h.fileSaver != nil {
		if err := h.fileService.SaveSync(); err != nil {
			h.logger.Error("Failed to save metrics synchronously", zap.Error(err))
		} else {
			h.logger.Info("Metrics saved synchronously")
		}
	}

	w.WriteHeader(http.StatusOK)
}

func (h *Handler) getMetric(w http.ResponseWriter, r *http.Request) {
	metricType := chi.URLParam(r, "type")
	metricName := chi.URLParam(r, "name")

	ctx := r.Context()

	metric, err := h.storage.GetMetric(ctx, metricType, metricName)
	if err != nil {
		h.logger.Warn("Metric not found", zap.String("type", metricType), zap.String("name", metricName), zap.Error(err))
		http.NotFound(w, r)
		return
	}

	switch metric.MType {
	case model.Gauge:
		fmt.Fprintf(w, "%g", *metric.Value)
	case model.Counter:
		fmt.Fprintf(w, "%d", *metric.Delta)
	default:
		http.Error(w, "ERROR: unknown metric type", http.StatusInternalServerError)
	}
}

func (h *Handler) listMetrics(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	metrics, err := h.storage.GetAll(ctx)
	if err != nil {
		h.logger.Error("Failed to get all metrics", zap.Error(err))
		http.Error(w, "ERROR: failed to retrieve metrics", http.StatusInternalServerError)
		return
	}

	initTemplates()

	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)

	if err := metricsTemplate.Execute(w, metrics); err != nil {
		h.logger.Error("Failed to execute template", zap.Error(err))
		http.Error(w, "ERROR: failed to render metrics", http.StatusInternalServerError)
	}
}

func (h *Handler) updateMetricJSON(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Content-Type") != "application/json" {
		http.Error(w, "ERROR: Content-Type must be application/json", http.StatusBadRequest)
		return
	}

	var metric model.Metrics

	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&metric); err != nil {
		h.logger.Error("Failed to decode JSON", zap.Error(err))
		http.Error(w, "ERROR: invalid JSON", http.StatusBadRequest)
		return
	}

	if metric.ID == "" {
		http.Error(w, "ERROR: metric ID is required", http.StatusBadRequest)
		return
	}

	ctx := r.Context()

	switch metric.MType {
	case model.Gauge:
		if metric.Value == nil {
			http.Error(w, "ERROR: value is required for gauge", http.StatusBadRequest)
			return
		}
		if err := h.storage.UpdateGauge(ctx, metric.ID, *metric.Value); err != nil {
			h.logger.Error("Failed to update gauge via JSON", zap.Error(err))
			http.Error(w, "ERROR: failed to update gauge", http.StatusInternalServerError)
			return
		}

	case model.Counter:
		if metric.Delta == nil {
			http.Error(w, "ERROR: delta is required for counter", http.StatusBadRequest)
			return
		}
		if err := h.storage.UpdateCounter(ctx, metric.ID, *metric.Delta); err != nil {
			h.logger.Error("Failed to update counter via JSON", zap.Error(err))
			http.Error(w, "ERROR: failed to update counter", http.StatusInternalServerError)
			return
		}

	default:
		http.Error(w, "ERROR: unknown metric type", http.StatusBadRequest)
		return
	}

	// Синхронное сохранение только для файлового хранилища
	if h.syncSave && h.fileSaver != nil {
		if err := h.fileService.SaveSync(); err != nil {
			h.logger.Error("Failed to save metrics synchronously", zap.Error(err))
		} else {
			h.logger.Info("Metrics saved synchronously")
		}
	}

	// Возвращаем обновленную метрику
	updatedMetric, err := h.storage.GetMetric(ctx, metric.MType, metric.ID)
	if err != nil {
		h.logger.Error("Failed to get updated metric", zap.Error(err))
		http.Error(w, "ERROR: failed to get updated metric", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(updatedMetric); err != nil {
		h.logger.Error("Failed to encode JSON response", zap.Error(err))
		http.Error(w, "ERROR: failed to encode response", http.StatusInternalServerError)
	}
}

func (h *Handler) getMetricJSON(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Content-Type") != "application/json" {
		http.Error(w, "ERROR: Content-Type must be application/json", http.StatusBadRequest)
		return
	}

	var metric model.Metrics

	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&metric); err != nil {
		h.logger.Error("Failed to decode JSON", zap.Error(err))
		http.Error(w, "ERROR: invalid JSON", http.StatusBadRequest)
		return
	}

	if metric.ID == "" || metric.MType == "" {
		http.Error(w, "ERROR: metric ID and type are required", http.StatusBadRequest)
		return
	}

	ctx := r.Context()

	foundMetric, err := h.storage.GetMetric(ctx, metric.MType, metric.ID)
	if err != nil {
		h.logger.Warn("Metric not found via JSON", zap.String("type", metric.MType), zap.String("id", metric.ID), zap.Error(err))
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(foundMetric); err != nil {
		h.logger.Error("Failed to encode JSON response", zap.Error(err))
		http.Error(w, "ERROR: failed to encode response", http.StatusInternalServerError)
	}
}

func NewHandler(storage Storage, fileService *service.FileStorageService, syncSave bool) http.Handler {
	// Инициализируем логгер
	err := middleware.InitLogger()
	if err != nil {
		panic(fmt.Sprintf("Failed to initialize logger: %v", err))
	}

	// Получаем логгер
	logger := middleware.GetLogger()
	sugar := middleware.GetSugar()

	handler := &Handler{
		storage:     storage,
		fileService: fileService,
		syncSave:    syncSave,
		logger:      logger,
		sugar:       sugar,
	}

	// Проверяем, поддерживает ли хранилище файловые операции
	if fileSaver, ok := storage.(FileSaver); ok {
		handler.fileSaver = fileSaver
	}

	router := chi.NewRouter()

	// Middleware
	router.Use(middleware.GzipDecompressionMiddleware)
	router.Use(middleware.LoggingMiddleware)
	router.Use(middleware.GzipCompressionMiddleware)

	// Старые эндпоинты
	router.Post("/update/{type}/{name}/{value}", handler.updateMetric)
	router.Get("/value/{type}/{name}", handler.getMetric)
	router.Get("/", handler.listMetrics)

	// Новые JSON эндпоинты
	router.Post("/update/", handler.updateMetricJSON)
	router.Post("/value/", handler.getMetricJSON)

	// Проверка соединения с БД
	router.Get("/ping", handler.pingDB)

	return router
}
