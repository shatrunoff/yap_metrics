package main

import (
	"context"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/shatrunoff/yap_metrics/internal/audit"
	"github.com/shatrunoff/yap_metrics/internal/config"
	"github.com/shatrunoff/yap_metrics/internal/handler"
	"github.com/shatrunoff/yap_metrics/internal/middleware"
	"github.com/shatrunoff/yap_metrics/internal/service"
	"github.com/shatrunoff/yap_metrics/internal/storage"
	"github.com/shatrunoff/yap_metrics/internal/utils"
)

// initServer собирает все зависимости и возвращает http.Server и функцию очистки ресурсов
func initServer(cfg *config.ServerConfig) (*http.Server, func(), error) {
	// Конфигурация хранилища
	storageConfig := &storage.Config{
		DatabaseDSN:     cfg.DatabaseDSN,
		FileStoragePath: cfg.FileStoragePath,
		Restore:         cfg.Restore,
	}

	// Создаем хранилище (PostgreSQL -> файл -> память)
	storageInstance, err := storage.NewStorage(storageConfig)
	if err != nil {
		return nil, nil, err
	}

	// Настраиваем файловый сервис при необходимости
	var fileService *service.FileStorageService

	if cfg.DatabaseDSN == "" && cfg.FileStoragePath != "" {
		if fileSaver, ok := storageInstance.(interface {
			SaveToFile(path string) error
			LoadFromFile(filename string) error
		}); ok {
			fileService = service.NewFileStorageService(fileSaver, cfg.FileStoragePath, cfg.StoreInterval)
			fileService.Start()

			// Логируем ошибки файлового сервиса в фоне
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

	// Определяем необходимость синхронного сохранения (только для файлового хранилища)
	syncSave := cfg.StoreInterval == 0 && cfg.DatabaseDSN == ""

	// Создаем аудит-нотификатор
	var auditNotifier *audit.AuditNotifier
	if cfg.AuditFile != "" || cfg.AuditURL != "" {
		// Инициализируем логгер для аудита
		if err := middleware.InitLogger(); err != nil {
			log.Printf("Failed to init logger for audit: %v", err)
		}
		auditNotifier = audit.NewAuditNotifier(middleware.GetLogger())

		// Добавляем FileObserver
		if cfg.AuditFile != "" {
			auditNotifier.Subscribe(audit.NewFileObserver(cfg.AuditFile))
			log.Printf("Audit file observer registered: %s", cfg.AuditFile)
		}

		// Добавляем URLObserver
		if cfg.AuditURL != "" {
			auditNotifier.Subscribe(audit.NewURLObserver(cfg.AuditURL))
			log.Printf("Audit URL observer registered: %s", cfg.AuditURL)
		}
	}

	// Сборка HTTP-хендлера и сервера
	var serverHandler http.Handler
	if cfg.CryptoKey != "" {
		serverHandler = handler.NewHandlerWithCrypto(storageInstance, fileService, syncSave, cfg.Key, auditNotifier, cfg.CryptoKey, cfg.TrustedSubnet)
	} else {
		serverHandler = handler.NewHandler(storageInstance, fileService, syncSave, cfg.Key, auditNotifier, cfg.TrustedSubnet)
	}
	server := &http.Server{Addr: cfg.ServerURL, Handler: serverHandler}

	// Функция очистки
	cleanup := func() {
		if fileService != nil {
			fileService.Stop()
		}
		if err := storageInstance.Close(); err != nil {
			log.Printf("Storage close error: %v", err)
		}
	}

	return server, cleanup, nil
}

func main() {

	// Выводим информацию о сборке
	utils.PrintBuildInfo()

	cfg := config.ParseServerConfig()

	log.Printf("Starting server with config: Address=%s, StoreInterval=%v, FileStoragePath=%s, Restore=%v, DatabaseDSN=%v",
		cfg.ServerURL, cfg.StoreInterval, cfg.FileStoragePath, cfg.Restore, cfg.DatabaseDSN != "")

	server, cleanup, err := initServer(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize server: %v", err)
	}
	defer cleanup()

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
	signal.Notify(stopChan, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)
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
