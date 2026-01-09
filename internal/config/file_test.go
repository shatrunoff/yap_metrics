package config

import (
	"os"
	"testing"
)

func TestLoadServerConfigFromFile(t *testing.T) {
	// Create a temporary config file
	configContent := `{
		"address": "localhost:9090",
		"restore": false,
		"store_interval": "5s",
		"store_file": "/tmp/test.db",
		"database_dsn": "postgres://test",
		"crypto_key": "/tmp/key.pem"
	}`

	tmpFile, err := os.CreateTemp("", "server_config_*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(configContent); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}
	tmpFile.Close()

	// Test loading config
	config, err := LoadServerConfigFromFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if config.Address != "localhost:9090" {
		t.Errorf("Expected address localhost:9090, got %s", config.Address)
	}
	if config.Restore == nil || *config.Restore != false {
		t.Errorf("Expected restore false, got %v", config.Restore)
	}
	if config.StoreInterval != "5s" {
		t.Errorf("Expected store_interval 5s, got %s", config.StoreInterval)
	}
}

func TestLoadAgentConfigFromFile(t *testing.T) {
	// Create a temporary config file
	configContent := `{
		"address": "localhost:9090",
		"report_interval": "5s",
		"poll_interval": "3s",
		"crypto_key": "/tmp/key.pem"
	}`

	tmpFile, err := os.CreateTemp("", "agent_config_*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(configContent); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}
	tmpFile.Close()

	// Test loading config
	config, err := LoadAgentConfigFromFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if config.Address != "localhost:9090" {
		t.Errorf("Expected address localhost:9090, got %s", config.Address)
	}
	if config.ReportInterval != "5s" {
		t.Errorf("Expected report_interval 5s, got %s", config.ReportInterval)
	}
	if config.PollInterval != "3s" {
		t.Errorf("Expected poll_interval 3s, got %s", config.PollInterval)
	}
}
func TestLoadServerConfigFromFileErrors(t *testing.T) {
	// Test empty filename
	config, err := LoadServerConfigFromFile("")
	if err != nil || config != nil {
		t.Error("Expected nil config and no error for empty filename")
	}

	// Test non-existent file
	_, err = LoadServerConfigFromFile("nonexistent.json")
	if err == nil {
		t.Error("Expected error for non-existent file")
	}

	// Test invalid JSON
	tmpFile, err := os.CreateTemp("", "invalid_*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	tmpFile.WriteString("invalid json")
	tmpFile.Close()

	_, err = LoadServerConfigFromFile(tmpFile.Name())
	if err == nil {
		t.Error("Expected error for invalid JSON")
	}
}

func TestLoadAgentConfigFromFileErrors(t *testing.T) {
	// Test empty filename
	config, err := LoadAgentConfigFromFile("")
	if err != nil || config != nil {
		t.Error("Expected nil config and no error for empty filename")
	}

	// Test non-existent file
	_, err = LoadAgentConfigFromFile("nonexistent.json")
	if err == nil {
		t.Error("Expected error for non-existent file")
	}

	// Test invalid JSON
	tmpFile, err := os.CreateTemp("", "invalid_*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	tmpFile.WriteString("invalid json")
	tmpFile.Close()

	_, err = LoadAgentConfigFromFile(tmpFile.Name())
	if err == nil {
		t.Error("Expected error for invalid JSON")
	}
}
