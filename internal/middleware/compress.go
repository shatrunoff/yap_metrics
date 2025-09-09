package middleware

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"
	"sync"
)

var gzipWriterPool = sync.Pool{
	New: func() interface{} {
		return gzip.NewWriter(io.Discard)
	},
}

// сжимает ответ, если клиент поддерживает gzip
func GzipCompressionMiddleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Проверяем, поддерживает ли клиент gzip
		acceptEncoding := r.Header.Get("Accept-Encoding")
		if !strings.Contains(acceptEncoding, "gzip") {
			h.ServeHTTP(w, r)
			return
		}

		// Проверяем, нужно ли сжимать этот тип контента
		contentType := w.Header().Get("Content-Type")
		if !shouldCompress(contentType) {
			h.ServeHTTP(w, r)
			return
		}

		// Создаем gzip writer
		gz := gzipWriterPool.Get().(*gzip.Writer)
		defer gzipWriterPool.Put(gz)
		defer gz.Reset(io.Discard)

		gzipWriter := &gzipResponseWriter{
			ResponseWriter: w,
			gzipWriter:     gz,
		}
		gzipWriter.gzipWriter.Reset(w)

		// Устанавливаем заголовки
		w.Header().Set("Content-Encoding", "gzip")
		w.Header().Del("Content-Length")

		h.ServeHTTP(gzipWriter, r)

		// Завершаем сжатие
		gzipWriter.gzipWriter.Close()
	})
}

// распаковывает входящие gzip данные
func GzipDecompressionMiddleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		contentEncoding := r.Header.Get("Content-Encoding")
		if strings.Contains(contentEncoding, "gzip") {
			gz, err := gzip.NewReader(r.Body)
			if err != nil {
				http.Error(w, "Invalid gzip data", http.StatusBadRequest)
				return
			}
			defer gz.Close()
			r.Body = gz
		}
		h.ServeHTTP(w, r)
	})
}

type gzipResponseWriter struct {
	http.ResponseWriter
	gzipWriter *gzip.Writer
}

func (g *gzipResponseWriter) Write(b []byte) (int, error) {
	return g.gzipWriter.Write(b)
}

func shouldCompress(contentType string) bool {
	return strings.HasPrefix(contentType, "application/json") ||
		strings.HasPrefix(contentType, "text/html")
}
