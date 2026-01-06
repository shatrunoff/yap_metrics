package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/shatrunoff/yap_metrics/internal/config"
	"github.com/shatrunoff/yap_metrics/internal/service"
	"github.com/shatrunoff/yap_metrics/internal/utils"
)

func main() {

	// выводим информацию о сборке
	utils.PrintBuildInfo()

	// инициализация конфига и агента
	cfg := config.ParseAgentConfig()

	agent := service.NewAgent(cfg)
	defer agent.Stop()

	go agent.Run()
	log.Printf("Metric collector app started with config: %+v", cfg)

	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, os.Interrupt, syscall.SIGTERM)
	<-stopChan

	log.Printf("Metric collector app complete the work")
}
