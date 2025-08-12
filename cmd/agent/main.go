package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/shatrunoff/yap_metrics/internal/config"
	"github.com/shatrunoff/yap_metrics/internal/service"
)

func parseAgentFlags() *config.AgentConfig {
	cfg := config.DefaultAgentConfig()

	flag.StringVar(&cfg.ServerURL, "a", cfg.ServerURL, "Server address localhost:8080")
	flag.DurationVar(&cfg.PollInterval, "p", cfg.PollInterval, "PollInterval=2s")
	flag.DurationVar(&cfg.ReportInterval, "r", cfg.ReportInterval, "ReportInterval=10s")
	flag.Parse()

	if flag.NArg() > 0 {
		log.Fatalf("ERROR: unknown arguments: %v", flag.Args())
	}
	return cfg
}

func main() {
	// инизиализация конфига и агента
	cfg := parseAgentFlags()

	agent := service.NewAgent(cfg)
	defer agent.Stop()

	go agent.Run()
	log.Printf("Metric collector app started with config: %+v", cfg)

	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, os.Interrupt, syscall.SIGTERM)
	<-stopChan

	log.Printf("Metric collector app complete the work")
}
