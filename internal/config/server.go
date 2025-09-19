package config

import (
	"flag"
	"os"
	"strconv"
	"time"
)

type ServerConfig struct {
	ServerURL       string
	StoreInterval   time.Duration
	FileStoragePath string
	Restore         bool
	DatabaseDSN     string
}

func DefaultServerConfig() *ServerConfig {
	return &ServerConfig{
		ServerURL:       "localhost:8080",
		StoreInterval:   300 * time.Second,
		FileStoragePath: "tmp/my-metrics.json",
		Restore:         true,
		DatabaseDSN:     "",
	}
}

func ParseServerConfig() *ServerConfig {
	cfg := DefaultServerConfig()

	// Флаги командной строки
	var storeIntervalSec int
	flag.StringVar(&cfg.ServerURL, "a", cfg.ServerURL, "Server address host:port")
	flag.IntVar(&storeIntervalSec, "i", int(cfg.StoreInterval.Seconds()), "Store interval in seconds")
	flag.StringVar(&cfg.FileStoragePath, "f", cfg.FileStoragePath, "File storage path")
	flag.BoolVar(&cfg.Restore, "r", cfg.Restore, "Restore from file")
	flag.StringVar(&cfg.DatabaseDSN, "d", cfg.DatabaseDSN, "Database DSN")
	flag.Parse()

	// Переменные окружения
	if envAddr := os.Getenv("ADDRESS"); envAddr != "" {
		cfg.ServerURL = envAddr
	}
	if envInterval := os.Getenv("STORE_INTERVAL"); envInterval != "" {
		if sec, err := strconv.Atoi(envInterval); err == nil {
			cfg.StoreInterval = time.Duration(sec) * time.Second
		}
	}
	if envPath := os.Getenv("FILE_STORAGE_PATH"); envPath != "" {
		cfg.FileStoragePath = envPath
	}
	if envRestore := os.Getenv("RESTORE"); envRestore != "" {
		if restore, err := strconv.ParseBool(envRestore); err == nil {
			cfg.Restore = restore
		}
	}
	if envDSN := os.Getenv("DATABASE_DSN"); envDSN != "" {
		cfg.DatabaseDSN = envDSN
	}

	return cfg
}
