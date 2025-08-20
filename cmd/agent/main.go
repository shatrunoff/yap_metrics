package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/shatrunoff/yap_metrics/internal/config"
	"github.com/shatrunoff/yap_metrics/internal/service"
)

func parseAgentConfig() *config.AgentConfig {
	var pollSec int
	var repSec int

	// получаем конфиг по умолчанию
	cfg := config.DefaultAgentConfig()

	// парсим аргументы командной строки
	flag.StringVar(&cfg.ServerURL, "a", cfg.ServerURL, "Server address host:port")
	flag.IntVar(&pollSec, "p", int(cfg.PollInterval.Seconds()), "PollInterval (s)")
	flag.IntVar(&repSec, "r", int(cfg.ReportInterval.Seconds()), "ReportInterval (s)")
	flag.Parse()

	cfg.PollInterval = time.Duration(pollSec) * time.Second
	cfg.ReportInterval = time.Duration(repSec) * time.Second

	if flag.NArg() > 0 {
		log.Fatalf("ERROR: unknown arguments: %v", flag.Args())
	}

	// парсим переменные окружения
	// ADDRESS
	if envAddr := os.Getenv("ADDRESS"); envAddr != "" {
		cfg.ServerURL = envAddr
	}
	// REPORT_INTERVAL
	if envReportInterval := os.Getenv("REPORT_INTERVAL"); envReportInterval != "" {
		if sec, err := strconv.Atoi(envReportInterval); err == nil {
			cfg.ReportInterval = time.Duration(sec) * time.Second
		}
	}
	// POLL_INTERVAL
	if envPoolInterval := os.Getenv("POLL_INTERVAL"); envPoolInterval != "" {
		if sec, err := strconv.Atoi(envPoolInterval); err == nil {
			cfg.PollInterval = time.Duration(sec) * time.Second
		}
	}

	return cfg
}

func main() {
	// инициализация конфига и агента
	cfg := parseAgentConfig()

	agent := service.NewAgent(cfg)
	defer agent.Stop()

	go agent.Run()
	log.Printf("Metric collector app started with config: %+v", cfg)

	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, os.Interrupt, syscall.SIGTERM)
	<-stopChan

	log.Printf("Metric collector app complete the work")
}
