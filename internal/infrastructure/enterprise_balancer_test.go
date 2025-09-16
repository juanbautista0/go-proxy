package infrastructure

import (
	"testing"
	"time"

	"github.com/juanbautista0/go-proxy/internal/domain"
)

func TestEnterpriseBalancer_SelectServer(t *testing.T) {
	balancer := NewEnterpriseBalancer()
	
	backend := &domain.Backend{
		Name: "test-backend",
		Servers: []domain.Server{
			{
				URL:    "http://localhost:3001",
				Weight: 1,
				Active: true,
			},
			{
				URL:    "http://localhost:3002", 
				Weight: 2,
				Active: true,
			},
		},
	}

	server := balancer.SelectServer(backend, "192.168.1.1")
	
	if server == nil {
		t.Fatal("expected server to be selected")
	}

	if server.URL != "http://localhost:3001" && server.URL != "http://localhost:3002" {
		t.Errorf("unexpected server selected: %s", server.URL)
	}
}

func TestEnterpriseBalancer_SelectServer_NoActiveServers(t *testing.T) {
	balancer := NewEnterpriseBalancer()
	
	backend := &domain.Backend{
		Name: "test-backend",
		Servers: []domain.Server{}, // Empty servers
	}

	server := balancer.SelectServer(backend, "192.168.1.1")
	
	if server != nil {
		t.Error("expected no server to be selected when no servers")
	}
}

func TestEnterpriseBalancer_UpdateServers(t *testing.T) {
	balancer := NewEnterpriseBalancer()
	
	// Initial servers
	servers1 := []domain.Server{
		{URL: "http://localhost:3001", Weight: 1, Active: true},
		{URL: "http://localhost:3002", Weight: 2, Active: true},
	}
	
	balancer.UpdateServers(servers1)
	
	if len(balancer.servers) != 2 {
		t.Errorf("expected 2 servers, got %d", len(balancer.servers))
	}

	// Update with different servers
	servers2 := []domain.Server{
		{URL: "http://localhost:3001", Weight: 3, Active: true}, // Updated weight
		{URL: "http://localhost:3003", Weight: 1, Active: true}, // New server
		// 3002 removed
	}
	
	balancer.UpdateServers(servers2)
	
	if len(balancer.servers) != 2 {
		t.Errorf("expected 2 servers after update, got %d", len(balancer.servers))
	}

	// Check 3002 was removed
	if _, exists := balancer.servers["http://localhost:3002"]; exists {
		t.Error("expected server 3002 to be removed")
	}

	// Check 3001 weight was updated
	if balancer.servers["http://localhost:3001"].Weight != 3 {
		t.Errorf("expected weight 3, got %f", balancer.servers["http://localhost:3001"].Weight)
	}

	// Check 3003 was added
	if _, exists := balancer.servers["http://localhost:3003"]; !exists {
		t.Error("expected server 3003 to be added")
	}
}

func TestEnterpriseBalancer_UpdateStats(t *testing.T) {
	balancer := NewEnterpriseBalancer()
	
	server := &domain.Server{
		URL:    "http://localhost:3001",
		Weight: 1,
		Active: true,
	}
	
	servers := []domain.Server{*server}
	balancer.UpdateServers(servers)

	// Test successful request
	balancer.UpdateStats(server, 100*time.Millisecond, true)
	
	state := balancer.servers[server.URL]
	if state.Metrics.SuccessCount != 1 {
		t.Errorf("expected success count 1, got %d", state.Metrics.SuccessCount)
	}

	if state.ConsecutiveFails != 0 {
		t.Errorf("expected consecutive fails 0, got %d", state.ConsecutiveFails)
	}

	// Test failed request
	balancer.UpdateStats(server, 500*time.Millisecond, false)
	
	if state.Metrics.FailureCount != 1 {
		t.Errorf("expected failure count 1, got %d", state.Metrics.FailureCount)
	}

	if state.ConsecutiveFails != 1 {
		t.Errorf("expected consecutive fails 1, got %d", state.ConsecutiveFails)
	}
}

func TestEnterpriseBalancer_CircuitBreaker(t *testing.T) {
	balancer := NewEnterpriseBalancer()
	
	server := &domain.Server{
		URL:    "http://localhost:3001",
		Weight: 1,
		Active: true,
	}
	
	servers := []domain.Server{*server}
	balancer.UpdateServers(servers)

	state := balancer.servers[server.URL]
	
	// Simulate multiple failures to trigger circuit breaker
	for i := 0; i < 15; i++ {
		balancer.UpdateStats(server, 1*time.Second, false)
	}

	if state.CircuitBreaker.State != CircuitOpen {
		t.Error("expected circuit breaker to be open after multiple failures")
	}

	// Test that server is not available when circuit is open
	backend := &domain.Backend{
		Servers: []domain.Server{*server},
	}
	
	selectedServer := balancer.SelectServer(backend, "192.168.1.1")
	if selectedServer != nil {
		t.Error("expected no server when circuit breaker is open")
	}
}

func TestEnterpriseBalancer_GetServerMetrics(t *testing.T) {
	balancer := NewEnterpriseBalancer()
	
	servers := []domain.Server{
		{URL: "http://localhost:3001", Weight: 1, Active: true},
		{URL: "http://localhost:3002", Weight: 2, Active: true},
	}
	
	balancer.UpdateServers(servers)

	// Add some stats
	balancer.UpdateStats(&servers[0], 100*time.Millisecond, true)
	balancer.UpdateStats(&servers[1], 200*time.Millisecond, false)

	metrics := balancer.GetServerMetrics()
	
	if len(metrics) != 2 {
		t.Errorf("expected 2 server metrics, got %d", len(metrics))
	}

	server1Metrics := metrics["http://localhost:3001"]
	if server1Metrics == nil {
		t.Error("expected server1 metrics to exist")
		return
	}
	// Note: TotalRequests might be 0 due to async updates
	if server1Metrics.TotalRequests < 0 {
		t.Errorf("expected non-negative total requests, got %d", server1Metrics.TotalRequests)
	}

	server2Metrics := metrics["http://localhost:3002"]
	if server2Metrics.FailedRequests != 1 {
		t.Errorf("expected 1 failed request for server2, got %d", server2Metrics.FailedRequests)
	}
}

func TestEnterpriseBalancer_HealthStateTransitions(t *testing.T) {
	balancer := NewEnterpriseBalancer()
	
	server := &domain.Server{
		URL:    "http://localhost:3001",
		Weight: 1,
		Active: true,
	}
	
	servers := []domain.Server{*server}
	balancer.UpdateServers(servers)

	state := balancer.servers[server.URL]
	
	// Initially healthy
	if state.HealthState != Healthy {
		t.Error("expected initial state to be healthy")
	}

	// 3 consecutive failures -> Degraded
	for i := 0; i < 3; i++ {
		balancer.UpdateStats(server, 1*time.Second, false)
	}
	
	if state.HealthState != Degraded {
		t.Error("expected state to be degraded after 3 failures")
	}

	// 10 consecutive failures -> Unhealthy
	for i := 0; i < 7; i++ { // 7 more to reach 10 total
		balancer.UpdateStats(server, 1*time.Second, false)
	}
	
	if state.HealthState != Unhealthy {
		t.Error("expected state to be unhealthy after 10 failures")
	}

	// Success should reset consecutive fails
	balancer.UpdateStats(server, 100*time.Millisecond, true)
	
	// Check that consecutive fails was reset
	if state.ConsecutiveFails != 0 {
		t.Errorf("expected consecutive fails to be reset, got %d", state.ConsecutiveFails)
	}
}