package service

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// интерфейс для сохранения в файл
type FileSaver interface {
	SaveToFile(path string) error
}

// сервис для работы с файловым хранилищем
type FileStorageService struct {
	saver         FileSaver
	filePath      string
	storeInterval time.Duration

	ctx    context.Context
	cancel context.CancelFunc

	wg    sync.WaitGroup
	errCh chan error
}

// создает сервис работы с файлами
func NewFileStorageService(
	saver FileSaver,
	filePath string,
	storeInterval time.Duration,
) *FileStorageService {
	ctx, cancel := context.WithCancel(context.Background())
	return &FileStorageService{
		saver:         saver,
		filePath:      filePath,
		storeInterval: storeInterval,
		ctx:           ctx,
		cancel:        cancel,
		errCh:         make(chan error, 1),
	}
}

// запускает периодическое сохранение
func (fss *FileStorageService) Start() {
	if fss.storeInterval <= 0 || fss.saver == nil {
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
			if err := fss.saver.SaveToFile(fss.filePath); err != nil {
				select {
				case fss.errCh <- fmt.Errorf("periodic save failed: %w", err):
				default:
				}
			}

		case <-fss.ctx.Done():
			if err := fss.saver.SaveToFile(fss.filePath); err != nil {
				select {
				case fss.errCh <- fmt.Errorf("shutdown save failed: %w", err):
				default:
				}
			}
			return
		}
	}
}

// выполняет синхронное сохранение
func (fss *FileStorageService) SaveSync() error {
	if fss.saver == nil {
		return nil
	}
	return fss.saver.SaveToFile(fss.filePath)
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
