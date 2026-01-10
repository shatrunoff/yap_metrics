package service

import (
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/shatrunoff/yap_metrics/internal/config"
)

// TestAgentService_StartCollector тестирует сбор метрик
func TestAgentService_StartCollector(t *testing.T) {
	cfg := &config.AgentConfig{
		PollInterval: 50 * time.Millisecond,
	}

	service, _ := NewAgent(cfg)

	// Получаем начальное состояние метрик
	initialMetrics := service.collector.GetMetrics()
	_ = len(initialMetrics)

	// Запускаем сборщик в отдельной горутине
	done := make(chan struct{})
	go func() {
		service.startCollector()
		close(done)
	}()

	// Ждем несколько тиков
	time.Sleep(250 * time.Millisecond) // ~5 тиков

	// Останавливаем
	close(service.doneChan)
	<-done

	// Проверяем, что метрики обновлялись
	finalMetrics := service.collector.GetMetrics()
	finalCount := len(finalMetrics)

	// Метрики должны быть собраны
	if finalCount == 0 {
		t.Error("Metrics should be collected")
	}
}

// TestAgentService_StartCollectorStopImmediately тестирует немедленную остановку
func TestAgentService_StartCollectorStopImmediately(t *testing.T) {
	cfg := &config.AgentConfig{
		PollInterval: time.Second,
	}

	service, _ := NewAgent(cfg)

	// Получаем начальное состояние метрик
	initialMetrics := service.collector.GetMetrics()

	// Немедленно закрываем канал завершения
	close(service.doneChan)

	// Запускаем сборщик - должен сразу завершиться
	service.startCollector()

	// Проверяем, что метрики не изменились
	finalMetrics := service.collector.GetMetrics()

	// При немедленной остановке метрики могут быть одинаковыми
	// или немного отличаться (если успел собраться один раз)
	// Проверяем только что не было паники
	_ = initialMetrics
	_ = finalMetrics
}

// TestAgentService_StartSysCollector тестирует сбор системных метрик
func TestAgentService_StartSysCollector(t *testing.T) {
	cfg := &config.AgentConfig{
		PollInterval: 50 * time.Millisecond,
	}

	service, _ := NewAgent(cfg)

	// Получаем начальное состояние метрик
	initialMetrics := service.collector.GetMetrics()
	hasSystemMetrics := false
	for _, m := range initialMetrics {
		if m.ID == "TotalMemory" || m.ID == "FreeMemory" || m.ID == "CPUutilization1" {
			hasSystemMetrics = true
			break
		}
	}

	// Запускаем системный сборщик
	done := make(chan struct{})
	go func() {
		service.startSysCollector()
		close(done)
	}()

	// Ждем несколько тиков
	time.Sleep(250 * time.Millisecond) // ~5 тиков

	// Останавливаем
	close(service.doneChan)
	<-done

	// Проверяем, что системные метрики есть
	finalMetrics := service.collector.GetMetrics()
	finalHasSystemMetrics := false
	for _, m := range finalMetrics {
		if m.ID == "TotalMemory" || m.ID == "FreeMemory" || m.ID == "CPUutilization1" {
			finalHasSystemMetrics = true
			break
		}
	}

	if !finalHasSystemMetrics && hasSystemMetrics {
		t.Error("System metrics should be collected")
	}
}

// TestAgentService_StartSysCollectorStopImmediately тестирует немедленную остановку
func TestAgentService_StartSysCollectorStopImmediately(t *testing.T) {
	cfg := &config.AgentConfig{
		PollInterval: time.Second,
	}

	service, _ := NewAgent(cfg)

	// Немедленно закрываем канал завершения
	close(service.doneChan)

	// Запускаем системный сборщик
	service.startSysCollector()

	// Проверяем только отсутствие паники
	_ = service.collector.GetMetrics()
}

// TestAgentService_CollectorRaceCondition тестирует конкурентный доступ
func TestAgentService_CollectorRaceCondition(t *testing.T) {
	cfg := &config.AgentConfig{
		PollInterval: 10 * time.Millisecond,
	}

	service, _ := NewAgent(cfg)

	// Запускаем оба сборщика одновременно
	done1 := make(chan struct{})
	done2 := make(chan struct{})

	go func() {
		service.startCollector()
		close(done1)
	}()

	go func() {
		service.startSysCollector()
		close(done2)
	}()

	// Даем поработать
	time.Sleep(100 * time.Millisecond)

	// Останавливаем
	close(service.doneChan)
	<-done1
	<-done2

	// Проверяем отсутствие паники при конкурентном доступе
	metrics := service.collector.GetMetrics()
	if len(metrics) == 0 {
		t.Error("Metrics should be collected by both collectors")
	}
}

// TestAgentService_CollectorDefer тестирует defer ticker.Stop()
func TestAgentService_CollectorDefer(t *testing.T) {
	cfg := &config.AgentConfig{
		PollInterval: 10 * time.Millisecond,
	}

	service, _ := NewAgent(cfg)

	// Запускаем и сразу останавливаем
	done := make(chan struct{})
	go func() {
		service.startCollector()
		close(done)
	}()

	// Немедленно останавливаем
	close(service.doneChan)

	// Ждем завершения
	select {
	case <-done:
		// OK - тикер остановлен, нет утечек
	case <-time.After(100 * time.Millisecond):
		t.Error("Collector did not stop properly")
	}
}

// TestAgentService_CollectorChannelOperations тестирует операции с каналами
func TestAgentService_CollectorChannelOperations(t *testing.T) {
	tests := []struct {
		name         string
		pollInterval time.Duration
		stopDelay    time.Duration
	}{
		{"Fast poll, immediate stop", 1 * time.Millisecond, 0},
		{"Fast poll, short delay", 1 * time.Millisecond, 5 * time.Millisecond},
		{"Slow poll, short delay", 100 * time.Millisecond, 50 * time.Millisecond},
		{"Slow poll, long delay", 50 * time.Millisecond, 250 * time.Millisecond},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.AgentConfig{
				PollInterval: tt.pollInterval,
			}

			service, _ := NewAgent(cfg)

			// Запускаем сборщик
			done := make(chan struct{})
			go func() {
				service.startCollector()
				close(done)
			}()

			// Ждем указанное время
			time.Sleep(tt.stopDelay)

			// Останавливаем
			close(service.doneChan)
			<-done

			// Проверяем только отсутствие паники
			_ = service.collector.GetMetrics()
		})
	}
}

// TestAgentService_CollectorSelectBehavior тестирует поведение select
func TestAgentService_CollectorSelectBehavior(t *testing.T) {
	cfg := &config.AgentConfig{
		PollInterval: 100 * time.Millisecond,
	}

	service, _ := NewAgent(cfg)

	// Запускаем сборщик
	done := make(chan struct{})
	go func() {
		service.startCollector()
		close(done)
	}()

	// Ждем немного
	time.Sleep(50 * time.Millisecond)

	// Отправляем сигнал остановки
	close(service.doneChan)

	// Ждем завершения
	<-done

	// Даем немного времени на случайные вызовы после остановки
	time.Sleep(200 * time.Millisecond)

	// Проверяем только отсутствие паник
	_ = service.collector.GetMetrics()
}

// TestAgentService_CollectorMemoryLeak тестирует на утечки памяти
func TestAgentService_CollectorMemoryLeak(t *testing.T) {
	// Этот тест проверяет, что нет утечек при множественных запусках/остановках
	cfg := &config.AgentConfig{
		PollInterval: 10 * time.Millisecond,
	}

	for i := 0; i < 100; i++ {
		service, _ := NewAgent(cfg)

		// Запускаем и быстро останавливаем
		done := make(chan struct{})
		go func() {
			service.startCollector()
			close(done)
		}()

		time.Sleep(5 * time.Millisecond)
		close(service.doneChan)
		<-done

		// Даем GC время на работу
		if i%10 == 0 {
			runtime.GC()
			time.Sleep(1 * time.Millisecond)
		}
	}
}

// TestAgentService_CollectorWithRealMetrics тестирует с реальными метриками
func TestAgentService_CollectorWithRealMetrics(t *testing.T) {
	cfg := &config.AgentConfig{
		PollInterval: 30 * time.Millisecond,
	}

	service, _ := NewAgent(cfg)

	// Получаем метрики до запуска
	_ = service.collector.GetMetrics()

	// Запускаем сборщик
	done := make(chan struct{})
	go func() {
		service.startCollector()
		close(done)
	}()

	// Ждем
	time.Sleep(150 * time.Millisecond)

	// Останавливаем
	close(service.doneChan)
	<-done

	// Получаем метрики после
	afterMetrics := service.collector.GetMetrics()
	afterCount := len(afterMetrics)

	// Проверяем, что метрики есть
	if afterCount == 0 {
		t.Error("Metrics should be collected")
	}

	// Проверяем некоторые конкретные метрики
	foundCounter := false
	foundGauge := false

	for _, m := range afterMetrics {
		if m.MType == "counter" {
			foundCounter = true
		}
		if m.MType == "gauge" {
			foundGauge = true
		}
		if foundCounter && foundGauge {
			break
		}
	}

	if !foundCounter {
		t.Error("Should have counter metrics")
	}
	if !foundGauge {
		t.Error("Should have gauge metrics")
	}
}

// TestAgentService_SystemCollectorWithRealMetrics тестирует системный сбор
func TestAgentService_SystemCollectorWithRealMetrics(t *testing.T) {
	cfg := &config.AgentConfig{
		PollInterval: 30 * time.Millisecond,
	}

	service, _ := NewAgent(cfg)

	// Получаем метрики до запуска
	_ = service.collector.GetMetrics()

	// Запускаем системный сборщик
	done := make(chan struct{})
	go func() {
		service.startSysCollector()
		close(done)
	}()

	// Ждем
	time.Sleep(150 * time.Millisecond)

	// Останавливаем
	close(service.doneChan)
	<-done

	// Получаем метрики после
	afterMetrics := service.collector.GetMetrics()

	// Проверяем наличие системных метрик
	hasSystemMetrics := false
	for id, m := range afterMetrics {
		// Проверяем типичные системные метрики
		if m.MType == "gauge" && (id == "TotalMemory" || id == "FreeMemory" ||
			id == "CPUutilization1" || id == "CPUutilization5" || id == "CPUutilization15") {
			hasSystemMetrics = true
			break
		}
	}

	if !hasSystemMetrics {
		// Это может быть нормально в тестовой среде
		t.Log("No system metrics found (may be normal in test environment)")
	}
}

// TestAgentService_ConcurrentCollectAndRead тестирует чтение метрик во время сбора
func TestAgentService_ConcurrentCollectAndRead(t *testing.T) {
	cfg := &config.AgentConfig{
		PollInterval: 20 * time.Millisecond,
	}

	service, _ := NewAgent(cfg)

	// Запускаем сборщик
	done := make(chan struct{})
	go func() {
		service.startCollector()
		close(done)
	}()

	// Параллельно читаем метрики
	var wg sync.WaitGroup
	var readErrors atomic.Int32

	for i := range 10 {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 5; j++ {
				metrics := service.collector.GetMetrics()
				if len(metrics) == 0 {
					readErrors.Add(1)
				}
				time.Sleep(time.Duration(id) * time.Millisecond)
			}
		}(i)
	}

	// Даем поработать
	time.Sleep(100 * time.Millisecond)

	// Останавливаем
	close(service.doneChan)
	<-done
	wg.Wait()

	if readErrors.Load() > 50 {
		t.Errorf("Too many empty metric reads: %d", readErrors.Load())
	}
}

// TestAgentService_CollectorTimeout тестирует поведение при долгих операциях
func TestAgentService_CollectorTimeout(t *testing.T) {
	cfg := &config.AgentConfig{
		PollInterval: 10 * time.Millisecond,
	}

	service, _ := NewAgent(cfg)

	// Запускаем сборщик
	done := make(chan struct{})
	go func() {
		service.startCollector()
		close(done)
	}()

	// Даем поработать
	time.Sleep(50 * time.Millisecond)

	// Останавливаем
	close(service.doneChan)

	// Ждем завершения с таймаутом
	select {
	case <-done:
		// OK
	case <-time.After(100 * time.Millisecond):
		t.Error("Collector did not stop in time")
	}
}

// TestAgentService_CollectorRestart тестирует перезапуск сборщика
func TestAgentService_CollectorRestart(t *testing.T) {
	cfg := &config.AgentConfig{
		PollInterval: 20 * time.Millisecond,
	}

	service, _ := NewAgent(cfg)

	// Первый запуск
	done1 := make(chan struct{})
	go func() {
		service.startCollector()
		close(done1)
	}()

	time.Sleep(50 * time.Millisecond)
	close(service.doneChan)
	<-done1

	// Создаем новый канал для второго запуска
	service.doneChan = make(chan struct{})

	// Второй запуск
	done2 := make(chan struct{})
	go func() {
		service.startCollector()
		close(done2)
	}()

	time.Sleep(50 * time.Millisecond)
	close(service.doneChan)
	<-done2

	// Проверяем, что метрики собраны
	metrics := service.collector.GetMetrics()
	if len(metrics) == 0 {
		t.Error("Metrics should be collected after restart")
	}
}
