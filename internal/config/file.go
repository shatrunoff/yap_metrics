package config

import (
	"encoding/json"
	"os"
)

// ServerConfigFile represents the JSON structure for server configuration
type ServerConfigFile struct {
	Address       string `json:"address"`
	Restore       *bool  `json:"restore"`
	StoreInterval string `json:"store_interval"`
	StoreFile     string `json:"store_file"`
	DatabaseDSN   string `json:"database_dsn"`
	CryptoKey     string `json:"crypto_key"`
}

// AgentConfigFile represents the JSON structure for agent configuration
type AgentConfigFile struct {
	Address        string `json:"address"`
	ReportInterval string `json:"report_interval"`
	PollInterval   string `json:"poll_interval"`
	CryptoKey      string `json:"crypto_key"`
}

// LoadServerConfigFromFile loads server configuration from JSON file
func LoadServerConfigFromFile(filename string) (*ServerConfigFile, error) {
	if filename == "" {
		return nil, nil
	}

	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var config ServerConfigFile
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

// LoadAgentConfigFromFile loads agent configuration from JSON file
func LoadAgentConfigFromFile(filename string) (*AgentConfigFile, error) {
	if filename == "" {
		return nil, nil
	}

	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var config AgentConfigFile
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}
