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
				(strings.HasPrefix(r.URL.Path, "/updates/") || strings.HasPrefix(r.URL.Path, "/update/")) {

				provided := r.Header.Get("HashSHA256")
				mustHave := strings.HasPrefix(r.URL.Path, "/updates/") // для батча подпись обязательна

				if provided == "" {
					if mustHave {
						http.Error(w, "missing HashSHA256 header", http.StatusBadRequest)
						return
					}
					// для одиночного /update/ заголовок опционален — пропускаем без проверки
				} else {
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
			}

			srw := &signingResponseWriter{ResponseWriter: w}
			next.ServeHTTP(srw, r)

			if !srw.decided {
				ct := srw.header.Get("Content-Type")
				if ct != "" {
					srw.sign = strings.HasPrefix(ct, "application/json")
					srw.decided = true
				}
			}

			if srw.sign {
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
				return
			}
		})
	}
}

type signingResponseWriter struct {
	http.ResponseWriter
	buf        bytes.Buffer
	statusCode int
	header     http.Header
	decided    bool
	sign       bool
	flushed    bool
}

func (s *signingResponseWriter) Header() http.Header {
	if s.header == nil {
		s.header = make(http.Header)
	}
	return s.header
}

func (s *signingResponseWriter) WriteHeader(code int) {
	s.statusCode = code
	// решение в зависимости от Content-Type
	if !s.decided {
		ct := s.header.Get("Content-Type")
		if ct != "" {
			s.sign = strings.HasPrefix(ct, "application/json")
			s.decided = true
		}
	}
	if s.decided && !s.sign && !s.flushed {
		for k, vv := range s.header {
			for _, v := range vv {
				s.ResponseWriter.Header().Add(k, v)
			}
		}
		s.ResponseWriter.WriteHeader(code)
		s.flushed = true
	}
}

func (s *signingResponseWriter) Write(b []byte) (int, error) {

	if !s.decided {
		ct := s.header.Get("Content-Type")
		if ct != "" {
			s.sign = strings.HasPrefix(ct, "application/json")
			s.decided = true
		} else {
			s.sign = false
			s.decided = true
		}
	}

	if s.sign {
		return s.buf.Write(b)
	}

	if !s.flushed {
		for k, vv := range s.header {
			for _, v := range vv {
				s.ResponseWriter.Header().Add(k, v)
			}
		}
		if s.statusCode != 0 {
			s.ResponseWriter.WriteHeader(s.statusCode)
		}
		s.flushed = true
	}
	return s.ResponseWriter.Write(b)
}
