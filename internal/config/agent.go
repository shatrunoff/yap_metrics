package config

import "time"

type AgentConfig struct {
	PollInterval   time.Duration
	ReportInterval time.Duration
	ServerURL      string
}

func DefaultAgentConfig() *AgentConfig {
	return &AgentConfig{
		PollInterval:   2 * time.Second,
		ReportInterval: 10 * time.Second,
		ServerURL:      "localhost:8080",
	}
}
