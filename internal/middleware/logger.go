package middleware

import (
	"log"
	"net/http"
)

type responseWriter struct {
	http.ResponseWriter
	status int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

func LoggingMiddleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// логирование отправленных метрик
		log.Printf("Incoming update: %s %s", r.Method, r.URL.Path)
		// обертка для захвата статуса
		rw := &responseWriter{w, http.StatusOK}

		h.ServeHTTP(rw, r)

		// статус логирования отправленных метрик
		log.Printf("Successfuly update: %s %s | Status: %d", r.Method, r.URL.Path, rw.status)
	})
}
