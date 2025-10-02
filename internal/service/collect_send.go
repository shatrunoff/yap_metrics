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
	jobs      chan model.Metrics
	workersWG sync.WaitGroup
}

func NewAgent(cfg *config.AgentConfig) *AgentService {
	return &AgentService{
		collector: agent.NewMetricsCollector(),
		sender:    agent.NewSender(cfg.ServerURL, cfg.Key),
		config:    cfg,
		doneChan:  make(chan struct{}),
		jobs:      make(chan model.Metrics, 256),
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

// запускает задачи отправки и собирает метрики, публикует задачи в пул
func (as *AgentService) startSender() {
	ticker := time.NewTicker(as.config.ReportInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			allMetrics := as.collector.GetMetrics()
			for _, m := range allMetrics {
				select {
				case as.jobs <- m:
				case <-as.doneChan:
					close(as.jobs)
					return
				}
			}
		case <-as.doneChan:
			// Закрываем очередь задач, чтобы завершить jobs
			close(as.jobs)
			return
		}
	}
}

// запуск воркеров с ограничением одновременно исходящих запросов
func (as *AgentService) startWorkers() {
	workers := as.config.RateLimit
	if workers <= 0 {
		workers = 1
	}
	as.workersWG.Add(workers)
	for i := 0; i < workers; i++ {
		go func() {
			defer as.workersWG.Done()
			for m := range as.jobs {
				// отправляем по одной метрике, используя батч-метод
				if err := as.sender.SendBatchWithRetry([]model.Metrics{m}); err != nil {
					log.Printf("FAIL to send metric %s: %v", m.ID, err)
				}
			}
		}()
	}
}

// сбор системных метрик
func (as *AgentService) startSysCollector() {
	ticker := time.NewTicker(as.config.PollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			as.collector.CollectSys()
		case <-as.doneChan:
			return
		}
	}
}

func (as *AgentService) Stop() {
	close(as.doneChan)
	as.wg.Wait()
	// дожидаемся завершения всех воркеров
	as.workersWG.Wait()
}

func (as *AgentService) Run() {
	// 3 горутины:
	// сбор runtime,
	// сбор sys,
	// отправитель
	as.wg.Add(3)

	// запуск воркеров отправки по лимиту
	as.startWorkers()

	// запуск сбора
	go func() {
		defer as.wg.Done()
		as.startCollector()
	}()

	// запуск сбора системных метрик
	go func() {
		defer as.wg.Done()
		as.startSysCollector()
	}()

	// запуск отправки
	go func() {
		defer as.wg.Done()
		as.startSender()
	}()

	<-as.doneChan
}
