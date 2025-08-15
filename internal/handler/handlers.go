package handler

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"sync"
	"text/template"

	"github.com/go-chi/chi/v5"
	"github.com/shatrunoff/yap_metrics/internal/model"
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
}

// хэндлер обновления метрики
func (h *Handler) updateMetric(w http.ResponseWriter, r *http.Request) {

	// для отладки
	log.Printf("Incoming update: %s %s", r.Method, r.URL.Path)

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

// основной хэндлер
func NewHandler(storage Storage) http.Handler {
	handler := &Handler{storage: storage}
	router := chi.NewRouter()

	router.Post("/update/{type}/{name}/{value}", handler.updateMetric)
	router.Get("/value/{type}/{name}", handler.getMetric)
	router.Get("/", handler.listMetrics)

	return router

}
