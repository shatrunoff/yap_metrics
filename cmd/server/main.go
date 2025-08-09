package main

import (
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/shatrunoff/yap_metrics/internal/handler"
)

func main() {
	server := &http.Server{
		Addr:    "localhost:8080",
		Handler: handler.NewHandler(),
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
