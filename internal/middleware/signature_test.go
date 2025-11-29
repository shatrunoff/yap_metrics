package middleware

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/shatrunoff/yap_metrics/internal/utils"
)

func TestSignatureMiddleware_NoKey(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	middleware := SignatureMiddleware("")(handler)

	req := httptest.NewRequest(http.MethodPost, "/update/", bytes.NewReader([]byte(`{"test":"data"}`)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	middleware.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestSignatureMiddleware_ValidHash(t *testing.T) {
	key := "secret"
	body := []byte(`{"test":"data"}`)
	hash := utils.ComputeHMACSHA256(body, key)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	middleware := SignatureMiddleware(key)(handler)

	req := httptest.NewRequest(http.MethodPost, "/update/", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("HashSHA256", hash)
	w := httptest.NewRecorder()

	middleware.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	if w.Header().Get("HashSHA256") == "" {
		t.Error("expected HashSHA256 header in response")
	}
}

func TestSignatureMiddleware_InvalidHash(t *testing.T) {
	key := "secret"
	body := []byte(`{"test":"data"}`)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := SignatureMiddleware(key)(handler)

	req := httptest.NewRequest(http.MethodPost, "/update/", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("HashSHA256", "invalid_hash")
	w := httptest.NewRecorder()

	middleware.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestSignatureMiddleware_MissingHashForBatch(t *testing.T) {
	key := "secret"
	body := []byte(`[{"test":"data"}]`)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := SignatureMiddleware(key)(handler)

	req := httptest.NewRequest(http.MethodPost, "/updates/", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	middleware.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestSignatureMiddleware_OptionalHashForSingle(t *testing.T) {
	key := "secret"
	body := []byte(`{"test":"data"}`)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	middleware := SignatureMiddleware(key)(handler)

	req := httptest.NewRequest(http.MethodPost, "/update/", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	middleware.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestSignatureMiddleware_NonJSON(t *testing.T) {
	key := "secret"

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("plain text"))
	})

	middleware := SignatureMiddleware(key)(handler)

	req := httptest.NewRequest(http.MethodPost, "/update/", bytes.NewReader([]byte("plain text")))
	req.Header.Set("Content-Type", "text/plain")
	w := httptest.NewRecorder()

	middleware.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	if w.Header().Get("HashSHA256") != "" {
		t.Error("unexpected HashSHA256 header for non-JSON response")
	}
}

func TestSigningResponseWriter_Header(t *testing.T) {
	w := httptest.NewRecorder()
	srw := &signingResponseWriter{ResponseWriter: w}

	srw.Header().Set("Test-Header", "value")

	if srw.Header().Get("Test-Header") != "value" {
		t.Error("expected header to be set")
	}
}

func TestSigningResponseWriter_Write(t *testing.T) {
	w := httptest.NewRecorder()
	srw := &signingResponseWriter{ResponseWriter: w}

	srw.Header().Set("Content-Type", "application/json")
	srw.Write([]byte("test"))

	if srw.buf.String() != "test" {
		t.Errorf("expected buffer to contain 'test', got %s", srw.buf.String())
	}
}

func TestSigningResponseWriter_WriteNonJSON(t *testing.T) {
	w := httptest.NewRecorder()
	srw := &signingResponseWriter{ResponseWriter: w}

	srw.Header().Set("Content-Type", "text/plain")
	srw.Write([]byte("test"))

	if w.Body.String() != "test" {
		t.Errorf("expected body to contain 'test', got %s", w.Body.String())
	}
}
