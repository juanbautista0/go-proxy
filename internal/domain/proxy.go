package domain

import (
	"net/http"
	"time"
)

type ProxyService interface {
	ServeHTTP(w http.ResponseWriter, r *http.Request)
	UpdateConfig(config *Config) error
	GetMetrics() *TrafficMetrics
	GetServerStats() map[string]*Server
}

type ConfigRepository interface {
	Load() (*Config, error)
	Watch(callback func(*Config)) error
}

type TriggerService interface {
	Start(config *Config, metrics *TrafficMetrics) error
	Stop() error
}

type ActionExecutor interface {
	Execute(actionName string, config ActionConfig) error
}

type HealthChecker interface {
	Start(backend *Backend) error
	Stop() error
	IsHealthy(serverURL string) bool
}

type LoadBalancer interface {
	SelectServer(backend *Backend, clientIP string) *Server
	UpdateStats(server *Server, responseTime time.Duration, success bool)
	GetServerMetrics() map[string]*Server
}
