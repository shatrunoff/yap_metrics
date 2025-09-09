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

		// Создаем обертку для захвата ответа
		writer := &gzipResponseWriter{
			ResponseWriter: w,
			gzipWriter:     nil,
		}
		defer writer.Close()

		h.ServeHTTP(writer, r)
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
	// Проверяем Content-Type
	contentType := g.Header().Get("Content-Type")
	if !shouldCompress(contentType) {
		return g.ResponseWriter.Write(b)
	}

	// Если gzip writer еще не создан, создаем его
	if g.gzipWriter == nil {
		gz := gzipWriterPool.Get().(*gzip.Writer)
		gz.Reset(g.ResponseWriter)
		g.gzipWriter = gz
		g.Header().Set("Content-Encoding", "gzip")
		g.Header().Del("Content-Length")
	}

	return g.gzipWriter.Write(b)
}

func (g *gzipResponseWriter) WriteHeader(statusCode int) {
	contentType := g.Header().Get("Content-Type")
	if shouldCompress(contentType) && g.gzipWriter == nil {
		gz := gzipWriterPool.Get().(*gzip.Writer)
		gz.Reset(g.ResponseWriter)
		g.gzipWriter = gz
		g.Header().Set("Content-Encoding", "gzip")
		g.Header().Del("Content-Length")
	}
	g.ResponseWriter.WriteHeader(statusCode)
}

func (g *gzipResponseWriter) Close() {
	if g.gzipWriter != nil {
		g.gzipWriter.Close()
		gzipWriterPool.Put(g.gzipWriter)
	}
}

func shouldCompress(contentType string) bool {
	return strings.HasPrefix(contentType, "application/json") ||
		strings.HasPrefix(contentType, "text/html")
}
