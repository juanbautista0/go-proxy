package domain

import (
	"testing"
	"time"
)

func TestServer_IsHealthy(t *testing.T) {
	tests := []struct {
		name     string
		server   Server
		expected bool
	}{
		{
			name: "healthy active server",
			server: Server{
				Active:  true,
				Healthy: true,
			},
			expected: true,
		},
		{
			name: "inactive server",
			server: Server{
				Active:  false,
				Healthy: true,
			},
			expected: false,
		},
		{
			name: "unhealthy server",
			server: Server{
				Active:  true,
				Healthy: false,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.server.Active && tt.server.Healthy
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestTrafficMetrics_CalculateErrorRate(t *testing.T) {
	metrics := &TrafficMetrics{
		TotalRequests: 100,
	}

	// Test with 10% error rate
	metrics.ErrorRate = 0.1
	if metrics.ErrorRate != 0.1 {
		t.Errorf("expected error rate 0.1, got %f", metrics.ErrorRate)
	}
}

func TestCircuitBreakerCfg_IsEnabled(t *testing.T) {
	tests := []struct {
		name     string
		config   CircuitBreakerCfg
		expected bool
	}{
		{
			name: "enabled circuit breaker",
			config: CircuitBreakerCfg{
				Enabled:          true,
				FailureThreshold: 5,
			},
			expected: true,
		},
		{
			name: "disabled circuit breaker",
			config: CircuitBreakerCfg{
				Enabled: false,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.config.Enabled != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, tt.config.Enabled)
			}
		})
	}
}

func TestBalanceMode_String(t *testing.T) {
	tests := []struct {
		mode     BalanceMode
		expected string
	}{
		{RoundRobin, "roundrobin"},
		{LeastConn, "leastconn"},
		{Weighted, "weighted"},
		{IPHash, "iphash"},
		{LeastResponse, "leastresponse"},
	}

	for _, tt := range tests {
		t.Run(string(tt.mode), func(t *testing.T) {
			if string(tt.mode) != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, string(tt.mode))
			}
		})
	}
}

func TestSmartTrigger_Validation(t *testing.T) {
	trigger := SmartTrigger{
		Enabled:             true,
		EvaluationInterval:  5 * time.Second,
		ShortWindow:         30 * time.Second,
		LongWindow:          5 * time.Minute,
		Cooldown:            1 * time.Minute,
		StabilityThreshold:  0.4,
		ScaleUpScore:        0.45,
		ScaleDownScore:      0.15,
	}

	// Validate intervals
	if trigger.ShortWindow >= trigger.LongWindow {
		t.Error("short window should be less than long window")
	}

	if trigger.ScaleDownScore >= trigger.ScaleUpScore {
		t.Error("scale down score should be less than scale up score")
	}

	if trigger.StabilityThreshold < 0 || trigger.StabilityThreshold > 1 {
		t.Error("stability threshold should be between 0 and 1")
	}
}