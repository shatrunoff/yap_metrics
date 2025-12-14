package config

import (
	"flag"
	"os"
	"testing"
	"time"
)

// TestDefaultServerConfig проверяет значения по умолчанию
func TestDefaultServerConfig(t *testing.T) {
	cfg := DefaultServerConfig()

	expected := &ServerConfig{
		ServerURL:       "localhost:8080",
		StoreInterval:   300 * time.Second,
		FileStoragePath: "tmp/my-metrics.json",
		Restore:         true,
		DatabaseDSN:     "",
		Key:             "",
		AuditFile:       "",
		AuditURL:        "",
	}

	if cfg.ServerURL != expected.ServerURL {
		t.Errorf("ServerURL: got %q, want %q", cfg.ServerURL, expected.ServerURL)
	}
	if cfg.StoreInterval != expected.StoreInterval {
		t.Errorf("StoreInterval: got %v, want %v", cfg.StoreInterval, expected.StoreInterval)
	}
	if cfg.FileStoragePath != expected.FileStoragePath {
		t.Errorf("FileStoragePath: got %q, want %q", cfg.FileStoragePath, expected.FileStoragePath)
	}
	if cfg.Restore != expected.Restore {
		t.Errorf("Restore: got %v, want %v", cfg.Restore, expected.Restore)
	}
	if cfg.DatabaseDSN != expected.DatabaseDSN {
		t.Errorf("DatabaseDSN: got %q, want %q", cfg.DatabaseDSN, expected.DatabaseDSN)
	}
	if cfg.Key != expected.Key {
		t.Errorf("Key: got %q, want %q", cfg.Key, expected.Key)
	}
	if cfg.AuditFile != expected.AuditFile {
		t.Errorf("AuditFile: got %q, want %q", cfg.AuditFile, expected.AuditFile)
	}
	if cfg.AuditURL != expected.AuditURL {
		t.Errorf("AuditURL: got %q, want %q", cfg.AuditURL, expected.AuditURL)
	}
}

// TestParseServerConfigEnv проверяет парсинг переменных окружения
func TestParseServerConfigEnv(t *testing.T) {
	// Сохраняем оригинальные аргументы
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	// Устанавливаем минимальные аргументы
	os.Args = []string{"test"}

	// Сохраняем оригинальные переменные окружения
	oldEnv := make(map[string]string)
	envVars := []string{"ADDRESS", "STORE_INTERVAL", "FILE_STORAGE_PATH",
		"RESTORE", "DATABASE_DSN", "KEY", "AUDIT_FILE", "AUDIT_URL"}

	for _, key := range envVars {
		if val := os.Getenv(key); val != "" {
			oldEnv[key] = val
		}
		os.Unsetenv(key)
	}
	defer func() {
		for key, val := range oldEnv {
			os.Setenv(key, val)
		}
	}()

	tests := []struct {
		name     string
		env      map[string]string
		expected *ServerConfig
	}{
		{
			name: "all env vars set",
			env: map[string]string{
				"ADDRESS":           "env.example.com:8081",
				"STORE_INTERVAL":    "120",
				"FILE_STORAGE_PATH": "/env/path.json",
				"RESTORE":           "false",
				"DATABASE_DSN":      "postgres://env@localhost/db",
				"KEY":               "env-secret",
				"AUDIT_FILE":        "/env/audit.log",
				"AUDIT_URL":         "http://env.audit.com",
			},
			expected: &ServerConfig{
				ServerURL:       "env.example.com:8081",
				StoreInterval:   120 * time.Second,
				FileStoragePath: "/env/path.json",
				Restore:         false,
				DatabaseDSN:     "postgres://env@localhost/db",
				Key:             "env-secret",
				AuditFile:       "/env/audit.log",
				AuditURL:        "http://env.audit.com",
			},
		},
		{
			name: "partial env vars set",
			env: map[string]string{
				"ADDRESS":           "env-host:9090",
				"FILE_STORAGE_PATH": "env-file.json",
				"RESTORE":           "true",
			},
			expected: &ServerConfig{
				ServerURL:       "env-host:9090",
				StoreInterval:   300 * time.Second, // default
				FileStoragePath: "env-file.json",
				Restore:         true,
				DatabaseDSN:     "", // default
				Key:             "", // default
				AuditFile:       "", // default
				AuditURL:        "", // default
			},
		},
		{
			name: "invalid STORE_INTERVAL",
			env: map[string]string{
				"STORE_INTERVAL": "invalid",
			},
			expected: &ServerConfig{
				ServerURL:       "localhost:8080",      // default
				StoreInterval:   300 * time.Second,     // default (не изменится из-за ошибки парсинга)
				FileStoragePath: "tmp/my-metrics.json", // default
				Restore:         true,                  // default
				DatabaseDSN:     "",                    // default
				Key:             "",                    // default
				AuditFile:       "",                    // default
				AuditURL:        "",                    // default
			},
		},
		{
			name: "invalid RESTORE",
			env: map[string]string{
				"RESTORE": "not-a-bool",
			},
			expected: &ServerConfig{
				ServerURL:       "localhost:8080",      // default
				StoreInterval:   300 * time.Second,     // default
				FileStoragePath: "tmp/my-metrics.json", // default
				Restore:         true,                  // default (не изменится из-за ошибки парсинга)
				DatabaseDSN:     "",                    // default
				Key:             "",                    // default
				AuditFile:       "",                    // default
				AuditURL:        "",                    // default
			},
		},
		{
			name: "empty string values",
			env: map[string]string{
				"ADDRESS":           "",
				"STORE_INTERVAL":    "",
				"FILE_STORAGE_PATH": "",
				"RESTORE":           "",
				"DATABASE_DSN":      "",
				"KEY":               "",
				"AUDIT_FILE":        "",
				"AUDIT_URL":         "",
			},
			expected: DefaultServerConfig(), // Пустые строки не должны перезаписывать значения
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Устанавливаем переменные окружения
			for key, value := range tt.env {
				os.Setenv(key, value)
			}

			// Сбрасываем флаги
			flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

			cfg := ParseServerConfig()

			// Очищаем переменные окружения для следующего теста
			for key := range tt.env {
				os.Unsetenv(key)
			}

			if cfg.ServerURL != tt.expected.ServerURL {
				t.Errorf("ServerURL: got %q, want %q", cfg.ServerURL, tt.expected.ServerURL)
			}
			if cfg.StoreInterval != tt.expected.StoreInterval {
				t.Errorf("StoreInterval: got %v, want %v", cfg.StoreInterval, tt.expected.StoreInterval)
			}
			if cfg.FileStoragePath != tt.expected.FileStoragePath {
				t.Errorf("FileStoragePath: got %q, want %q", cfg.FileStoragePath, tt.expected.FileStoragePath)
			}
			if cfg.Restore != tt.expected.Restore {
				t.Errorf("Restore: got %v, want %v", cfg.Restore, tt.expected.Restore)
			}
			if cfg.DatabaseDSN != tt.expected.DatabaseDSN {
				t.Errorf("DatabaseDSN: got %q, want %q", cfg.DatabaseDSN, tt.expected.DatabaseDSN)
			}
			if cfg.Key != tt.expected.Key {
				t.Errorf("Key: got %q, want %q", cfg.Key, tt.expected.Key)
			}
			if cfg.AuditFile != tt.expected.AuditFile {
				t.Errorf("AuditFile: got %q, want %q", cfg.AuditFile, tt.expected.AuditFile)
			}
			if cfg.AuditURL != tt.expected.AuditURL {
				t.Errorf("AuditURL: got %q, want %q", cfg.AuditURL, tt.expected.AuditURL)
			}
		})
	}
}

// TestParseServerConfigFlagsAndEnv проверяет приоритет переменных окружения над флагами
func TestParseServerConfigFlagsAndEnv(t *testing.T) {
	// Сохраняем оригинальные аргументы
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	// Устанавливаем аргументы командной строки
	os.Args = []string{
		"test",
		"-a", "flag-host:8080",
		"-i", "30",
		"-f", "flag-file.json",
		"-r", "false",
		"-d", "flag-dsn",
		"-k", "flag-key",
		"-audit-file", "flag-audit.log",
		"-audit-url", "http://flag.audit",
	}

	// Сохраняем и очищаем переменные окружения
	oldEnv := make(map[string]string)
	envVars := []string{"ADDRESS", "STORE_INTERVAL", "FILE_STORAGE_PATH",
		"RESTORE", "DATABASE_DSN", "KEY", "AUDIT_FILE", "AUDIT_URL"}

	for _, key := range envVars {
		if val := os.Getenv(key); val != "" {
			oldEnv[key] = val
		}
		os.Unsetenv(key)
	}
	defer func() {
		for key, val := range oldEnv {
			os.Setenv(key, val)
		}
	}()

	// Устанавливаем переменные окружения (должны перезаписать флаги)
	os.Setenv("ADDRESS", "env-host:9090")
	os.Setenv("STORE_INTERVAL", "60")
	os.Setenv("FILE_STORAGE_PATH", "env-file.json")
	os.Setenv("RESTORE", "true")
	os.Setenv("DATABASE_DSN", "env-dsn")
	os.Setenv("KEY", "env-key")
	os.Setenv("AUDIT_FILE", "env-audit.log")
	os.Setenv("AUDIT_URL", "http://env.audit")

	// Сбрасываем флаги
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	cfg := ParseServerConfig()

	// Проверяем, что переменные окружения имеют приоритет
	expected := &ServerConfig{
		ServerURL:       "env-host:9090",
		StoreInterval:   60 * time.Second,
		FileStoragePath: "env-file.json",
		Restore:         true,
		DatabaseDSN:     "env-dsn",
		Key:             "env-key",
		AuditFile:       "env-audit.log",
		AuditURL:        "http://env.audit",
	}

	if cfg.ServerURL != expected.ServerURL {
		t.Errorf("ServerURL: got %q, want %q (env should override flag)", cfg.ServerURL, expected.ServerURL)
	}
	if cfg.StoreInterval != expected.StoreInterval {
		t.Errorf("StoreInterval: got %v, want %v (env should override flag)", cfg.StoreInterval, expected.StoreInterval)
	}
	if cfg.FileStoragePath != expected.FileStoragePath {
		t.Errorf("FileStoragePath: got %q, want %q (env should override flag)", cfg.FileStoragePath, expected.FileStoragePath)
	}
	if cfg.Restore != expected.Restore {
		t.Errorf("Restore: got %v, want %v (env should override flag)", cfg.Restore, expected.Restore)
	}
	if cfg.DatabaseDSN != expected.DatabaseDSN {
		t.Errorf("DatabaseDSN: got %q, want %q (env should override flag)", cfg.DatabaseDSN, expected.DatabaseDSN)
	}
	if cfg.Key != expected.Key {
		t.Errorf("Key: got %q, want %q (env should override flag)", cfg.Key, expected.Key)
	}
	if cfg.AuditFile != expected.AuditFile {
		t.Errorf("AuditFile: got %q, want %q (env should override flag)", cfg.AuditFile, expected.AuditFile)
	}
	if cfg.AuditURL != expected.AuditURL {
		t.Errorf("AuditURL: got %q, want %q (env should override flag)", cfg.AuditURL, expected.AuditURL)
	}
}

// TestServerConfigFields проверяет, что все поля доступны и имеют правильные типы
func TestServerConfigFields(t *testing.T) {
	cfg := &ServerConfig{}

	// Проверяем, что поля существуют и имеют правильные типы
	_ = cfg.ServerURL
	_ = cfg.StoreInterval
	_ = cfg.FileStoragePath
	_ = cfg.Restore
	_ = cfg.DatabaseDSN
	_ = cfg.Key
	_ = cfg.AuditFile
	_ = cfg.AuditURL

	// Если код компилируется, тест пройден
	t.Log("All ServerConfig fields are accessible")
}
