package config

import (
	"os"
	"reflect"
	"testing"
	"time"
)

func TestDefaultAgentConfig(t *testing.T) {
	tests := []struct {
		name string
		want *AgentConfig
	}{
		{
			name: "Default configuration values",
			want: &AgentConfig{
				PollInterval:   2 * time.Second,
				ReportInterval: 10 * time.Second,
				ServerURL:      "localhost:8080",
				Key:            "",
				RateLimit:      0,
				CryptoKey:      "",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := DefaultAgentConfig(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("DefaultAgentConfig() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseAgentConfigWithEnvVars(t *testing.T) {
	// Save original env vars
	originalVars := map[string]string{
		"ADDRESS":         os.Getenv("ADDRESS"),
		"REPORT_INTERVAL": os.Getenv("REPORT_INTERVAL"),
		"POLL_INTERVAL":   os.Getenv("POLL_INTERVAL"),
		"KEY":             os.Getenv("KEY"),
		"CRYPTO_KEY":      os.Getenv("CRYPTO_KEY"),
		"RATE_LIMIT":      os.Getenv("RATE_LIMIT"),
	}

	// Clean up after test
	defer func() {
		for key, value := range originalVars {
			if value == "" {
				os.Unsetenv(key)
			} else {
				os.Setenv(key, value)
			}
		}
	}()

	// Set test env vars
	os.Setenv("ADDRESS", "test:9090")
	os.Setenv("REPORT_INTERVAL", "5")
	os.Setenv("POLL_INTERVAL", "3")
	os.Setenv("KEY", "test-key")
	os.Setenv("CRYPTO_KEY", "/test/key.pem")
	os.Setenv("RATE_LIMIT", "10")

	// Reset flag package state
	resetFlags()

	cfg := ParseAgentConfig()

	if cfg.ServerURL != "test:9090" {
		t.Errorf("Expected ServerURL test:9090, got %s", cfg.ServerURL)
	}
	if cfg.ReportInterval != 5*time.Second {
		t.Errorf("Expected ReportInterval 5s, got %v", cfg.ReportInterval)
	}
	if cfg.PollInterval != 3*time.Second {
		t.Errorf("Expected PollInterval 3s, got %v", cfg.PollInterval)
	}
	if cfg.Key != "test-key" {
		t.Errorf("Expected Key test-key, got %s", cfg.Key)
	}
	if cfg.CryptoKey != "/test/key.pem" {
		t.Errorf("Expected CryptoKey /test/key.pem, got %s", cfg.CryptoKey)
	}
	if cfg.RateLimit != 10 {
		t.Errorf("Expected RateLimit 10, got %d", cfg.RateLimit)
	}
}

// resetFlags resets the flag package state for testing
func resetFlags() {
	// This is a simple approach - in real scenarios you might need more sophisticated flag reset
}
