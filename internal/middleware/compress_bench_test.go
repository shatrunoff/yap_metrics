package middleware

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// BenchmarkGzipCompression измеряет производительность сжатия gzip
func BenchmarkGzipCompression(b *testing.B) {
	// Подготавливаем большой JSON ответ
	largeJSON := `{"metrics":[` + strings.Repeat(`{"id":"metric","type":"gauge","value":123.45},`, 1000) + `]}`

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(largeJSON))
	})

	middleware := GzipCompressionMiddleware(handler)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Accept-Encoding", "gzip")
		w := httptest.NewRecorder()
		middleware.ServeHTTP(w, req)
	}
}

// BenchmarkGzipDecompression измеряет производительность декомпрессии gzip
func BenchmarkGzipDecompression(b *testing.B) {
	// Подготавливаем сжатые данные
	data := []byte(`{"id":"test","type":"gauge","value":123.45}`)
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	gz.Write(data)
	gz.Close()
	compressedData := buf.Bytes()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		w.Write(body)
	})

	middleware := GzipDecompressionMiddleware(handler)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(compressedData))
		req.Header.Set("Content-Encoding", "gzip")
		w := httptest.NewRecorder()
		middleware.ServeHTTP(w, req)
	}
}

// BenchmarkGzipCompressionWithoutAcceptEncoding проверяет overhead без сжатия
func BenchmarkGzipCompressionWithoutAcceptEncoding(b *testing.B) {
	largeJSON := `{"metrics":[` + strings.Repeat(`{"id":"metric","type":"gauge","value":123.45},`, 1000) + `]}`

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(largeJSON))
	})

	middleware := GzipCompressionMiddleware(handler)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		// Без Accept-Encoding: gzip
		w := httptest.NewRecorder()
		middleware.ServeHTTP(w, req)
	}
}

// BenchmarkGzipPool измеряет эффективность использования пула
func BenchmarkGzipPool(b *testing.B) {
	data := []byte(strings.Repeat("test data ", 1000))

	b.Run("WithPool", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			var buf bytes.Buffer
			gz := gzipWriterPool.Get().(*gzip.Writer)
			gz.Reset(&buf)
			gz.Write(data)
			gz.Close()
			gzipWriterPool.Put(gz)
		}
	})

	b.Run("WithoutPool", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			var buf bytes.Buffer
			gz := gzip.NewWriter(&buf)
			gz.Write(data)
			gz.Close()
		}
	})
}

// BenchmarkShouldCompress измеряет производительность проверки типа контента
func BenchmarkShouldCompress(b *testing.B) {
	contentTypes := []string{
		"application/json",
		"application/json; charset=utf-8",
		"text/html",
		"text/html; charset=utf-8",
		"image/png",
		"application/octet-stream",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		shouldCompress(contentTypes[i%len(contentTypes)])
	}
}
