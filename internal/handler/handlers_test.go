package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
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

func TestHandler_updateMetricJSON_Errors(t *testing.T) {
	st := storage.NewMemStorage()
	h := NewHandler(st, nil, false, "", nil, "")

	tests := []struct {
		name           string
		contentType    string
		body           string
		expectedStatus int
	}{
		{
			name:           "wrong content type",
			contentType:    "text/plain",
			body:           `{"id":"test","type":"gauge","value":1.0}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid JSON",
			contentType:    "application/json",
			body:           `invalid json`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "missing ID",
			contentType:    "application/json",
			body:           `{"type":"gauge","value":1.0}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "unknown type",
			contentType:    "application/json",
			body:           `{"id":"test","type":"unknown","value":1.0}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "gauge without value",
			contentType:    "application/json",
			body:           `{"id":"test","type":"gauge"}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "counter without delta",
			contentType:    "application/json",
			body:           `{"id":"test","type":"counter"}`,
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/update/", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", tt.contentType)
			w := httptest.NewRecorder()

			h.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d, body: %s", tt.expectedStatus, w.Code, w.Body.String())
			}
		})
	}
}

func TestHandler_getMetricJSON_Errors(t *testing.T) {
	st := storage.NewMemStorage()
	h := NewHandler(st, nil, false, "", nil, "")

	tests := []struct {
		name           string
		contentType    string
		body           string
		expectedStatus int
	}{
		{
			name:           "wrong content type",
			contentType:    "text/plain",
			body:           `{"id":"test","type":"gauge"}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid JSON",
			contentType:    "application/json",
			body:           `invalid`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "missing ID",
			contentType:    "application/json",
			body:           `{"type":"gauge"}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "missing type",
			contentType:    "application/json",
			body:           `{"id":"test"}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "metric not found",
			contentType:    "application/json",
			body:           `{"id":"nonexistent","type":"gauge"}`,
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/value/", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", tt.contentType)
			w := httptest.NewRecorder()

			h.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestHandler_updateMetricsBatch_Errors(t *testing.T) {
	st := storage.NewMemStorage()
	h := NewHandler(st, nil, false, "", nil, "")

	tests := []struct {
		name           string
		contentType    string
		body           string
		expectedStatus int
	}{
		{
			name:           "wrong content type",
			contentType:    "text/plain",
			body:           `[{"id":"test","type":"gauge","value":1.0}]`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid JSON",
			contentType:    "application/json",
			body:           `invalid`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "empty batch",
			contentType:    "application/json",
			body:           `[]`,
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/updates/", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", tt.contentType)
			w := httptest.NewRecorder()

			h.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestHandler_updateMetricsBatch_Success(t *testing.T) {
	st := storage.NewMemStorage()
	h := NewHandler(st, nil, false, "", nil, "")

	metrics := []model.Metrics{
		{ID: "g1", MType: "gauge", Value: func() *float64 { v := 1.1; return &v }()},
		{ID: "c1", MType: "counter", Delta: func() *int64 { d := int64(100); return &d }()},
	}

	body, _ := json.Marshal(metrics)
	req := httptest.NewRequest(http.MethodPost, "/updates/", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	// Verify metrics were saved
	ctx := context.Background()
	m, _ := st.GetMetric(ctx, "gauge", "g1")
	if m.Value == nil || *m.Value != 1.1 {
		t.Errorf("Expected g1=1.1, got %v", m.Value)
	}
}

func TestHandler_listMetrics_WithData(t *testing.T) {
	st := storage.NewMemStorage()
	ctx := context.Background()
	st.UpdateGauge(ctx, "g1", 1.1)
	st.UpdateCounter(ctx, "c1", 100)

	h := NewHandler(st, nil, false, "", nil, "")

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	if w.Header().Get("Content-Type") != "text/html" {
		t.Errorf("expected Content-Type text/html, got %s", w.Header().Get("Content-Type"))
	}

	body := w.Body.String()
	if len(body) == 0 {
		t.Error("expected non-empty body")
	}
}

func TestHandler_pingDB_MemStorage(t *testing.T) {
	st := storage.NewMemStorage()
	h := NewHandler(st, nil, false, "", nil, "")

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestGetClientIP(t *testing.T) {
	tests := []struct {
		name       string
		headers    map[string]string
		remoteAddr string
		expected   string
	}{
		{
			name:       "from X-Forwarded-For",
			headers:    map[string]string{"X-Forwarded-For": "192.168.1.1, 10.0.0.1"},
			remoteAddr: "127.0.0.1:8080",
			expected:   "192.168.1.1",
		},
		{
			name:       "from X-Real-IP",
			headers:    map[string]string{"X-Real-IP": "192.168.1.2"},
			remoteAddr: "127.0.0.1:8080",
			expected:   "192.168.1.2",
		},
		{
			name:       "from RemoteAddr",
			headers:    map[string]string{},
			remoteAddr: "192.168.1.3:8080",
			expected:   "192.168.1.3",
		},
		{
			name:       "from RemoteAddr without port",
			headers:    map[string]string{},
			remoteAddr: "192.168.1.4",
			expected:   "192.168.1.4",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}
			req.RemoteAddr = tt.remoteAddr

			ip := getClientIP(req)
			if ip != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, ip)
			}
		})
	}
}

func TestHandler_updateMetricJSON_ValidGauge(t *testing.T) {
	st := storage.NewMemStorage()
	h := NewHandler(st, nil, false, "", nil, "")

	body := `{"id":"test_gauge","type":"gauge","value":42.5}`
	req := httptest.NewRequest(http.MethodPost, "/update/", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	// Verify response contains updated metric
	var resp model.Metrics
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Value == nil || *resp.Value != 42.5 {
		t.Errorf("Expected value 42.5, got %v", resp.Value)
	}
}

func TestHandler_updateMetricJSON_ValidCounter(t *testing.T) {
	st := storage.NewMemStorage()
	h := NewHandler(st, nil, false, "", nil, "")

	body := `{"id":"test_counter","type":"counter","delta":100}`
	req := httptest.NewRequest(http.MethodPost, "/update/", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var resp model.Metrics
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Delta == nil || *resp.Delta != 100 {
		t.Errorf("Expected delta 100, got %v", resp.Delta)
	}
}

func TestHandler_getMetricJSON_ValidGauge(t *testing.T) {
	st := storage.NewMemStorage()
	ctx := context.Background()
	st.UpdateGauge(ctx, "test_gauge", 123.45)

	h := NewHandler(st, nil, false, "", nil, "")

	body := `{"id":"test_gauge","type":"gauge"}`
	req := httptest.NewRequest(http.MethodPost, "/value/", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var resp model.Metrics
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Value == nil || *resp.Value != 123.45 {
		t.Errorf("Expected value 123.45, got %v", resp.Value)
	}
}

func TestHandler_updateMetric_WithSyncSave(t *testing.T) {
	// Skip - syncSave requires fileService to be properly initialized
	t.Skip("Requires proper fileService initialization")
}

func TestHandler_getMetric_Counter(t *testing.T) {
	st := storage.NewMemStorage()
	ctx := context.Background()
	st.UpdateCounter(ctx, "test_counter", 100)

	h := NewHandler(st, nil, false, "", nil, "")

	req := httptest.NewRequest(http.MethodGet, "/value/counter/test_counter", nil)
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	if w.Body.String() != "100" {
		t.Errorf("Expected body '100', got '%s'", w.Body.String())
	}
}

func TestHandler_updateMetric_AllBranches(t *testing.T) {
	st := storage.NewMemStorage()
	h := NewHandler(st, nil, false, "", nil, "")

	// Test gauge update and verify storage
	req := httptest.NewRequest(http.MethodPost, "/update/gauge/test_g/123.45", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("gauge update: expected %d, got %d", http.StatusOK, w.Code)
	}

	// Test counter update
	req = httptest.NewRequest(http.MethodPost, "/update/counter/test_c/100", nil)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("counter update: expected %d, got %d", http.StatusOK, w.Code)
	}
}

func TestHandler_listMetrics_Empty(t *testing.T) {
	st := storage.NewMemStorage()
	h := NewHandler(st, nil, false, "", nil, "")

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected %d, got %d", http.StatusOK, w.Code)
	}
	if !bytes.Contains(w.Body.Bytes(), []byte("<html>")) {
		t.Error("Expected HTML response")
	}
}

func TestHandler_updateMetricJSON_AllTypes(t *testing.T) {
	st := storage.NewMemStorage()
	h := NewHandler(st, nil, false, "", nil, "")

	tests := []struct {
		name string
		body string
	}{
		{"gauge", `{"id":"g1","type":"gauge","value":1.5}`},
		{"counter", `{"id":"c1","type":"counter","delta":10}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/update/", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			h.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("expected %d, got %d", http.StatusOK, w.Code)
			}
		})
	}
}

func TestHandler_updateMetricsBatch_AllTypes(t *testing.T) {
	st := storage.NewMemStorage()
	h := NewHandler(st, nil, false, "", nil, "")

	body := `[{"id":"g1","type":"gauge","value":1.5},{"id":"c1","type":"counter","delta":10}]`
	req := httptest.NewRequest(http.MethodPost, "/updates/", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected %d, got %d", http.StatusOK, w.Code)
	}

	// Verify metrics saved
	ctx := context.Background()
	m, _ := st.GetMetric(ctx, "gauge", "g1")
	if m.Value == nil || *m.Value != 1.5 {
		t.Errorf("Expected g1=1.5")
	}
	m, _ = st.GetMetric(ctx, "counter", "c1")
	if m.Delta == nil || *m.Delta != 10 {
		t.Errorf("Expected c1=10")
	}
}

func TestHandler_getMetric_UnknownType(t *testing.T) {
	st := storage.NewMemStorage()
	h := NewHandler(st, nil, false, "", nil, "")

	req := httptest.NewRequest(http.MethodGet, "/value/unknown/test", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected %d, got %d", http.StatusNotFound, w.Code)
	}
}

// mockErrorStorage для тестирования error branches
type mockErrorStorage struct {
	storage.Storage
}

func (m *mockErrorStorage) UpdateGauge(ctx context.Context, name string, value float64) error {
	return errors.New("storage error")
}
func (m *mockErrorStorage) UpdateCounter(ctx context.Context, name string, delta int64) error {
	return errors.New("storage error")
}
func (m *mockErrorStorage) GetMetric(ctx context.Context, mtype, name string) (model.Metrics, error) {
	return model.Metrics{}, errors.New("storage error")
}
func (m *mockErrorStorage) GetAll(ctx context.Context) (map[string]model.Metrics, error) {
	return nil, errors.New("storage error")
}
func (m *mockErrorStorage) UpdateMetricsBatch(ctx context.Context, metrics []model.Metrics) error {
	return errors.New("storage error")
}
func (m *mockErrorStorage) Ping(ctx context.Context) error {
	return errors.New("ping error")
}
func (m *mockErrorStorage) Close() error { return nil }

func TestHandler_updateMetric_StorageError(t *testing.T) {
	st := &mockErrorStorage{}
	h := NewHandler(st, nil, false, "", nil, "")

	// Gauge error
	req := httptest.NewRequest(http.MethodPost, "/update/gauge/test/1.0", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected %d, got %d", http.StatusInternalServerError, w.Code)
	}

	// Counter error
	req = httptest.NewRequest(http.MethodPost, "/update/counter/test/1", nil)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected %d, got %d", http.StatusInternalServerError, w.Code)
	}
}

func TestHandler_updateMetricJSON_StorageError(t *testing.T) {
	st := &mockErrorStorage{}
	h := NewHandler(st, nil, false, "", nil, "")

	// Gauge error
	body := `{"id":"test","type":"gauge","value":1.0}`
	req := httptest.NewRequest(http.MethodPost, "/update/", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected %d, got %d", http.StatusInternalServerError, w.Code)
	}

	// Counter error
	body = `{"id":"test","type":"counter","delta":1}`
	req = httptest.NewRequest(http.MethodPost, "/update/", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected %d, got %d", http.StatusInternalServerError, w.Code)
	}
}

func TestHandler_updateMetricsBatch_StorageError(t *testing.T) {
	st := &mockErrorStorage{}
	h := NewHandler(st, nil, false, "", nil, "")

	body := `[{"id":"test","type":"gauge","value":1.0}]`
	req := httptest.NewRequest(http.MethodPost, "/updates/", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected %d, got %d", http.StatusInternalServerError, w.Code)
	}
}

func TestHandler_listMetrics_StorageError(t *testing.T) {
	st := &mockErrorStorage{}
	h := NewHandler(st, nil, false, "", nil, "")

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected %d, got %d", http.StatusInternalServerError, w.Code)
	}
}

func TestHandler_pingDB_Error(t *testing.T) {
	st := &mockErrorStorage{}
	h := NewHandler(st, nil, false, "", nil, "")

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected %d, got %d", http.StatusInternalServerError, w.Code)
	}
}
