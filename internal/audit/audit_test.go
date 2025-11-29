package audit

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"go.uber.org/zap"
)

func TestCreateEvent(t *testing.T) {
	metricNames := []string{"cpu", "memory"}
	ipAddress := "192.168.1.1"

	event := CreateEvent(metricNames, ipAddress)

	if event.Timestamp == 0 {
		t.Error("expected non-zero timestamp")
	}
	if len(event.Metrics) != 2 {
		t.Errorf("expected 2 metrics, got %d", len(event.Metrics))
	}
	if event.IPAddress != ipAddress {
		t.Errorf("expected IP %s, got %s", ipAddress, event.IPAddress)
	}
}

func TestNewAuditNotifier(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	notifier := NewAuditNotifier(logger)

	if notifier == nil {
		t.Error("expected non-nil notifier")
	}
	if notifier.logger == nil {
		t.Error("expected logger to be set")
	}
	if notifier.observers == nil {
		t.Error("expected observers to be initialized")
	}
}

func TestAuditNotifier_Subscribe(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	notifier := NewAuditNotifier(logger)

	observer := &mockObserver{}
	notifier.Subscribe(observer)

	if !notifier.HasObservers() {
		t.Error("expected to have observers")
	}
}

func TestAuditNotifier_Notify(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	notifier := NewAuditNotifier(logger)

	observer := &mockObserver{}
	notifier.Subscribe(observer)

	event := CreateEvent([]string{"test"}, "127.0.0.1")
	notifier.Notify(event)

	if !observer.called {
		t.Error("expected observer to be called")
	}
	if observer.event.IPAddress != "127.0.0.1" {
		t.Errorf("expected IP 127.0.0.1, got %s", observer.event.IPAddress)
	}
}

func TestAuditNotifier_HasObservers(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	notifier := NewAuditNotifier(logger)

	if notifier.HasObservers() {
		t.Error("expected no observers initially")
	}

	observer := &mockObserver{}
	notifier.Subscribe(observer)

	if !notifier.HasObservers() {
		t.Error("expected to have observers after subscription")
	}
}

func TestFileObserver_OnEvent(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "audit.log")

	observer := NewFileObserver(filePath)
	event := CreateEvent([]string{"cpu", "memory"}, "192.168.1.1")

	err := observer.OnEvent(event)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}

	var savedEvent AuditEvent
	err = json.Unmarshal(data, &savedEvent)
	if err != nil {
		t.Fatalf("failed to unmarshal event: %v", err)
	}

	if savedEvent.IPAddress != "192.168.1.1" {
		t.Errorf("expected IP 192.168.1.1, got %s", savedEvent.IPAddress)
	}
	if len(savedEvent.Metrics) != 2 {
		t.Errorf("expected 2 metrics, got %d", len(savedEvent.Metrics))
	}
}

func TestFileObserver_OnEvent_Multiple(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "audit.log")

	observer := NewFileObserver(filePath)

	event1 := CreateEvent([]string{"cpu"}, "192.168.1.1")
	event2 := CreateEvent([]string{"memory"}, "192.168.1.2")

	observer.OnEvent(event1)
	observer.OnEvent(event2)

	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}

	lines := 0
	for _, b := range data {
		if b == '\n' {
			lines++
		}
	}

	if lines != 2 {
		t.Errorf("expected 2 lines, got %d", lines)
	}
}

func TestURLObserver_OnEvent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected application/json content type, got %s", r.Header.Get("Content-Type"))
		}

		var event AuditEvent
		json.NewDecoder(r.Body).Decode(&event)

		if event.IPAddress != "192.168.1.1" {
			t.Errorf("expected IP 192.168.1.1, got %s", event.IPAddress)
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	observer := NewURLObserver(server.URL)
	event := CreateEvent([]string{"cpu"}, "192.168.1.1")

	err := observer.OnEvent(event)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestURLObserver_OnEvent_Timeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(15 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	observer := NewURLObserver(server.URL)
	event := CreateEvent([]string{"cpu"}, "192.168.1.1")

	err := observer.OnEvent(event)
	if err == nil {
		t.Error("expected timeout error")
	}
}

type mockObserver struct {
	called bool
	event  AuditEvent
	err    error
}

func (m *mockObserver) OnEvent(event AuditEvent) error {
	m.called = true
	m.event = event
	return m.err
}

func TestAuditNotifier_Notify_WithError(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	notifier := NewAuditNotifier(logger)

	observer := &mockObserver{err: os.ErrPermission}
	notifier.Subscribe(observer)

	event := CreateEvent([]string{"test"}, "127.0.0.1")
	notifier.Notify(event)

	if !observer.called {
		t.Error("expected observer to be called even with error")
	}
}
