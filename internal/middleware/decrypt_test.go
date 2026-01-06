package middleware

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/shatrunoff/yap_metrics/internal/crypto"
)

func TestDecryptionMiddleware(t *testing.T) {
	// Generate test key pair
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	testData := []byte("test message")
	encryptedData, err := crypto.Encrypt(testData, &privateKey.PublicKey)
	if err != nil {
		t.Fatalf("Failed to encrypt test data: %v", err)
	}

	tests := []struct {
		name            string
		privateKey      *rsa.PrivateKey
		contentEncoding string
		body            []byte
		expectDecrypt   bool
		expectError     bool
	}{
		{
			name:            "decrypt with rsa encoding",
			privateKey:      privateKey,
			contentEncoding: "rsa",
			body:            encryptedData,
			expectDecrypt:   true,
			expectError:     false,
		},
		{
			name:            "no decryption without rsa encoding",
			privateKey:      privateKey,
			contentEncoding: "gzip",
			body:            testData,
			expectDecrypt:   false,
			expectError:     false,
		},
		{
			name:            "no decryption with nil key",
			privateKey:      nil,
			contentEncoding: "rsa",
			body:            encryptedData,
			expectDecrypt:   false,
			expectError:     false,
		},
		{
			name:            "error with invalid encrypted data",
			privateKey:      privateKey,
			contentEncoding: "rsa",
			body:            []byte("invalid encrypted data"),
			expectDecrypt:   false,
			expectError:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test handler that captures the request body
			var capturedBody []byte
			testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				body, err := io.ReadAll(r.Body)
				if err != nil {
					t.Fatalf("Failed to read body in test handler: %v", err)
				}
				capturedBody = body
				w.WriteHeader(http.StatusOK)
			})

			// Create middleware
			middleware := DecryptionMiddleware(tt.privateKey)
			handler := middleware(testHandler)

			// Create request
			req := httptest.NewRequest("POST", "/test", bytes.NewReader(tt.body))
			if tt.contentEncoding != "" {
				req.Header.Set("Content-Encoding", tt.contentEncoding)
			}

			// Create response recorder
			rr := httptest.NewRecorder()

			// Execute request
			handler.ServeHTTP(rr, req)

			if tt.expectError {
				if rr.Code != http.StatusBadRequest {
					t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rr.Code)
				}
				return
			}

			if rr.Code != http.StatusOK {
				t.Errorf("Expected status %d, got %d", http.StatusOK, rr.Code)
			}

			if tt.expectDecrypt {
				if !bytes.Equal(capturedBody, testData) {
					t.Errorf("Expected decrypted body %s, got %s", string(testData), string(capturedBody))
				}
			} else {
				if !bytes.Equal(capturedBody, tt.body) {
					t.Errorf("Expected original body %s, got %s", string(tt.body), string(capturedBody))
				}
			}
		})
	}
}

func TestDecryptionMiddlewareReadError(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	// Create a request with a body that will cause read error
	req := httptest.NewRequest("POST", "/test", &errorReader{})
	req.Header.Set("Content-Encoding", "rsa")

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Handler should not be called when read fails")
	})

	middleware := DecryptionMiddleware(privateKey)
	handler := middleware(testHandler)

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}
}

// errorReader is a helper that always returns an error when reading
type errorReader struct{}

func (e *errorReader) Read(p []byte) (n int, err error) {
	return 0, io.ErrUnexpectedEOF
}
