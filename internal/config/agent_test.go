package config

import (
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
