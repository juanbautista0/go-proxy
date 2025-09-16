package infrastructure

import (
	"context"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/juanbautista0/go-proxy/internal/domain"
)

type AdvancedHealthChecker struct {
	backends map[string]*domain.Backend
	stopChs  map[string]chan struct{}
	client   *http.Client
	mu       sync.RWMutex
}

type HealthCheckResult struct {
	URL           string
	Healthy       bool
	ResponseTime  time.Duration
	StatusCode    int
	Error         error
	Timestamp     time.Time
}

func NewAdvancedHealthChecker() *AdvancedHealthChecker {
	return &AdvancedHealthChecker{
		backends: make(map[string]*domain.Backend),
		stopChs:  make(map[string]chan struct{}),
		client: &http.Client{
			Timeout: 5 * time.Second,
			Transport: &http.Transport{
				DialContext: (&net.Dialer{
					Timeout:   2 * time.Second,
					KeepAlive: 30 * time.Second,
				}).DialContext,
				MaxIdleConns:        100,
				IdleConnTimeout:     90 * time.Second,
				TLSHandshakeTimeout: 2 * time.Second,
			},
		},
	}
}

func (hc *AdvancedHealthChecker) Start(backend *domain.Backend) error {
	hc.mu.Lock()
	defer hc.mu.Unlock()

	backendKey := backend.Name
	hc.backends[backendKey] = backend
	
	if stopCh, exists := hc.stopChs[backendKey]; exists {
		close(stopCh)
	}
	
	stopCh := make(chan struct{})
	hc.stopChs[backendKey] = stopCh

	interval := backend.HealthInterval
	if interval == 0 {
		interval = 10 * time.Second
	}

	go hc.healthCheckLoop(backend, stopCh, interval)
	return nil
}

func (hc *AdvancedHealthChecker) Stop() error {
	hc.mu.Lock()
	defer hc.mu.Unlock()

	for _, stopCh := range hc.stopChs {
		close(stopCh)
	}
	hc.stopChs = make(map[string]chan struct{})
	return nil
}

func (hc *AdvancedHealthChecker) IsHealthy(serverURL string) bool {
	// Esta función ahora es manejada por el IntelligentBalancer
	return true
}

func (hc *AdvancedHealthChecker) healthCheckLoop(backend *domain.Backend, stopCh chan struct{}, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Health check inicial inmediato
	hc.performHealthChecks(backend)

	for {
		select {
		case <-ticker.C:
			hc.performHealthChecks(backend)
		case <-stopCh:
			return
		}
	}
}

func (hc *AdvancedHealthChecker) performHealthChecks(backend *domain.Backend) {
	var wg sync.WaitGroup
	results := make(chan HealthCheckResult, len(backend.Servers))

	for _, server := range backend.Servers {
		if !server.Active {
			continue
		}

		wg.Add(1)
		go func(srv domain.Server) {
			defer wg.Done()
			result := hc.checkServerHealth(srv, backend.HealthCheck)
			results <- result
		}(server)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	// Procesar resultados
	for result := range results {
		hc.processHealthCheckResult(result, backend)
	}
}

func (hc *AdvancedHealthChecker) checkServerHealth(server domain.Server, healthPath string) HealthCheckResult {
	start := time.Now()
	result := HealthCheckResult{
		URL:       server.URL,
		Timestamp: start,
	}

	if healthPath == "" {
		healthPath = "/health"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	url := server.URL + healthPath
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		result.Error = err
		result.Healthy = false
		return result
	}

	// Headers para health check
	req.Header.Set("User-Agent", "go-proxy-health-checker/1.0")
	req.Header.Set("Accept", "*/*")

	resp, err := hc.client.Do(req)
	result.ResponseTime = time.Since(start)

	if err != nil {
		result.Error = err
		result.Healthy = false
		return result
	}
	defer resp.Body.Close()

	result.StatusCode = resp.StatusCode
	
	// Criterios de salud más estrictos
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		// Verificar tiempo de respuesta
		if result.ResponseTime < 5*time.Second {
			result.Healthy = true
		} else {
			result.Healthy = false // Timeout considerado unhealthy
		}
	} else {
		result.Healthy = false
	}

	return result
}

func (hc *AdvancedHealthChecker) processHealthCheckResult(result HealthCheckResult, backend *domain.Backend) {
	hc.mu.Lock()
	defer hc.mu.Unlock()

	// Encontrar el servidor y actualizar su estado
	for i := range backend.Servers {
		server := &backend.Servers[i]
		if server.URL == result.URL {
			server.Healthy = result.Healthy
			server.LastHealthCheck = result.Timestamp
			server.ResponseTime = result.ResponseTime
			
			// Log de cambios de estado
			if !result.Healthy && result.Error != nil {
				// En producción, usar logger apropiado
				// log.Printf("Health check failed for %s: %v", result.URL, result.Error)
			}
			break
		}
	}
}

// Método para obtener métricas de health checks
func (hc *AdvancedHealthChecker) GetHealthMetrics() map[string]interface{} {
	hc.mu.RLock()
	defer hc.mu.RUnlock()

	metrics := make(map[string]interface{})
	
	for backendName, backend := range hc.backends {
		backendMetrics := make(map[string]interface{})
		
		healthyCount := 0
		totalCount := 0
		
		for _, server := range backend.Servers {
			if server.Active {
				totalCount++
				if server.Healthy {
					healthyCount++
				}
			}
		}
		
		backendMetrics["healthy_servers"] = healthyCount
		backendMetrics["total_servers"] = totalCount
		backendMetrics["health_ratio"] = float64(healthyCount) / float64(totalCount)
		
		metrics[backendName] = backendMetrics
	}
	
	return metrics
}