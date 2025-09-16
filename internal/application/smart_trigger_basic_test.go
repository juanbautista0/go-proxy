package application

import (
	"net/http"
	"testing"
	"time"

	"github.com/juanbautista0/go-proxy/internal/domain"
)

func TestSmartTriggerService_Creation(t *testing.T) {
	executor := &mockActionExecutor{}
	proxyService := &mockProxyService{}
	service := NewSmartTriggerService(executor, proxyService)

	if service == nil {
		t.Error("expected service to be created")
	}
}

func TestSmartTriggerService_BasicMetrics(t *testing.T) {
	executor := &mockActionExecutor{}
	proxyService := &mockProxyService{
		metrics: &domain.TrafficMetrics{
			RequestsPerSecond:   100,
			AverageResponseTime: 200 * time.Millisecond,
			ErrorRate:           0.02,
			ActiveConnections:   50,
		},
	}
	
	service := NewSmartTriggerService(executor, proxyService)
	metrics := service.proxyService.GetMetrics()
	
	if metrics.RequestsPerSecond != 100 {
		t.Errorf("expected RPS 100, got %d", metrics.RequestsPerSecond)
	}
}

// Mock implementations
type mockActionExecutor struct {
	executedActions []string
}

func (m *mockActionExecutor) Execute(actionName string, config domain.ActionConfig) error {
	m.executedActions = append(m.executedActions, actionName)
	return nil
}

type mockProxyService struct {
	metrics *domain.TrafficMetrics
}

func (m *mockProxyService) ServeHTTP(w http.ResponseWriter, r *http.Request) {}
func (m *mockProxyService) UpdateConfig(config *domain.Config) error { return nil }
func (m *mockProxyService) GetMetrics() *domain.TrafficMetrics {
	if m.metrics == nil {
		return &domain.TrafficMetrics{}
	}
	return m.metrics
}
func (m *mockProxyService) GetServerStats() map[string]*domain.Server {
	return make(map[string]*domain.Server)
}