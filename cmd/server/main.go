package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/shatrunoff/yap_metrics/internal/config"
	"github.com/shatrunoff/yap_metrics/internal/handler"
	"github.com/shatrunoff/yap_metrics/internal/repository"
	"github.com/shatrunoff/yap_metrics/internal/service"
	"github.com/shatrunoff/yap_metrics/internal/storage"
)

func main() {
	cfg := config.ParseServerConfig()

	log.Printf("Starting server with config: Address=%s, StoreInterval=%v, FileStoragePath=%s, Restore=%v, DatabaseDSN=%v",
		cfg.ServerURL, cfg.StoreInterval, cfg.FileStoragePath, cfg.Restore, cfg.DatabaseDSN != "")

	var storageInstance storage.Storage
	var fileService *service.FileStorageService

	// Приоритет 1: PostgreSQL если указан DSN
	if cfg.DatabaseDSN != "" {
		log.Printf("Initializing PostgreSQL storage...")

		// Сначала подключаемся к БД
		db, err := connectToDatabase(cfg.DatabaseDSN)
		if err != nil {
			log.Fatalf("Failed to connect to database: %v", err)
		}
		defer db.Close()

		// Запускаем миграции ДО создания хранилища
		log.Printf("Running database migrations...")
		migrator := repository.NewMigrator(db)
		if err := migrator.RunMigrations(); err != nil {
			log.Fatalf("Failed to run migrations: %v", err)
		}
		log.Printf("Database migrations completed successfully")

		// Теперь создаем PostgreSQL хранилище
		pgStorage, err := storage.NewPostgresStorageFromDB(db)
		if err != nil {
			log.Fatalf("Failed to create PostgreSQL storage: %v", err)
		}
		storageInstance = pgStorage

		log.Printf("PostgreSQL storage initialized successfully")

		// Для PostgreSQL файловый сервис не нужен
		fileService = service.NewFileStorageService(nil, "", 0)

	} else if cfg.FileStoragePath != "" {
		// Приоритет 2: Файловое хранилище
		log.Printf("Initializing file storage...")
		memStorage := storage.NewMemStorage()

		if cfg.Restore {
			if err := memStorage.LoadFromFile(cfg.FileStoragePath); err != nil {
				log.Printf("WARNING: failed to load metrics from file: %v", err)
			} else {
				log.Printf("Metrics loaded from %s", cfg.FileStoragePath)
			}
		}

		storageInstance = memStorage
		fileService = service.NewFileStorageService(memStorage, cfg.FileStoragePath, cfg.StoreInterval)
		fileService.Start()
		defer fileService.Stop()

		go func() {
			for err := range fileService.Err() {
				log.Printf("File storage error: %v", err)
			}
		}()

	} else {
		// Приоритет 3: In-memory хранилище
		log.Printf("Initializing in-memory storage...")
		storageInstance = storage.NewMemStorage()
		fileService = service.NewFileStorageService(nil, "", 0)
	}

	// Определяем, нужно ли синхронное сохранение (только для файлового хранилища)
	syncSave := cfg.StoreInterval == 0 && cfg.DatabaseDSN == ""

	serverHandler := handler.NewHandler(storageInstance, fileService, syncSave)

	server := &http.Server{
		Addr:    cfg.ServerURL,
		Handler: serverHandler,
	}

	go func() {
		log.Printf("Server started on %s", server.Addr)
		if cfg.DatabaseDSN != "" {
			log.Printf("Using PostgreSQL storage")
		} else if cfg.FileStoragePath != "" {
			log.Printf("Using file storage: %s", cfg.FileStoragePath)
		} else {
			log.Printf("Using in-memory storage")
		}

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, os.Interrupt, syscall.SIGTERM)
	<-stopChan

	log.Printf("Shutting down server...")
	server.Shutdown(context.Background())
	log.Printf("Server stopped")
}

func connectToDatabase(dsn string) (*sql.DB, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Проверяем соединение
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Настраиваем пул соединений
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	return db, nil
}
