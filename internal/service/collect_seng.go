package service

import (
	"log"
	"time"

	"github.com/shatrunoff/yap_metrics/internal/agent"
	"github.com/shatrunoff/yap_metrics/internal/config"
)

type AgentService struct {
	collector *agent.MetricsCollector
	sender    *agent.Sender
	config    *config.AgentConfig
	doneChan  chan struct{}
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
}

func (as *AgentService) Run() {
	// запуск сбора и отправки
	go as.startCollector()
	go as.startSender()

	<-as.doneChan
}
