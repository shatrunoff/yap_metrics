package audit

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"sync"
	"time"

	"go.uber.org/zap"
)

// AuditEvent представляет событие аудита
type AuditEvent struct {
	Timestamp int64    `json:"ts"`
	Metrics   []string `json:"metrics"`
	IPAddress string   `json:"ip_address"`
}

// Observer интерфейс для паттерна Наблюдатель
type Observer interface {
	OnEvent(event AuditEvent) error
}

// AuditNotifier уведомляет наблюдателей о событиях
type AuditNotifier struct {
	observers []Observer
	mu        sync.RWMutex
	logger    *zap.Logger
}

// NewAuditNotifier создает новый AuditNotifier
func NewAuditNotifier(logger *zap.Logger) *AuditNotifier {
	return &AuditNotifier{
		observers: make([]Observer, 0),
		logger:    logger,
	}
}

// Subscribe добавляет наблюдателя
func (n *AuditNotifier) Subscribe(observer Observer) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.observers = append(n.observers, observer)
}

// Notify уведомляет всех наблюдателей о событии
func (n *AuditNotifier) Notify(event AuditEvent) {
	n.mu.RLock()
	defer n.mu.RUnlock()

	for _, observer := range n.observers {
		if err := observer.OnEvent(event); err != nil {
			if n.logger != nil {
				n.logger.Error("Failed to notify observer", zap.Error(err))
			}
		}
	}
}

// HasObservers проверяет, есть ли подписанные наблюдатели
func (n *AuditNotifier) HasObservers() bool {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return len(n.observers) > 0
}

// FileObserver записывает события аудита в файл
type FileObserver struct {
	filePath string
	mu       sync.Mutex
}

// NewFileObserver создает FileObserver
func NewFileObserver(filePath string) *FileObserver {
	return &FileObserver{
		filePath: filePath,
	}
}

// OnEvent записывает событие в файл
func (f *FileObserver) OnEvent(event AuditEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	data, err := json.Marshal(event)
	if err != nil {
		return err
	}

	file, err := os.OpenFile(f.filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.Write(append(data, '\n'))
	return err
}

// URLObserver отправляет события аудита по HTTP
type URLObserver struct {
	url    string
	client *http.Client
}

// NewURLObserver создает URLObserver
func NewURLObserver(url string) *URLObserver {
	return &URLObserver{
		url: url,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// OnEvent отправляет событие по HTTP POST
func (u *URLObserver) OnEvent(event AuditEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}

	resp, err := u.client.Post(u.url, "application/json", bytes.NewReader(data))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

// CreateEvent создает событие аудита
func CreateEvent(metricNames []string, ipAddress string) AuditEvent {
	return AuditEvent{
		Timestamp: time.Now().Unix(),
		Metrics:   metricNames,
		IPAddress: ipAddress,
	}
}
