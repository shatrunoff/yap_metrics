package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/shatrunoff/yap_metrics/internal/model"
	"github.com/shatrunoff/yap_metrics/internal/storage"
)

func TestHandler_updateMetric(t *testing.T) {
	tests := []struct {
		name           string
		metricType     string
		metricName     string
		metricValue    string
		expectedStatus int
	}{
		{
			name:           "valid gauge metric",
			metricType:     "gauge",
			metricName:     "temperature",
			metricValue:    "25.5",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "valid counter metric",
			metricType:     "counter",
			metricName:     "requests",
			metricValue:    "100",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "invalid gauge value",
			metricType:     "gauge",
			metricName:     "temperature",
			metricValue:    "invalid",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid counter value",
			metricType:     "counter",
			metricName:     "requests",
			metricValue:    "invalid",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "unknown metric type",
			metricType:     "unknown",
			metricName:     "test",
			metricValue:    "100",
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			st := storage.NewMemStorage()
			h := NewHandler(st, nil, false, "", nil, "")

			url := "/update/" + tt.metricType + "/" + tt.metricName + "/" + tt.metricValue
			req := httptest.NewRequest(http.MethodPost, url, nil)
			w := httptest.NewRecorder()

			h.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestHandler_getMetric(t *testing.T) {
	tests := []struct {
		name           string
		setupMetric    func(st storage.Storage)
		metricType     string
		metricName     string
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "get existing gauge metric",
			setupMetric: func(st storage.Storage) {
				st.UpdateGauge(context.Background(), "temperature", 25.5)
			},
			metricType:     "gauge",
			metricName:     "temperature",
			expectedStatus: http.StatusOK,
			expectedBody:   "25.5",
		},
		{
			name: "get existing counter metric",
			setupMetric: func(st storage.Storage) {
				st.UpdateCounter(context.Background(), "requests", 100)
			},
			metricType:     "counter",
			metricName:     "requests",
			expectedStatus: http.StatusOK,
			expectedBody:   "100",
		},
		{
			name:           "get non-existing metric",
			setupMetric:    func(st storage.Storage) {},
			metricType:     "gauge",
			metricName:     "nonexistent",
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			st := storage.NewMemStorage()
			tt.setupMetric(st)
			h := NewHandler(st, nil, false, "", nil, "")

			url := "/value/" + tt.metricType + "/" + tt.metricName
			req := httptest.NewRequest(http.MethodGet, url, nil)
			w := httptest.NewRecorder()

			h.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.expectedBody != "" && w.Body.String() != tt.expectedBody {
				t.Errorf("expected body %s, got %s", tt.expectedBody, w.Body.String())
			}
		})
	}
}

func TestHandler_updateMetricJSON(t *testing.T) {
	tests := []struct {
		name           string
		contentType    string
		metric         model.Metrics
		expectedStatus int
	}{
		{
			name:        "valid gauge metric",
			contentType: "application/json",
			metric: model.Metrics{
				ID:    "cpu",
				MType: "gauge",
				Value: func() *float64 { v := 75.5; return &v }(),
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:        "valid counter metric",
			contentType: "application/json",
			metric: model.Metrics{
				ID:    "requests",
				MType: "counter",
				Delta: func() *int64 { d := int64(10); return &d }(),
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:        "missing content type",
			contentType: "",
			metric: model.Metrics{
				ID:    "test",
				MType: "gauge",
				Value: func() *float64 { v := 1.0; return &v }(),
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:        "gauge without value",
			contentType: "application/json",
			metric: model.Metrics{
				ID:    "test",
				MType: "gauge",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:        "counter without delta",
			contentType: "application/json",
			metric: model.Metrics{
				ID:    "test",
				MType: "counter",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:        "unknown metric type",
			contentType: "application/json",
			metric: model.Metrics{
				ID:    "test",
				MType: "unknown",
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			st := storage.NewMemStorage()
			h := NewHandler(st, nil, false, "", nil, "")

			body, _ := json.Marshal(tt.metric)
			req := httptest.NewRequest(http.MethodPost, "/update/", bytes.NewReader(body))
			if tt.contentType != "" {
				req.Header.Set("Content-Type", tt.contentType)
			}
			w := httptest.NewRecorder()

			h.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestHandler_getMetricJSON(t *testing.T) {
	tests := []struct {
		name           string
		setupMetric    func(st storage.Storage)
		contentType    string
		requestMetric  model.Metrics
		expectedStatus int
	}{
		{
			name: "get existing gauge",
			setupMetric: func(st storage.Storage) {
				st.UpdateGauge(context.Background(), "cpu", 75.5)
			},
			contentType: "application/json",
			requestMetric: model.Metrics{
				ID:    "cpu",
				MType: "gauge",
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:        "get non-existing metric",
			setupMetric: func(st storage.Storage) {},
			contentType: "application/json",
			requestMetric: model.Metrics{
				ID:    "nonexistent",
				MType: "gauge",
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:        "missing content type",
			setupMetric: func(st storage.Storage) {},
			contentType: "",
			requestMetric: model.Metrics{
				ID:    "test",
				MType: "gauge",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:        "missing metric ID",
			setupMetric: func(st storage.Storage) {},
			contentType: "application/json",
			requestMetric: model.Metrics{
				MType: "gauge",
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			st := storage.NewMemStorage()
			tt.setupMetric(st)
			h := NewHandler(st, nil, false, "", nil, "")

			body, _ := json.Marshal(tt.requestMetric)
			req := httptest.NewRequest(http.MethodPost, "/value/", bytes.NewReader(body))
			if tt.contentType != "" {
				req.Header.Set("Content-Type", tt.contentType)
			}
			w := httptest.NewRecorder()

			h.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestHandler_updateMetricsBatch(t *testing.T) {
	tests := []struct {
		name           string
		contentType    string
		metrics        []model.Metrics
		expectedStatus int
	}{
		{
			name:        "valid batch",
			contentType: "application/json",
			metrics: []model.Metrics{
				{
					ID:    "cpu",
					MType: "gauge",
					Value: func() *float64 { v := 75.5; return &v }(),
				},
				{
					ID:    "requests",
					MType: "counter",
					Delta: func() *int64 { d := int64(10); return &d }(),
				},
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "empty batch",
			contentType:    "application/json",
			metrics:        []model.Metrics{},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "missing content type",
			contentType:    "",
			metrics:        []model.Metrics{},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			st := storage.NewMemStorage()
			h := NewHandler(st, nil, false, "", nil, "")

			body, _ := json.Marshal(tt.metrics)
			req := httptest.NewRequest(http.MethodPost, "/updates/", bytes.NewReader(body))
			if tt.contentType != "" {
				req.Header.Set("Content-Type", tt.contentType)
			}
			w := httptest.NewRecorder()

			h.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestHandler_listMetrics(t *testing.T) {
	st := storage.NewMemStorage()
	st.UpdateGauge(context.Background(), "cpu", 75.5)
	st.UpdateCounter(context.Background(), "requests", 100)

	h := NewHandler(st, nil, false, "", nil, "")

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	if contentType := w.Header().Get("Content-Type"); contentType != "text/html" {
		t.Errorf("expected Content-Type text/html, got %s", contentType)
	}
}

func TestHandler_pingDB(t *testing.T) {
	st := storage.NewMemStorage()
	h := NewHandler(st, nil, false, "", nil, "")

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	if body := w.Body.String(); body != "DB connection OK" {
		t.Errorf("expected body 'DB connection OK', got %s", body)
	}
}

func Test_getClientIP(t *testing.T) {
	tests := []struct {
		name          string
		remoteAddr    string
		xForwardedFor string
		xRealIP       string
		expectedIP    string
	}{
		{
			name:          "from X-Forwarded-For",
			remoteAddr:    "192.168.1.1:12345",
			xForwardedFor: "10.0.0.1, 10.0.0.2",
			xRealIP:       "",
			expectedIP:    "10.0.0.1",
		},
		{
			name:          "from X-Real-IP",
			remoteAddr:    "192.168.1.1:12345",
			xForwardedFor: "",
			xRealIP:       "10.0.0.1",
			expectedIP:    "10.0.0.1",
		},
		{
			name:          "from RemoteAddr",
			remoteAddr:    "192.168.1.1:12345",
			xForwardedFor: "",
			xRealIP:       "",
			expectedIP:    "192.168.1.1",
		},
		{
			name:          "RemoteAddr without port",
			remoteAddr:    "192.168.1.1",
			xForwardedFor: "",
			xRealIP:       "",
			expectedIP:    "192.168.1.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.RemoteAddr = tt.remoteAddr
			if tt.xForwardedFor != "" {
				req.Header.Set("X-Forwarded-For", tt.xForwardedFor)
			}
			if tt.xRealIP != "" {
				req.Header.Set("X-Real-IP", tt.xRealIP)
			}

			ip := getClientIP(req)
			if ip != tt.expectedIP {
				t.Errorf("expected IP %s, got %s", tt.expectedIP, ip)
			}
		})
	}
}

func Test_initTemplates(t *testing.T) {
	// Просто вызываем функцию, чтобы увеличить покрытие
	initTemplates()
	if metricsTemplate == nil {
		t.Error("expected template to be initialized")
	}
}
