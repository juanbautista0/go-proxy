// +build integration

package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/juanbautista0/go-proxy/internal/application"
	"github.com/juanbautista0/go-proxy/internal/domain"
	"github.com/juanbautista0/go-proxy/internal/infrastructure"
)

func TestIntegration_FullWorkflow(t *testing.T) {
	// Create temp config file
	tempFile, err := os.CreateTemp("", "integration_test_*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tempFile.Name())

	configContent := `
proxy:
  port: 8080
backends:
  - name: "web-servers"
    servers:
      - url: "http://localhost:3001"
        weight: 1
        max_connections: 100
        health_check_endpoint: "/health"
    health_check: "/health"
    balance_mode: "adaptive_weighted"
triggers:
  smart:
    enabled: true
    evaluation_interval: "1s"
    scale_up_score: 0.7
    scale_down_score: 0.3
actions:
  scale_up:
    url: "http://localhost:8082/actions/scale_up"
    method: "POST"
  scale_down:
    url: "http://localhost:8082/actions/scale_down"
    method: "POST"
`
	tempFile.WriteString(configContent)
	tempFile.Close()

	// Setup components
	configManager := infrastructure.NewConfigManager(tempFile.Name())
	config, err := configManager.Load()
	if err != nil {
		t.Fatal(err)
	}

	loadBalancer := infrastructure.NewEnterpriseBalancer()
	healthChecker := infrastructure.NewHealthChecker()
	actionExecutor := infrastructure.NewHTTPActionExecutor()
	
	proxyService := application.NewProxyService(loadBalancer, healthChecker)
	proxyService.UpdateConfig(config)

	configAPI := infrastructure.NewConfigAPI(configManager)

	// Test 1: Add server via API
	t.Run("AddServer", func(t *testing.T) {
		addReq := infrastructure.AddServerRequest{
			BackendName:         "web-servers",
			URL:                 "http://localhost:3002",
			Weight:              2,
			MaxConnections:      150,
			HealthCheckEndpoint: "/health",
		}

		body, _ := json.Marshal(addReq)
		req := httptest.NewRequest("POST", "/servers", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		configAPI.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Errorf("expected status 201, got %d", w.Code)
		}

		// Verify server was added to balancer
		metrics := loadBalancer.GetServerMetrics()
		if len(metrics) != 2 {
			t.Errorf("expected 2 servers in balancer, got %d", len(metrics))
		}
	})

	// Test 2: Scale up action
	t.Run("ScaleUpAction", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/actions/scale_up", nil)
		w := httptest.NewRecorder()

		configAPI.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}

		// Verify server was added
		updatedConfig := configManager.GetConfig()
		if len(updatedConfig.Backends[0].Servers) != 3 {
			t.Errorf("expected 3 servers after scale up, got %d", len(updatedConfig.Backends[0].Servers))
		}
	})

	// Test 3: Remove server
	t.Run("RemoveServer", func(t *testing.T) {
		removeReq := infrastructure.RemoveServerRequest{
			BackendName: "web-servers",
			ServerURL:   "http://localhost:3004", // Added by scale up
		}

		body, _ := json.Marshal(removeReq)
		req := httptest.NewRequest("DELETE", "/servers", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		configAPI.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}

		// Verify server was removed from balancer
		metrics := loadBalancer.GetServerMetrics()
		if len(metrics) != 2 {
			t.Errorf("expected 2 servers after removal, got %d", len(metrics))
		}
	})

	// Test 4: Proxy request handling
	t.Run("ProxyRequest", func(t *testing.T) {
		// Create mock backend server
		backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Hello from backend"))
		}))
		defer backend.Close()

		// Update config to use mock backend
		config := configManager.GetConfig()
		config.Backends[0].Servers[0].URL = backend.URL
		configManager.Update(config)

		// Make request through proxy
		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()

		proxyService.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}

		if w.Body.String() != "Hello from backend" {
			t.Errorf("unexpected response body: %s", w.Body.String())
		}
	})

	// Test 5: Metrics collection
	t.Run("MetricsCollection", func(t *testing.T) {
		metrics := proxyService.GetMetrics()
		
		if metrics == nil {
			t.Error("expected metrics to be available")
		}

		serverStats := proxyService.GetServerStats()
		if len(serverStats) == 0 {
			t.Error("expected server stats to be available")
		}
	})
}

func TestIntegration_SmartTriggerWorkflow(t *testing.T) {
	// Setup mock action executor that tracks executions
	actionExecutor := &mockIntegrationActionExecutor{
		executions: make([]string, 0),
	}

	// Setup proxy service with high load metrics
	proxyService := &mockIntegrationProxyService{
		metrics: &domain.TrafficMetrics{
			RequestsPerSecond:   500, // High load
			AverageResponseTime: 800 * time.Millisecond,
			ErrorRate:           0.1,
			ActiveConnections:   200,
		},
	}

	smartTrigger := application.NewSmartTriggerService(actionExecutor, proxyService)

	config := &domain.Config{
		Triggers: domain.TriggerConfig{
			Smart: domain.SmartTrigger{
				Enabled:            true,
				EvaluationInterval: 100 * time.Millisecond,
				ScaleUpScore:       0.5,
				ScaleDownScore:     0.2,
				Cooldown:           50 * time.Millisecond,
			},
		},
		Actions: map[string]domain.ActionConfig{
			"scale_up": {
				URL:    "http://localhost:8082/actions/scale_up",
				Method: "POST",
			},
		},
	}

	// Start smart trigger
	smartTrigger.Start(config, proxyService.GetMetrics())

	// Wait for evaluation
	time.Sleep(200 * time.Millisecond)

	smartTrigger.Stop()

	// Verify scale up was triggered
	if len(actionExecutor.executions) == 0 {
		t.Error("expected at least one action execution")
	}

	if actionExecutor.executions[0] != "scale_up" {
		t.Errorf("expected scale_up action, got %s", actionExecutor.executions[0])
	}
}

// Mock implementations for integration tests
type mockIntegrationActionExecutor struct {
	executions []string
}

func (m *mockIntegrationActionExecutor) Execute(action string) error {
	m.executions = append(m.executions, action)
	return nil
}

func (m *mockIntegrationActionExecutor) UpdateActions(actions map[string]infrastructure.ActionConfig) {}

type mockIntegrationProxyService struct {
	metrics *domain.TrafficMetrics
}

func (m *mockIntegrationProxyService) GetMetrics() *domain.TrafficMetrics {
	return m.metrics
}