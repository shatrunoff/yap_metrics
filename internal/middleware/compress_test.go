package middleware

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGzipCompressionMiddleware(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	tests := []struct {
		name           string
		acceptEncoding string
		expectGzip     bool
	}{
		{
			name:           "with gzip support",
			acceptEncoding: "gzip",
			expectGzip:     true,
		},
		{
			name:           "without gzip support",
			acceptEncoding: "",
			expectGzip:     false,
		},
		{
			name:           "with deflate only",
			acceptEncoding: "deflate",
			expectGzip:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.acceptEncoding != "" {
				req.Header.Set("Accept-Encoding", tt.acceptEncoding)
			}
			w := httptest.NewRecorder()

			middleware := GzipCompressionMiddleware(handler)
			middleware.ServeHTTP(w, req)

			if tt.expectGzip {
				if w.Header().Get("Content-Encoding") != "gzip" {
					t.Error("expected gzip encoding")
				}

				// Проверяем, что данные действительно сжаты
				gz, err := gzip.NewReader(w.Body)
				if err != nil {
					t.Fatalf("failed to create gzip reader: %v", err)
				}
				defer gz.Close()

				body, err := io.ReadAll(gz)
				if err != nil {
					t.Fatalf("failed to read gzip data: %v", err)
				}

				if string(body) != `{"status":"ok"}` {
					t.Errorf("unexpected body: %s", body)
				}
			} else {
				if w.Header().Get("Content-Encoding") == "gzip" {
					t.Error("unexpected gzip encoding")
				}

				if w.Body.String() != `{"status":"ok"}` {
					t.Errorf("unexpected body: %s", w.Body.String())
				}
			}
		})
	}
}

func TestGzipDecompressionMiddleware(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
		w.Write(body)
	})

	tests := []struct {
		name        string
		compressed  bool
		expectError bool
	}{
		{
			name:        "with gzip compression",
			compressed:  true,
			expectError: false,
		},
		{
			name:        "without compression",
			compressed:  false,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testBody := []byte("test data")
			var body io.Reader

			if tt.compressed {
				var buf bytes.Buffer
				gz := gzip.NewWriter(&buf)
				gz.Write(testBody)
				gz.Close()
				body = &buf
			} else {
				body = bytes.NewReader(testBody)
			}

			req := httptest.NewRequest(http.MethodPost, "/", body)
			if tt.compressed {
				req.Header.Set("Content-Encoding", "gzip")
			}
			w := httptest.NewRecorder()

			middleware := GzipDecompressionMiddleware(handler)
			middleware.ServeHTTP(w, req)

			if !tt.expectError && w.Body.String() != string(testBody) {
				t.Errorf("expected body %s, got %s", testBody, w.Body.String())
			}
		})
	}
}

func TestGzipDecompressionMiddleware_InvalidData(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader([]byte("invalid gzip data")))
	req.Header.Set("Content-Encoding", "gzip")
	w := httptest.NewRecorder()

	middleware := GzipDecompressionMiddleware(handler)
	middleware.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func Test_shouldCompress(t *testing.T) {
	tests := []struct {
		contentType string
		expected    bool
	}{
		{"application/json", true},
		{"application/json; charset=utf-8", true},
		{"text/html", true},
		{"text/html; charset=utf-8", true},
		{"text/plain", false},
		{"image/png", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.contentType, func(t *testing.T) {
			result := shouldCompress(tt.contentType)
			if result != tt.expected {
				t.Errorf("shouldCompress(%q) = %v, want %v", tt.contentType, result, tt.expected)
			}
		})
	}
}

func TestGzipResponseWriter_WriteHeader(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"status":"created"}`))
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	w := httptest.NewRecorder()

	middleware := GzipCompressionMiddleware(handler)
	middleware.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status %d, got %d", http.StatusCreated, w.Code)
	}

	if w.Header().Get("Content-Encoding") != "gzip" {
		t.Error("expected gzip encoding")
	}
}

func TestGzipResponseWriter_NonCompressibleContent(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("plain text"))
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	w := httptest.NewRecorder()

	middleware := GzipCompressionMiddleware(handler)
	middleware.ServeHTTP(w, req)

	if w.Header().Get("Content-Encoding") == "gzip" {
		t.Error("unexpected gzip encoding for non-compressible content")
	}

	if w.Body.String() != "plain text" {
		t.Errorf("unexpected body: %s", w.Body.String())
	}
}
