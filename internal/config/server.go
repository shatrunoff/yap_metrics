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
	Key             string
	AuditFile       string
	AuditURL        string
	CryptoKey       string
}

func DefaultServerConfig() *ServerConfig {
	return &ServerConfig{
		ServerURL:       "localhost:8080",
		StoreInterval:   300 * time.Second,
		FileStoragePath: "tmp/my-metrics.json",
		Restore:         true,
		DatabaseDSN:     "",
		Key:             "",
		AuditFile:       "",
		AuditURL:        "",
		CryptoKey:       "",
	}
}

func ParseServerConfig() *ServerConfig {
	cfg := DefaultServerConfig()

	// Флаги командной строки
	var storeIntervalSec int
	var configFile string
	flag.StringVar(&cfg.ServerURL, "a", cfg.ServerURL, "Server address host:port")
	flag.IntVar(&storeIntervalSec, "i", int(cfg.StoreInterval.Seconds()), "Store interval in seconds")
	flag.StringVar(&cfg.FileStoragePath, "f", cfg.FileStoragePath, "File storage path")
	flag.BoolVar(&cfg.Restore, "r", cfg.Restore, "Restore from file")
	flag.StringVar(&cfg.DatabaseDSN, "d", cfg.DatabaseDSN, "Database DSN")
	flag.StringVar(&cfg.Key, "k", cfg.Key, "Signing key for HashSHA256 header")
	flag.StringVar(&cfg.CryptoKey, "crypto-key", cfg.CryptoKey, "Path to private key file for decryption")
	flag.StringVar(&cfg.AuditFile, "audit-file", cfg.AuditFile, "Audit log file path")
	flag.StringVar(&cfg.AuditURL, "audit-url", cfg.AuditURL, "Audit log URL")
	flag.StringVar(&configFile, "c", "", "Config file path")
	flag.StringVar(&configFile, "config", "", "Config file path")
	flag.Parse()

	// Получаем путь к файлу конфигурации из переменной окружения, если не задан флагом
	if configFile == "" {
		configFile = os.Getenv("CONFIG")
	}

	// Загружаем конфигурацию из файла (приоритет ниже флагов и переменных окружения)
	if fileConfig, err := LoadServerConfigFromFile(configFile); err == nil && fileConfig != nil {
		if fileConfig.Address != "" {
			cfg.ServerURL = fileConfig.Address
		}
		if fileConfig.StoreInterval != "" {
			if duration, err := time.ParseDuration(fileConfig.StoreInterval); err == nil {
				cfg.StoreInterval = duration
			}
		}
		if fileConfig.StoreFile != "" {
			cfg.FileStoragePath = fileConfig.StoreFile
		}
		if fileConfig.Restore != nil {
			cfg.Restore = *fileConfig.Restore
		}
		if fileConfig.DatabaseDSN != "" {
			cfg.DatabaseDSN = fileConfig.DatabaseDSN
		}
		if fileConfig.CryptoKey != "" {
			cfg.CryptoKey = fileConfig.CryptoKey
		}
	}

	// Применяем storeIntervalSec из флагов
	if storeIntervalSec != int(DefaultServerConfig().StoreInterval.Seconds()) {
		cfg.StoreInterval = time.Duration(storeIntervalSec) * time.Second
	}

	// Переменные окружения (наивысший приоритет)
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
	if envKey := os.Getenv("KEY"); envKey != "" {
		cfg.Key = envKey
	}
	if envCryptoKey := os.Getenv("CRYPTO_KEY"); envCryptoKey != "" {
		cfg.CryptoKey = envCryptoKey
	}
	if envAuditFile := os.Getenv("AUDIT_FILE"); envAuditFile != "" {
		cfg.AuditFile = envAuditFile
	}
	if envAuditURL := os.Getenv("AUDIT_URL"); envAuditURL != "" {
		cfg.AuditURL = envAuditURL
	}

	return cfg
}
