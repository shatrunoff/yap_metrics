package service

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// сохранение метрик в файл
type Saver interface {
	SaveToFile(path string) error
}

// запуск, остановка сохранения
type FileStorageService struct {
	storage       Saver
	filePath      string
	storeInterval time.Duration

	ctx    context.Context
	cancel context.CancelFunc

	wg    sync.WaitGroup
	errCh chan error
}

// создаёт сервис работы с файлами
func NewFileStorageService(
	storage Saver,
	filePath string,
	storeInterval time.Duration,
) *FileStorageService {
	ctx, cancel := context.WithCancel(context.Background())
	return &FileStorageService{
		storage:       storage,
		filePath:      filePath,
		storeInterval: storeInterval,
		ctx:           ctx,
		cancel:        cancel,
		errCh:         make(chan error, 1),
	}
}

// запускает периодическое сохранение
func (fss *FileStorageService) Start() {
	if fss.storeInterval <= 0 {
		return
	}

	fss.wg.Add(1)
	go func() {
		defer fss.wg.Done()
		fss.startPeriodicSave()
	}()
}

func (fss *FileStorageService) startPeriodicSave() {
	ticker := time.NewTicker(fss.storeInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := fss.storage.SaveToFile(fss.filePath); err != nil {
				select {
				case fss.errCh <- fmt.Errorf("failed to save metrics: %w", err):
				default:
				}
			}
		case <-fss.ctx.Done():
			if err := fss.storage.SaveToFile(fss.filePath); err != nil {
				select {
				case fss.errCh <- fmt.Errorf("failed to save metrics on shutdown: %w", err):
				default:
				}
			}
			return
		}
	}
}

// выполняет синхронное сохранение
func (fss *FileStorageService) SaveSync() error {
	return fss.storage.SaveToFile(fss.filePath)
}

// завершает работу сервиса
func (fss *FileStorageService) Stop() {
	fss.cancel()
	fss.wg.Wait()
	close(fss.errCh)
}

// возвращает канал для получения ошибок
func (fss *FileStorageService) Err() <-chan error {
	return fss.errCh
}
