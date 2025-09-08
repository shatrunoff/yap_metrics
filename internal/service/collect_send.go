package service

import (
	"log"
	"sync"
	"time"

	"github.com/shatrunoff/yap_metrics/internal/agent"
	"github.com/shatrunoff/yap_metrics/internal/config"
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
		sender:    agent.NewSender(cfg.ServerURL, true),
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

// отправка метрик
func (as *AgentService) startSender() {
	ticker := time.NewTicker(as.config.ReportInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			metrics := as.collector.GetMetrics()
			if err := as.sender.Send(metrics); err != nil {
				log.Printf("FAIL to send metrics: %v", err)
			}
		case <-as.doneChan:
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
