package main

import (
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/shatrunoff/yap_metrics/internal/config"
	"github.com/shatrunoff/yap_metrics/internal/handler"
	"github.com/shatrunoff/yap_metrics/internal/service"
	"github.com/shatrunoff/yap_metrics/internal/storage"
)

func main() {
	cfg := config.ParseServerConfig()
	memStorage := storage.NewMemStorage()

	// Загрузка метрик при старте
	if cfg.Restore {
		if err := memStorage.LoadFromFile(cfg.FileStoragePath); err != nil {
			log.Printf("WARNING: failed to load metrics from file: %v", err)
		} else {
			log.Printf("Metrics loaded from %s", cfg.FileStoragePath)
		}
	}

	// Создаем сервис для сохранения метрик
	fileService := service.NewFileStorageService(memStorage, cfg.FileStoragePath, cfg.StoreInterval)

	// Запускаем периодическое сохранение (если интервал не 0)
	fileService.Start()
	defer fileService.Stop()

	go func() {
		for err := range fileService.Err() {
			log.Printf("File storage error: %v", err)
		}
	}()

	// Создаем хэндлер с поддержкой синхронного сохранения
	serverHandler := handler.NewHandler(memStorage, fileService, cfg.StoreInterval == 0)

	server := &http.Server{
		Addr:    cfg.ServerURL,
		Handler: serverHandler,
	}

	go func() {
		log.Printf("Server started on %s", server.Addr)
		log.Printf("Store interval: %v, File path: %s, Restore: %v",
			cfg.StoreInterval, cfg.FileStoragePath, cfg.Restore)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, os.Interrupt, syscall.SIGTERM)
	<-stopChan

	log.Printf("Server stopped on %s", server.Addr)
}
