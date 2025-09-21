package service

import (
	"log"
	"sync"
	"time"

	"github.com/shatrunoff/yap_metrics/internal/agent"
	"github.com/shatrunoff/yap_metrics/internal/config"
	"github.com/shatrunoff/yap_metrics/internal/model"
)

type AgentService struct {
	collector *agent.MetricsCollector
	sender    *agent.Sender
	config    *config.AgentConfig
	doneChan  chan struct{}
	wg        sync.WaitGroup
}

func NewAgent(cfg *config.AgentConfig) *AgentService {
	return &AgentService{
		collector: agent.NewMetricsCollector(),
		sender:    agent.NewSender(cfg.ServerURL),
		config:    cfg,
		doneChan:  make(chan struct{}),
	}
}

// собирает метрики
func (as *AgentService) startCollector() {
	ticker := time.NewTicker(as.config.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			as.collector.Collect()
		case <-as.doneChan:
			return
		}
	}
}

// отправка метрик через JSON
func (as *AgentService) startSender() {
	ticker := time.NewTicker(as.config.ReportInterval)
	defer ticker.Stop()

	// Буфер для накопления метрик
	var metricsBuffer []model.Metrics
	bufferSize := 10

	for {
		select {
		case <-ticker.C:
			// все метрики
			allMetrics := as.collector.GetMetrics()

			// map в slice
			metricsSlice := make([]model.Metrics, 0, len(allMetrics))
			for _, metric := range allMetrics {
				metricsSlice = append(metricsSlice, metric)
			}

			// Добавляем в буфер
			metricsBuffer = append(metricsBuffer, metricsSlice...)

			// Если буфер достиг размера батча или больше, то отправляем
			if len(metricsBuffer) >= bufferSize {
				if err := as.sender.SendBatch(metricsBuffer); err != nil {
					log.Printf("FAIL to send metrics batch: %v", err)
				} else {
					log.Printf("Successfully sent batch of %d metrics", len(metricsBuffer))
				}
				// Очищаем буфер после отправки
				metricsBuffer = nil
			}

		case <-as.doneChan:
			// При остановке отправляем оставшиеся метрики
			if len(metricsBuffer) > 0 {
				if err := as.sender.SendBatch(metricsBuffer); err != nil {
					log.Printf("FAIL to send final metrics batch: %v", err)
				} else {
					log.Printf("Successfully sent final batch of %d metrics", len(metricsBuffer))
				}
			}
			return
		}
	}
}

func (as *AgentService) Stop() {
	close(as.doneChan)
	as.wg.Wait()
}

func (as *AgentService) Run() {
	// 2 горутины
	as.wg.Add(2)

	// запуск сбора
	go func() {
		defer as.wg.Done()
		as.startCollector()
	}()

	// запуск отправки
	go func() {
		defer as.wg.Done()
		as.startSender()
	}()

	<-as.doneChan
}
