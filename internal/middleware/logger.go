package middleware

import (
	"net/http"
	"time"

	"go.uber.org/zap"
)

var Logger *zap.Logger
var Sugar *zap.SugaredLogger

func InitLogger() error {
	var err error
	Logger, err = zap.NewProduction()
	if err != nil {
		return err
	}
	Sugar = Logger.Sugar()
	return nil
}

func GetLogger() *zap.Logger {
	return Logger
}

func GetSugar() *zap.SugaredLogger {
	return Sugar
}

type responseWriter struct {
	http.ResponseWriter
	status      int
	size        int
	wroteHeader bool
}

func (rw *responseWriter) WriteHeader(code int) {
	if rw.wroteHeader {
		return
	}
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
	rw.wroteHeader = true
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	if !rw.wroteHeader {
		rw.WriteHeader(http.StatusOK)
	}
	size, err := rw.ResponseWriter.Write(b)
	rw.size += size
	return size, err
}

func LoggingMiddleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// проверка на nil
		if Sugar == nil {
			h.ServeHTTP(w, r)
			return
		}

		start := time.Now()

		// логируем запрос
		Sugar.Infow("Request",
			zap.String("Method", r.Method),
			zap.String("URI", r.RequestURI),
		)

		// захват статуса
		rw := &responseWriter{ResponseWriter: w, status: http.StatusOK}

		// передаем управление
		h.ServeHTTP(rw, r)

		duration := time.Since(start)

		logData := []any{
			"URI", r.RequestURI,
			"Method", r.Method,
			"Duration", duration.String(),

			"Status", rw.status,
			"Size", rw.size,
		}

		Sugar.Infow("Request completed", logData...)
	})
}
