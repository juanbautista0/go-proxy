package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/juanbautista0/go-proxy/internal/application"
	"github.com/juanbautista0/go-proxy/internal/domain"
	"github.com/juanbautista0/go-proxy/internal/infrastructure"
)

func main() {
	configPath := "config.yaml"
	if len(os.Args) > 1 {
		configPath = os.Args[1]
	}

	// Verificar si el archivo existe
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		log.Fatalf("Config file not found: %s", configPath)
	}

	// Infraestructura
	configManager := infrastructure.NewConfigManager(configPath)
	actionExecutor := infrastructure.NewHTTPActionExecutor()
	loadBalancer := infrastructure.NewEnterpriseBalancer()
	healthChecker := infrastructure.NewHealthChecker()

	// Cargar configuración inicial
	config, err := configManager.Load()
	if err != nil {
		log.Fatal("Error loading config:", err)
	}

	// Aplicación
	proxyService := application.NewProxyService(loadBalancer, healthChecker)
	
	// Sistema de triggers inteligente o legacy
	var triggerService domain.TriggerService
	if config.Triggers.Smart.Enabled {
		smartTrigger := application.NewSmartTriggerService(actionExecutor, proxyService)
		triggerService = application.NewHybridTriggerService(smartTrigger, actionExecutor)
		log.Println("🧠 Smart Trigger System enabled")
	} else {
		triggerService = application.NewTriggerService(actionExecutor)
		log.Println("⚠️  Legacy Trigger System enabled")
	}

	proxyService.UpdateConfig(config)
	triggerService.Start(config, proxyService.GetMetrics())

	// Iniciar health checks
	for _, backend := range config.Backends {
		healthChecker.Start(&backend)
	}

	// Callback para cambios de configuración
	configManager.AddCallback(func(newConfig *domain.Config) {
		log.Println("Config updated, reloading...")
		proxyService.UpdateConfig(newConfig)
		triggerService.Stop()
		triggerService.Start(newConfig, proxyService.GetMetrics())
	})

	// Servidor de métricas
	metricsServer := infrastructure.NewMetricsServer(proxyService)
	go func() {
		log.Println("Metrics server starting on :8081")
		if err := metricsServer.Start(8081); err != nil {
			log.Printf("Metrics server error: %v", err)
		}
	}()

	// API de configuración
	configAPI := infrastructure.NewConfigAPI(configManager)
	go func() {
		log.Println("Config API starting on :8082")
		http.ListenAndServe(":8082", configAPI)
	}()

	// Servidor HTTP
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", config.Proxy.Port),
		Handler: proxyService,
	}

	// Métricas en goroutine separada
	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for range ticker.C {
			proxyService.GetMetrics()
		}
	}()

	// Graceful shutdown
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh

		log.Println("Shutting down...")
		triggerService.Stop()
		server.Close()
	}()

	log.Printf("Proxy server starting on port %d", config.Proxy.Port)
	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatal("Server error:", err)
	}
}
