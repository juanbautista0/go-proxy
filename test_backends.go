package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

func main() {
	ports := []int{3001, 3002, 3003}
	var wg sync.WaitGroup

	// Iniciar servidores en goroutines
	for _, port := range ports {
		wg.Add(1)
		go func(p int) {
			defer wg.Done()
			startServer(p)
		}(port)
	}

	log.Println("🚀 Test backends started on ports 3001, 3002, 3003")

	// Graceful shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	log.Println("🛑 Shutting down test backends...")
}

func startServer(port int) {
	mux := http.NewServeMux()
	
	// Health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status":"healthy","port":%d,"timestamp":"%s"}`, port, time.Now().Format(time.RFC3339))
	})

	// Root endpoint
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"message":"Hello from backend %d","port":%d,"path":"%s","method":"%s","timestamp":"%s"}`, 
			port, port, r.URL.Path, r.Method, time.Now().Format(time.RFC3339))
	})

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: mux,
	}

	log.Printf("Backend server starting on port %d", port)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Printf("Backend server %d error: %v", port, err)
	}
}