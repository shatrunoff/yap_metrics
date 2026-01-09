package middleware

import (
	"bytes"
	"crypto/rsa"
	"io"
	"net/http"

	"github.com/shatrunoff/yap_metrics/internal/crypto"
)

func DecryptionMiddleware(privateKey *rsa.PrivateKey) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if privateKey == nil || r.Header.Get("Content-Encoding") != "rsa" {
				next.ServeHTTP(w, r)
				return
			}

			body, err := io.ReadAll(r.Body)
			if err != nil {
				http.Error(w, "Failed to read request body", http.StatusBadRequest)
				return
			}
			r.Body.Close()

			decryptedData, err := crypto.Decrypt(body, privateKey)
			if err != nil {
				http.Error(w, "Failed to decrypt data", http.StatusBadRequest)
				return
			}

			r.Body = io.NopCloser(bytes.NewReader(decryptedData))
			r.Header.Set("Content-Encoding", "")
			r.ContentLength = int64(len(decryptedData))

			next.ServeHTTP(w, r)
		})
	}
}
