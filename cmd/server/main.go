package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/shatrunoff/yap_metrics/internal/handler"
	"github.com/shatrunoff/yap_metrics/internal/storage"
)

func parseServerFlags() string {
	var serverAddress string

	flag.StringVar(&serverAddress, "a", "localhost:8080", "Server address localhost:8080")
	flag.Parse()

	if flag.NArg() > 0 {
		log.Fatalf("ERROR: unknown arguments: %v", flag.Args())
	}
	return serverAddress
}

func main() {

	serverAddress := parseServerFlags()
	memStorage := storage.NewMemStorage()

	server := &http.Server{
		Addr:    serverAddress,
		Handler: handler.NewHandler(memStorage),
	}

	go func() {
		log.Printf("Server started on %s", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, os.Interrupt, syscall.SIGTERM)
	<-stopChan

	log.Printf("Server stopped on %s", server.Addr)
}
