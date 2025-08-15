package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/shatrunoff/yap_metrics/internal/config"
	"github.com/shatrunoff/yap_metrics/internal/service"
)

func parseAgentFlags() *config.AgentConfig {
	var pollSec int
	var repSec int

	cfg := config.DefaultAgentConfig()

	flag.StringVar(&cfg.ServerURL, "a", cfg.ServerURL, "Server address host:port")
	flag.IntVar(&pollSec, "p", int(cfg.PollInterval.Seconds()), "PollInterval (s)")
	flag.IntVar(&repSec, "r", int(cfg.ReportInterval.Seconds()), "ReportInterval (s)")
	flag.Parse()

	cfg.PollInterval = time.Duration(pollSec) * time.Second
	cfg.ReportInterval = time.Duration(repSec) * time.Second

	if flag.NArg() > 0 {
		log.Fatalf("ERROR: unknown arguments: %v", flag.Args())
	}
	return cfg
}

func main() {
	// инициализация конфига и агента
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
