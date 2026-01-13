package service

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/shatrunoff/yap_metrics/internal/agent"
	"github.com/shatrunoff/yap_metrics/internal/config"
	"github.com/shatrunoff/yap_metrics/internal/model"
)

type AgentService struct {
	collector  *agent.MetricsCollector
	sender     *agent.Sender
	grpcSender *agent.GRPCSender
	config     *config.AgentConfig
	doneChan   chan struct{}
	wg         sync.WaitGroup
	jobs       chan model.Metrics
	workersWG  sync.WaitGroup
	useGRPC    bool
}

// Параметры буферизации отправки метрик по умолчанию
const (
	defaultBatchSize    = 10
	defaultBatchTimeout = 100 * time.Millisecond
)

// calcJobsBufferSize рассчитывает размер буфера очереди задач отправки на основе
// количества воркеров (RateLimit) и периода отчётности (ReportInterval)
func calcJobsBufferSize(cfg *config.AgentConfig) int {
	workers := cfg.RateLimit
	if workers <= 0 {
		workers = 1
	}
	cycles := int(cfg.ReportInterval / defaultBatchTimeout)
	if cycles < 1 {
		cycles = 1
	}
	cap := max(workers*defaultBatchSize*cycles, 64)
	return cap
}

func NewAgent(cfg *config.AgentConfig) (*AgentService, error) {
	as := &AgentService{
		collector: agent.NewMetricsCollector(),
		config:    cfg,
		doneChan:  make(chan struct{}),
		jobs:      make(chan model.Metrics, calcJobsBufferSize(cfg)),
	}

	// Если указан gRPC адрес, используем gRPC
	if cfg.GRPCAddress != "" {
		grpcSender, err := agent.NewGRPCSender(cfg.GRPCAddress)
		if err != nil {
			return nil, fmt.Errorf("failed to create gRPC sender: %w", err)
		}
		as.grpcSender = grpcSender
		as.useGRPC = true
	} else {
		// Иначе используем HTTP
		var sender *agent.Sender
		var err error
		if cfg.CryptoKey != "" {
			sender, err = agent.NewSenderWithCrypto(cfg.ServerURL, cfg.Key, cfg.CryptoKey)
			if err != nil {
				return nil, fmt.Errorf("failed to create sender with crypto: %w", err)
			}
		} else {
			sender = agent.NewSender(cfg.ServerURL, cfg.Key)
		}
		as.sender = sender
	}

	return as, nil
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

// запускает производителей задач отправки; собирает метрики и публикует задачи в пул
func (as *AgentService) startSender() {
	ticker := time.NewTicker(as.config.ReportInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			allMetrics := as.collector.GetMetrics()
			// фиксируем срез для детерминированной публикации и корректного досыла при остановке
			list := make([]model.Metrics, 0, len(allMetrics))
			for _, m := range allMetrics {
				list = append(list, m)
			}
			for i := 0; i < len(list); i++ {
				m := list[i]
				select {
				case as.jobs <- m:
				case <-as.doneChan:
					// Остановка во время публикации — досылаем текущий и оставшиеся
					for ; i < len(list); i++ {
						as.jobs <- list[i]
					}
					close(as.jobs)
					return
				}
			}
		case <-as.doneChan:
			// Перед завершением — отправляем финальный снапшот метрик
			allMetrics := as.collector.GetMetrics()
			for _, m := range allMetrics {
				as.jobs <- m
			}
			// После публикации завершаем работников закрытием очереди
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

			// Локальный буфер для батч-отправки и тикер для таймаута
			batch := make([]model.Metrics, 0, defaultBatchSize)
			ticker := time.NewTicker(defaultBatchTimeout)
			defer ticker.Stop()

			flush := func() {
				if len(batch) == 0 {
					return
				}
				var err error
				if as.useGRPC {
					err = as.grpcSender.SendBatch(batch)
				} else {
					err = as.sender.SendBatchWithRetry(batch)
				}
				if err != nil {
					log.Printf("FAIL to send batch (%d): %v", len(batch), err)
				}
				batch = batch[:0]
			}

			for {
				select {
				case m, ok := <-as.jobs:
					if !ok {
						// очередь закрыта — отправляем остатки и выходим
						flush()
						return
					}
					batch = append(batch, m)
					if len(batch) >= defaultBatchSize {
						flush()
					}
				case <-ticker.C:
					// Периодическая отправка накопленных метрик
					flush()
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
	select {
	case <-as.doneChan:
		// Already closed
		return
	default:
		close(as.doneChan)
	}
	as.wg.Wait()
	// дожидаемся завершения всех воркеров
	as.workersWG.Wait()
	// закрываем gRPC соединение
	if as.grpcSender != nil {
		as.grpcSender.Close()
	}
}

func (as *AgentService) Run() {
	// 3 горутины: сбор runtime, сбор sys, отправитель
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
