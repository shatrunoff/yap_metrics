package service

import (
	"log"
	"sync"
	"time"

	"github.com/shatrunoff/yap_metrics/internal/storage"
)

type FileStorageService struct {
	storage       *storage.MemStorage
	filePath      string
	storeInterval time.Duration
	doneChan      chan struct{}
	wg            sync.WaitGroup
	mu            sync.Mutex
}

func NewFileStorageService(storage *storage.MemStorage, filePath string, storeInterval time.Duration) *FileStorageService {
	return &FileStorageService{
		storage:       storage,
		filePath:      filePath,
		storeInterval: storeInterval,
		doneChan:      make(chan struct{}),
	}
}

func (fss *FileStorageService) Start() {
	if fss.storeInterval > 0 {
		// Периодическое сохранение
		fss.wg.Add(1)
		go func() {
			defer fss.wg.Done()
			fss.startPeriodicSave()
		}()
	}
}

func (fss *FileStorageService) startPeriodicSave() {
	ticker := time.NewTicker(fss.storeInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := fss.storage.SaveToFile(fss.filePath); err != nil {
				log.Printf("ERROR: failed to save metrics to file: %v", err)
			} else {
				log.Printf("Metrics successfully saved to %s", fss.filePath)
			}
		case <-fss.doneChan:
			// Сохраняем при завершении
			if err := fss.storage.SaveToFile(fss.filePath); err != nil {
				log.Printf("ERROR: failed to save metrics on shutdown: %v", err)
			}
			return
		}
	}
}

func (fss *FileStorageService) Stop() {
	close(fss.doneChan)
	fss.wg.Wait()
}

// Синхронное сохранение (для режима storeInterval=0)
func (fss *FileStorageService) SaveSync() error {
	fss.mu.Lock()
	defer fss.mu.Unlock()

	return fss.storage.SaveToFile(fss.filePath)
}
