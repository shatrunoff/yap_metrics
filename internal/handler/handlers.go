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
	"github.com/shatrunoff/yap_metrics/internal/storage"
	"go.uber.org/zap"
)

const htmlPage = `
<!DOCTYPE html>
<html>
<head>
    <title>Metrics</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 40px; }
        h1 { color: #333; }
        ul { list-style-type: none; padding: 0; }
        li { padding: 8px; margin: 4px; background: #f5f5f5; border-radius: 4px; }
        .metric-type { font-weight: bold; color: #555; }
    </style>
</head>
<body>
    <h1>Metrics</h1>
    <ul>
        {{range $name, $metric := .}}
            <li>
                <span class="metric-type">{{$metric.MType}}</span> {{$metric.ID}}:
                {{if eq $metric.MType "gauge"}}{{$metric.Value}}{{end}}
                {{if eq $metric.MType "counter"}}{{$metric.Delta}}{{end}}
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

type Handler struct {
	storage     storage.Storage
	fileService *service.FileStorageService
	syncSave    bool
	logger      *zap.Logger
	sugar       *zap.SugaredLogger
	fileSaver   interface {
		SaveToFile(path string) error
		LoadFromFile(filename string) error
	}
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

// хэндлер обновления метрики
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

// хэндлер получения метрики
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

// хэндлер получения всех метрик
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

// хэндлер обновления метрики через JSON
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

// хэндлер получения метрики через JSON
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

// хэндлер обновления метрик батчами
func (h *Handler) updateMetricsBatch(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Content-Type") != "application/json" {
		http.Error(w, "ERROR: Content-Type must be application/json", http.StatusBadRequest)
		return
	}

	var metrics []model.Metrics

	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&metrics); err != nil {
		h.logger.Error("Failed to decode JSON batch", zap.Error(err))
		http.Error(w, "ERROR: invalid JSON", http.StatusBadRequest)
		return
	}

	if len(metrics) == 0 {
		http.Error(w, "ERROR: empty metrics batch", http.StatusBadRequest)
		return
	}

	ctx := r.Context()

	// Используем batch, если хранилище поддерживает
	if batchStorage, ok := h.storage.(interface {
		UpdateMetricsBatch(ctx context.Context, metrics []model.Metrics) error
	}); ok {
		// batch метод
		if err := batchStorage.UpdateMetricsBatch(ctx, metrics); err != nil {
			h.logger.Error("Failed to update metrics batch", zap.Error(err))
			http.Error(w, "ERROR: failed to update metrics batch", http.StatusInternalServerError)
			return
		}
	} else {
		// обрабатываем по одной метрике
		var hasErrors bool
		for _, metric := range metrics {
			if metric.ID == "" {
				continue
			}

			var err error
			switch metric.MType {
			case model.Gauge:
				if metric.Value != nil {
					err = h.storage.UpdateGauge(ctx, metric.ID, *metric.Value)
				}

			case model.Counter:
				if metric.Delta != nil {
					err = h.storage.UpdateCounter(ctx, metric.ID, *metric.Delta)
				}
			}

			// обработка остальных метрик и логируем ошибку
			if err != nil {
				h.logger.Error("Failed to update metric in batch",
					zap.String("id", metric.ID),
					zap.String("type", metric.MType),
					zap.Error(err))
				hasErrors = true
			}
		}

		if hasErrors {
			http.Error(w, "ERROR: some metrics failed to update", http.StatusInternalServerError)
			return
		}
	}

	// Синхронное сохранение только для файлового хранилища
	if h.syncSave && h.fileSaver != nil {
		if err := h.fileService.SaveSync(); err != nil {
			h.logger.Error("Failed to save metrics synchronously after batch", zap.Error(err))
		} else {
			h.logger.Info("Metrics saved synchronously after batch update")
		}
	}

	w.WriteHeader(http.StatusOK)
}

// основной хэндлер
func NewHandler(storage storage.Storage, fileService *service.FileStorageService, syncSave bool) http.Handler {
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
	if fileSaver, ok := storage.(interface {
		SaveToFile(path string) error
		LoadFromFile(filename string) error
	}); ok {
		handler.fileSaver = fileSaver
	}

	router := chi.NewRouter()

	// Middleware
	router.Use(middleware.GzipDecompressionMiddleware)
	router.Use(middleware.LoggingMiddleware)
	router.Use(middleware.GzipCompressionMiddleware)

	// эндпоинты единичных запросов
	router.Post("/update/{type}/{name}/{value}", handler.updateMetric)
	router.Get("/value/{type}/{name}", handler.getMetric)
	router.Get("/", handler.listMetrics)

	// JSON эндпоинты
	router.Post("/update/", handler.updateMetricJSON)
	router.Post("/value/", handler.getMetricJSON)
	router.Post("/updates/", handler.updateMetricsBatch)

	// Проверка соединения с БД
	router.Get("/ping", handler.pingDB)

	return router
}
