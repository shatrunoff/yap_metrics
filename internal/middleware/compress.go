package middleware

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"
	"sync"

	"go.uber.org/zap"
)

var gzipWriterPool = sync.Pool{
	New: func() interface{} {
		return gzip.NewWriter(io.Discard)
	},
}

// сжимает ответы если клиент поддерживает gzip
func GzipMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logger := GetLogger()

		// Проверяем, поддерживает ли клиент gzip
		acceptsGzip := strings.Contains(r.Header.Get("Accept-Encoding"), "gzip")

		// Проверяем, сжат ли запрос
		isGzipped := strings.Contains(r.Header.Get("Content-Encoding"), "gzip")

		// Обрабатываем сжатый запрос
		if isGzipped {
			gz, err := gzip.NewReader(r.Body)
			if err != nil {
				logger.Error("Failed to create gzip reader", zap.Error(err))
				http.Error(w, "Invalid gzip encoding", http.StatusBadRequest)
				return
			}
			defer gz.Close()
			r.Body = gz
		}

		// Если клиент не поддерживает gzip, пропускаем сжатие
		if !acceptsGzip {
			next.ServeHTTP(w, r)
			return
		}

		// Проверяем, поддерживаем ли мы сжатие для этого content-type
		contentType := w.Header().Get("Content-Type")
		if !shouldCompress(contentType) {
			next.ServeHTTP(w, r)
			return
		}

		// Создаем gzip writer
		gz := gzipWriterPool.Get().(*gzip.Writer)
		defer gzipWriterPool.Put(gz)
		defer gz.Reset(io.Discard)

		gz.Reset(w)
		defer gz.Close()

		// Обертываем ResponseWriter
		gzw := &gzipResponseWriter{
			ResponseWriter: w,
			Writer:         gz,
		}

		// Устанавливаем заголовки
		w.Header().Set("Content-Encoding", "gzip")
		w.Header().Del("Content-Length")

		next.ServeHTTP(gzw, r)
	})
}

// оборачивает ResponseWriter для сжатия
type gzipResponseWriter struct {
	http.ResponseWriter
	io.Writer
}

func (g *gzipResponseWriter) Write(b []byte) (int, error) {
	return g.Writer.Write(b)
}

// проверяет, нужно ли сжимать данный content-type
func shouldCompress(contentType string) bool {
	compressibleTypes := []string{
		"application/json",
		"text/plain",
	}

	for _, t := range compressibleTypes {
		if strings.Contains(contentType, t) {
			return true
		}
	}
	return false
}
