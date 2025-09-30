package middleware

import (
	"bytes"
	"io"
	"net/http"
	"strings"

	"github.com/shatrunoff/yap_metrics/internal/utils"
)

// проверяет подпись в тексте запроса и подписывает ответы, используя HMAC-SHA256
func SignatureMiddleware(key string) func(http.Handler) http.Handler {
	if key == "" {
		return func(h http.Handler) http.Handler { return h }
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// проверяем хэш запроса
			if r.Method == http.MethodPost && strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") &&
				strings.HasPrefix(r.URL.Path, "/updates/") {
				provided := r.Header.Get("HashSHA256")
				if provided == "" {
					http.Error(w, "missing HashSHA256 header", http.StatusBadRequest)
					return
				}

				body, err := io.ReadAll(r.Body)
				if err != nil {
					http.Error(w, "failed to read body", http.StatusBadRequest)
					return
				}
				// вычисляем HMAC
				expected := utils.ComputeHMACSHA256(body, key)
				if !strings.EqualFold(provided, expected) {
					http.Error(w, "invalid HashSHA256", http.StatusBadRequest)
					return
				}
				r.Body = io.NopCloser(bytes.NewReader(body))
			}

			srw := &signingResponseWriter{ResponseWriter: w}
			next.ServeHTTP(srw, r)

			sig := utils.ComputeHMACSHA256(srw.buf.Bytes(), key)

			for k, vv := range srw.header {
				for _, v := range vv {
					w.Header().Add(k, v)
				}
			}
			w.Header().Set("HashSHA256", sig)
			if srw.statusCode != 0 {
				w.WriteHeader(srw.statusCode)
			}
			_, _ = w.Write(srw.buf.Bytes())
		})
	}
}

type signingResponseWriter struct {
	http.ResponseWriter
	buf        bytes.Buffer
	statusCode int
	header     http.Header
}

func (s *signingResponseWriter) Header() http.Header {
	if s.header == nil {
		s.header = make(http.Header)
	}
	return s.header
}

func (s *signingResponseWriter) WriteHeader(code int) {
	s.statusCode = code
}

func (s *signingResponseWriter) Write(b []byte) (int, error) {
	return s.buf.Write(b)
}
