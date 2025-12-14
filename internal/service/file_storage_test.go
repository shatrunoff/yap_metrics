package service

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockFileSaver — имитация FileSaver
type MockFileSaver struct {
	mu          sync.Mutex
	saveCount   int
	lastError   error
	failOnSave  bool
	saveHistory []string
}

func (m *MockFileSaver) SaveToFile(path string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.saveHistory = append(m.saveHistory, path)
	m.saveCount++

	if m.failOnSave {
		m.lastError = errors.New("mock save error")
		return m.lastError
	}
	m.lastError = nil
	return nil
}

func (m *MockFileSaver) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.saveCount = 0
	m.lastError = nil
	m.failOnSave = false
	m.saveHistory = nil
}

func (m *MockFileSaver) SaveCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.saveCount
}

func TestNewFileStorageService(t *testing.T) {
	saver := &MockFileSaver{}
	svc := NewFileStorageService(saver, "/tmp/test.json", 1*time.Second)
	assert.NotNil(t, svc)
	assert.Equal(t, saver, svc.saver)
	assert.Equal(t, "/tmp/test.json", svc.filePath)
}

func TestFileStorageService_SaveSync(t *testing.T) {
	saver := &MockFileSaver{}
	svc := NewFileStorageService(saver, "/tmp/test.json", 0)

	err := svc.SaveSync()
	require.NoError(t, err)
	assert.Equal(t, 1, saver.SaveCount())
}

func TestFileStorageService_SaveSync_NilSaver(t *testing.T) {
	svc := NewFileStorageService(nil, "/tmp/test.json", 0)
	err := svc.SaveSync()
	require.NoError(t, err) // должно работать без паники
}

func TestFileStorageService_Start_WithZeroInterval(t *testing.T) {
	saver := &MockFileSaver{}
	svc := NewFileStorageService(saver, "/tmp/test.json", 0)
	svc.Start()
	svc.Stop()

	// Никаких сохранений не должно быть, кроме явных
	assert.Equal(t, 0, saver.SaveCount())
}

func TestFileStorageService_Start_WithPeriodicSave(t *testing.T) {
	saver := &MockFileSaver{}
	interval := 100 * time.Millisecond
	svc := NewFileStorageService(saver, "/tmp/test.json", interval)
	svc.Start()

	// Ждём 2-3 тика
	time.Sleep(250 * time.Millisecond)

	svc.Stop()

	saveCount := saver.SaveCount()
	// Должно быть 2 или 3 сохранения (2 периодических + 1 при shutdown)
	assert.GreaterOrEqual(t, saveCount, 2)
	assert.LessOrEqual(t, saveCount, 4)
}

func TestFileStorageService_Stop_ShutdownSaves(t *testing.T) {
	saver := &MockFileSaver{}
	svc := NewFileStorageService(saver, "/tmp/test.json", 1*time.Hour)
	svc.Start()

	// Сразу останавливаем — должен быть 1 save (shutdown)
	svc.Stop()

	assert.Equal(t, 1, saver.SaveCount())
}

func TestFileStorageService_ErrorChannel(t *testing.T) {
	saver := &MockFileSaver{failOnSave: true}
	svc := NewFileStorageService(saver, "/tmp/test.json", 50*time.Millisecond)
	svc.Start()

	// Ждём первую ошибку
	select {
	case err := <-svc.Err():
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "periodic save failed")
	case <-time.After(200 * time.Millisecond):
		t.Fatal("expected error, got timeout")
	}

	svc.Stop()
}

func TestFileStorageService_ShutdownError(t *testing.T) {
	saver := &MockFileSaver{failOnSave: true}
	svc := NewFileStorageService(saver, "/tmp/test.json", 1*time.Hour)
	svc.Start()

	// Сразу завершаем — должен прийти ошибка shutdown save
	svc.Stop()

	select {
	case err := <-svc.Err():
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "shutdown save failed")
	case <-time.After(100 * time.Millisecond):
		t.Fatal("expected shutdown error")
	}
}

func TestFileStorageService_ErrChannelClosedAfterStop(t *testing.T) {
	saver := &MockFileSaver{}
	svc := NewFileStorageService(saver, "/tmp/test.json", 1*time.Hour)
	svc.Start()
	svc.Stop()

	// Канал должен быть закрыт — попытка чтения вернёт (zero, false)
	_, ok := <-svc.Err()
	assert.False(t, ok, "error channel should be closed after Stop")
}

func TestFileStorageService_StartWithoutSaver(t *testing.T) {
	svc := NewFileStorageService(nil, "/tmp/test.json", 100*time.Millisecond)
	svc.Start()
	time.Sleep(150 * time.Millisecond)
	svc.Stop()

	// Никаких паник, никаких операций
	// Тест проходит, если не упал
}
