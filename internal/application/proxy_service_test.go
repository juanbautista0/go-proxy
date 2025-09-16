package application

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/juanbautista0/go-proxy/internal/domain"
	"github.com/juanbautista0/go-proxy/internal/infrastructure"
)

func TestProxyService_UpdateConfig(t *testing.T) {
	lb := infrastructure.NewEnterpriseBalancer()
	hc := &mockHealthChecker{}
	service := NewProxyService(lb, hc)

	config := &domain.Config{
		Proxy: domain.ProxyConfig{Port: 8080},
		Backends: []domain.Backend{
			{
				Name: "test-backend",
				Servers: []domain.Server{
					{URL: "http://localhost:3001", Weight: 1, Active: true},
					{URL: "http://localhost:3002", Weight: 2, Active: true},
				},
			},
		},
	}

	err := service.UpdateConfig(config)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	// Verify config was updated
	if service.config.Proxy.Port != 8080 {
		t.Errorf("expected port 8080, got %d", service.config.Proxy.Port)
	}

	// Verify balancer was updated
	metrics := lb.GetServerMetrics()
	if len(metrics) != 2 {
		t.Errorf("expected 2 servers in balancer, got %d", len(metrics))
	}
}

func TestProxyService_ServeHTTP_NoBackends(t *testing.T) {
	lb := infrastructure.NewEnterpriseBalancer()
	hc := &mockHealthChecker{}
	service := NewProxyService(lb, hc)

	// No config set
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected status 503, got %d", w.Code)
	}
}

func TestProxyService_ServeHTTP_NoActiveServers(t *testing.T) {
	lb := infrastructure.NewEnterpriseBalancer()
	hc := &mockHealthChecker{}
	service := NewProxyService(lb, hc)

	config := &domain.Config{
		Backends: []domain.Backend{
			{
				Name:    "test-backend",
				Servers: []domain.Server{}, // No servers
			},
		},
	}
	service.UpdateConfig(config)

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected status 503, got %d", w.Code)
	}
}

func TestProxyService_GetMetrics(t *testing.T) {
	lb := infrastructure.NewEnterpriseBalancer()
	hc := &mockHealthChecker{}
	service := NewProxyService(lb, hc)

	// Simulate some requests
	service.requestCount = 100

	metrics := service.GetMetrics()

	if metrics.RequestsPerSecond != 100 {
		t.Errorf("expected RPS 100, got %d", metrics.RequestsPerSecond)
	}

	if metrics.TotalRequests != 100 {
		t.Errorf("expected total requests 100, got %d", metrics.TotalRequests)
	}

	// Should reset counter after getting metrics
	if service.requestCount != 0 {
		t.Errorf("expected request count to be reset, got %d", service.requestCount)
	}
}

func TestProxyService_GetServerStats(t *testing.T) {
	lb := infrastructure.NewEnterpriseBalancer()
	hc := &mockHealthChecker{}
	service := NewProxyService(lb, hc)

	config := &domain.Config{
		Backends: []domain.Backend{
			{
				Name: "test-backend",
				Servers: []domain.Server{
					{URL: "http://localhost:3001", Weight: 1, Active: true},
				},
			},
		},
	}
	service.UpdateConfig(config)

	stats := service.GetServerStats()

	if len(stats) != 1 {
		t.Errorf("expected 1 server stat, got %d", len(stats))
	}

	if _, exists := stats["http://localhost:3001"]; !exists {
		t.Error("expected server stats for http://localhost:3001")
	}
}

func TestProxyService_GetClientIP(t *testing.T) {
	lb := infrastructure.NewEnterpriseBalancer()
	hc := &mockHealthChecker{}
	service := NewProxyService(lb, hc)

	tests := []struct {
		name     string
		headers  map[string]string
		remoteAddr string
		expected string
	}{
		{
			name:     "X-Forwarded-For header",
			headers:  map[string]string{"X-Forwarded-For": "192.168.1.1, 10.0.0.1"},
			expected: "192.168.1.1",
		},
		{
			name:     "X-Real-IP header",
			headers:  map[string]string{"X-Real-IP": "192.168.1.2"},
			expected: "192.168.1.2",
		},
		{
			name:       "RemoteAddr fallback",
			remoteAddr: "192.168.1.3:12345",
			expected:   "192.168.1.3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			
			for key, value := range tt.headers {
				req.Header.Set(key, value)
			}
			
			if tt.remoteAddr != "" {
				req.RemoteAddr = tt.remoteAddr
			}

			ip := service.getClientIP(req)
			if ip != tt.expected {
				t.Errorf("expected IP %s, got %s", tt.expected, ip)
			}
		})
	}
}

func TestProxyService_ShouldRetry(t *testing.T) {
	lb := infrastructure.NewEnterpriseBalancer()
	hc := &mockHealthChecker{}
	service := NewProxyService(lb, hc)

	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "connection refused",
			err:      &mockError{msg: "connection refused"},
			expected: true,
		},
		{
			name:     "timeout",
			err:      &mockError{msg: "timeout"},
			expected: true,
		},
		{
			name:     "no route to host",
			err:      &mockError{msg: "no route to host"},
			expected: true,
		},
		{
			name:     "other error",
			err:      &mockError{msg: "other error"},
			expected: false,
		},
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.shouldRetry(tt.err)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestProxyService_UpdateGlobalMetrics(t *testing.T) {
	lb := infrastructure.NewEnterpriseBalancer()
	hc := &mockHealthChecker{}
	service := NewProxyService(lb, hc)

	// Test successful request
	service.updateGlobalMetrics(100*time.Millisecond, true)
	
	if service.metrics.AverageResponseTime != 100*time.Millisecond {
		t.Errorf("expected avg response time 100ms, got %v", service.metrics.AverageResponseTime)
	}

	// Test failed request
	service.metrics.TotalRequests = 1
	service.updateGlobalMetrics(200*time.Millisecond, false)
	
	if service.metrics.ErrorRate == 0 {
		t.Error("expected error rate to be > 0 after failed request")
	}
}

// Mock implementations
type mockHealthChecker struct{}

func (m *mockHealthChecker) Start(backend *domain.Backend) error { return nil }
func (m *mockHealthChecker) Stop() error { return nil }
func (m *mockHealthChecker) IsHealthy(serverURL string) bool { return true }

type mockError struct {
	msg string
}

func (e *mockError) Error() string {
	return e.msg
}