package infrastructure

import (
	"net/http"
	"sync"
	"time"

	"github.com/juanbautista0/go-proxy/internal/domain"
)

type HealthCheckerImpl struct {
	backend *domain.Backend
	stopCh  chan struct{}
	client  *http.Client
	mu      sync.RWMutex
}

func NewHealthChecker() *HealthCheckerImpl {
	return &HealthCheckerImpl{
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

func (hc *HealthCheckerImpl) Start(backend *domain.Backend) error {
	hc.mu.Lock()
	defer hc.mu.Unlock()
	
	hc.backend = backend
	hc.stopCh = make(chan struct{})
	
	interval := backend.HealthInterval
	if interval == 0 {
		interval = 10 * time.Second
	}
	
	go hc.healthCheckLoop(interval)
	return nil
}

func (hc *HealthCheckerImpl) Stop() error {
	hc.mu.Lock()
	defer hc.mu.Unlock()
	
	if hc.stopCh != nil {
		close(hc.stopCh)
		hc.stopCh = nil
	}
	return nil
}

func (hc *HealthCheckerImpl) IsHealthy(serverURL string) bool {
	hc.mu.RLock()
	defer hc.mu.RUnlock()
	
	if hc.backend == nil {
		return false
	}
	
	for _, server := range hc.backend.Servers {
		if server.URL == serverURL {
			return server.Healthy
		}
	}
	return false
}

func (hc *HealthCheckerImpl) healthCheckLoop(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			hc.checkAllServers()
		case <-hc.stopCh:
			return
		}
	}
}

func (hc *HealthCheckerImpl) checkAllServers() {
	hc.mu.Lock()
	defer hc.mu.Unlock()
	
	if hc.backend == nil {
		return
	}
	
	for i := range hc.backend.Servers {
		server := &hc.backend.Servers[i]
		if server.Active {
			healthy := hc.checkServer(server)
			server.Healthy = healthy
			server.LastHealthCheck = time.Now()
		}
	}
}

func (hc *HealthCheckerImpl) checkServer(server *domain.Server) bool {
	// Usar endpoint individual del servidor o fallback al del backend
	healthEndpoint := server.HealthCheckEndpoint
	if healthEndpoint == "" {
		healthEndpoint = hc.backend.HealthCheck
	}
	
	if healthEndpoint == "" {
		return true // Sin health check configurado
	}
	
	url := server.URL + healthEndpoint
	resp, err := hc.client.Get(url)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	
	return resp.StatusCode >= 200 && resp.StatusCode < 300
}