package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/shatrunoff/yap_metrics/internal/config"
	"github.com/shatrunoff/yap_metrics/internal/handler"
	"github.com/shatrunoff/yap_metrics/internal/service"
	"github.com/shatrunoff/yap_metrics/internal/storage"
)

func main() {
	cfg := config.ParseServerConfig()

	log.Printf("Starting server with config: Address=%s, StoreInterval=%v, FileStoragePath=%s, Restore=%v, DatabaseDSN=%v",
		cfg.ServerURL, cfg.StoreInterval, cfg.FileStoragePath, cfg.Restore, cfg.DatabaseDSN != "")

	// Конфигурация хранилища
	storageConfig := &storage.Config{
		DatabaseDSN:     cfg.DatabaseDSN,
		FileStoragePath: cfg.FileStoragePath,
		Restore:         cfg.Restore,
	}

	// Создаем хранилище с приоритетом:
	// PostgreSQL -> файл -> память
	storageInstance, err := storage.NewStorage(storageConfig)
	if err != nil {
		log.Fatalf("Failed to initialize storage: %v", err)
	}
	defer storageInstance.Close()

	var fileService *service.FileStorageService

	// Для файлового хранилища создаем файловый сервис
	if cfg.DatabaseDSN == "" && cfg.FileStoragePath != "" {
		// Проверяем, поддерживает ли хранилище файловые операции
		if fileSaver, ok := storageInstance.(interface {
			SaveToFile(path string) error
			LoadFromFile(filename string) error
		}); ok {
			fileService = service.NewFileStorageService(fileSaver, cfg.FileStoragePath, cfg.StoreInterval)
			fileService.Start()
			defer fileService.Stop()

			go func() {
				for err := range fileService.Err() {
					log.Printf("File storage error: %v", err)
				}
			}()
		} else {
			log.Printf("WARNING: Storage doesn't support file operations, using in-memory only")
			fileService = service.NewFileStorageService(nil, "", 0)
		}
	} else {
		// Для PostgreSQL или чистого in-memory создаем пустой файловый сервис
		fileService = service.NewFileStorageService(nil, "", 0)
	}

	// Определяем, нужно ли синхронное сохранение (только для файлового хранилища)
	syncSave := cfg.StoreInterval == 0 && cfg.DatabaseDSN == ""

	serverHandler := handler.NewHandler(storageInstance, fileService, syncSave)

	server := &http.Server{
		Addr:    cfg.ServerURL,
		Handler: serverHandler,
	}

	// Канал для сигнала о готовности сервера
	ready := make(chan bool, 1)

	go func() {
		log.Printf("Server starting on %s", server.Addr)

		// Логируем тип используемого хранилища
		if cfg.DatabaseDSN != "" {
			log.Printf("Using PostgreSQL storage")
		} else if cfg.FileStoragePath != "" {
			log.Printf("Using file storage: %s", cfg.FileStoragePath)
		} else {
			log.Printf("Using in-memory storage")
		}

		ready <- true

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Ждем запуска сервера
	<-ready
	log.Printf("Server started successfully")

	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, os.Interrupt, syscall.SIGTERM)
	<-stopChan

	log.Printf("Shutting down server...")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Server shutdown error: %v", err)
	}

	log.Printf("Server stopped")
}
