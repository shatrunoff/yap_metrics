package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestInitLogger(t *testing.T) {
	err := InitLogger()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if Logger == nil {
		t.Error("expected Logger to be initialized")
	}
	if Sugar == nil {
		t.Error("expected Sugar to be initialized")
	}
}

func TestGetLogger(t *testing.T) {
	InitLogger()
	logger := GetLogger()
	if logger == nil {
		t.Error("expected non-nil logger")
	}
}

func TestGetSugar(t *testing.T) {
	InitLogger()
	sugar := GetSugar()
	if sugar == nil {
		t.Error("expected non-nil sugar")
	}
}

func TestLoggingMiddleware(t *testing.T) {
	InitLogger()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test response"))
	})

	middleware := LoggingMiddleware(handler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	middleware.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if w.Body.String() != "test response" {
		t.Errorf("expected body 'test response', got %s", w.Body.String())
	}
}

func TestLoggingMiddleware_NilSugar(t *testing.T) {
	// Сохраняем текущий Sugar
	oldSugar := Sugar
	Sugar = nil
	defer func() { Sugar = oldSugar }()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test"))
	})

	middleware := LoggingMiddleware(handler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	middleware.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestResponseWriter_WriteHeader(t *testing.T) {
	w := httptest.NewRecorder()
	rw := &responseWriter{ResponseWriter: w}

	rw.WriteHeader(http.StatusCreated)

	if rw.status != http.StatusCreated {
		t.Errorf("expected status %d, got %d", http.StatusCreated, rw.status)
	}
	if !rw.wroteHeader {
		t.Error("expected wroteHeader to be true")
	}
}

func TestResponseWriter_WriteHeader_Multiple(t *testing.T) {
	w := httptest.NewRecorder()
	rw := &responseWriter{ResponseWriter: w}

	rw.WriteHeader(http.StatusCreated)
	rw.WriteHeader(http.StatusBadRequest)

	if rw.status != http.StatusCreated {
		t.Errorf("expected status to remain %d, got %d", http.StatusCreated, rw.status)
	}
}

func TestResponseWriter_Write(t *testing.T) {
	w := httptest.NewRecorder()
	rw := &responseWriter{ResponseWriter: w}

	data := []byte("test data")
	n, err := rw.Write(data)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if n != len(data) {
		t.Errorf("expected to write %d bytes, wrote %d", len(data), n)
	}
	if rw.size != len(data) {
		t.Errorf("expected size %d, got %d", len(data), rw.size)
	}
	if !rw.wroteHeader {
		t.Error("expected wroteHeader to be true after Write")
	}
	if rw.status != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rw.status)
	}
}

func TestResponseWriter_Write_WithExplicitStatus(t *testing.T) {
	w := httptest.NewRecorder()
	rw := &responseWriter{ResponseWriter: w}

	rw.WriteHeader(http.StatusBadRequest)
	data := []byte("test data")
	rw.Write(data)

	if rw.status != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rw.status)
	}
}
