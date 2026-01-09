package config

import (
	"flag"
	"log"
	"os"
	"strconv"
	"time"
)

type AgentConfig struct {
	PollInterval   time.Duration
	ReportInterval time.Duration
	ServerURL      string
	Key            string
	RateLimit      int
	CryptoKey      string
}

func DefaultAgentConfig() *AgentConfig {
	return &AgentConfig{
		PollInterval:   2 * time.Second,
		ReportInterval: 10 * time.Second,
		ServerURL:      "localhost:8080",
		Key:            "",
		RateLimit:      0,
		CryptoKey:      "",
	}
}

type agentFlags struct {
	pollSec    int
	repSec     int
	key        string
	rateLimit  int
	cryptoKey  string
	configFile string
}

func parseAgentFlags(cfg *AgentConfig) *agentFlags {
	f := &agentFlags{}
	flag.StringVar(&cfg.ServerURL, "a", cfg.ServerURL, "Server address host: ")
	flag.IntVar(&f.pollSec, "p", int(cfg.PollInterval.Seconds()), "PollInterval (s)")
	flag.IntVar(&f.repSec, "r", int(cfg.ReportInterval.Seconds()), "ReportInterval (s)")
	flag.StringVar(&f.key, "k", cfg.Key, "Signing key for HashSHA256 header")
	flag.StringVar(&f.cryptoKey, "crypto-key", cfg.CryptoKey, "Path to public key file for encryption")
	flag.IntVar(&f.rateLimit, "l", cfg.RateLimit, "Max concurrent outgoing requests")
	flag.StringVar(&f.configFile, "c", "", "Config file path")
	flag.StringVar(&f.configFile, "config", "", "Config file path")
	flag.Parse()

	if flag.NArg() > 0 {
		log.Fatalf("ERROR: unknown arguments: %v", flag.Args())
	}
	return f
}

func applyAgentFileConfig(cfg *AgentConfig, configFile string) {
	if configFile == "" {
		configFile = os.Getenv("CONFIG")
	}
	fileConfig, err := LoadAgentConfigFromFile(configFile)
	if err != nil || fileConfig == nil {
		return
	}
	if fileConfig.Address != "" {
		cfg.ServerURL = fileConfig.Address
	}
	if fileConfig.ReportInterval != "" {
		if duration, err := time.ParseDuration(fileConfig.ReportInterval); err == nil {
			cfg.ReportInterval = duration
		}
	}
	if fileConfig.PollInterval != "" {
		if duration, err := time.ParseDuration(fileConfig.PollInterval); err == nil {
			cfg.PollInterval = duration
		}
	}
	if fileConfig.CryptoKey != "" {
		cfg.CryptoKey = fileConfig.CryptoKey
	}
}

func applyAgentFlagValues(cfg *AgentConfig, f *agentFlags) {
	defaults := DefaultAgentConfig()
	if f.pollSec != int(defaults.PollInterval.Seconds()) {
		cfg.PollInterval = time.Duration(f.pollSec) * time.Second
	}
	if f.repSec != int(defaults.ReportInterval.Seconds()) {
		cfg.ReportInterval = time.Duration(f.repSec) * time.Second
	}
	if f.key != "" {
		cfg.Key = f.key
	}
	if f.cryptoKey != "" {
		cfg.CryptoKey = f.cryptoKey
	}
	if f.rateLimit > 0 {
		cfg.RateLimit = f.rateLimit
	}
}

func applyAgentEnvConfig(cfg *AgentConfig) {
	if envAddr := os.Getenv("ADDRESS"); envAddr != "" {
		cfg.ServerURL = envAddr
	}
	if envKey := os.Getenv("KEY"); envKey != "" {
		cfg.Key = envKey
	}
	if envCryptoKey := os.Getenv("CRYPTO_KEY"); envCryptoKey != "" {
		cfg.CryptoKey = envCryptoKey
	}
	if envReportInterval := os.Getenv("REPORT_INTERVAL"); envReportInterval != "" {
		if sec, err := strconv.Atoi(envReportInterval); err == nil {
			cfg.ReportInterval = time.Duration(sec) * time.Second
		}
	}
	if envPoolInterval := os.Getenv("POLL_INTERVAL"); envPoolInterval != "" {
		if sec, err := strconv.Atoi(envPoolInterval); err == nil {
			cfg.PollInterval = time.Duration(sec) * time.Second
		}
	}
	if envRateLimit := os.Getenv("RATE_LIMIT"); envRateLimit != "" {
		if rl, err := strconv.Atoi(envRateLimit); err == nil && rl > 0 {
			cfg.RateLimit = rl
		}
	}
}

func ParseAgentConfig() *AgentConfig {
	cfg := DefaultAgentConfig()
	f := parseAgentFlags(cfg)
	applyAgentFileConfig(cfg, f.configFile)
	applyAgentFlagValues(cfg, f)
	applyAgentEnvConfig(cfg)
	return cfg
}
