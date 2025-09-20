package infrastructure

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/juanbautista0/go-proxy/internal/domain"
)

type WebSocketMetrics struct {
	proxyService domain.ProxyService
	loadBalancer *EnterpriseBalancer
}

type MetricsData struct {
	Timestamp time.Time `json:"timestamp"`
	Metrics   struct {
		RequestsPerSecond   int     `json:"requests_per_second"`
		TotalRequests       int64   `json:"total_requests"`
		ActiveConnections   int64   `json:"active_connections"`
		SuccessfulRequests  int64   `json:"successful_requests"`
		FailedRequests      int64   `json:"failed_requests"`
		AverageResponseTime string  `json:"average_response_time"`
		ErrorRate           float64 `json:"error_rate"`
	} `json:"metrics"`
	Servers  map[string]ServerStatus `json:"servers"`
	Draining []string                `json:"draining_servers"`
}

type ServerStatus struct {
	Status         string `json:"status"`
	Connections    int64  `json:"connections"`
	TotalRequests  int64  `json:"total_requests"`
	FailedRequests int64  `json:"failed_requests"`
	ResponseTime   string `json:"response_time"`
	Weight         int    `json:"weight"`
	Active         bool   `json:"active"`
	Draining       bool   `json:"draining"`
}

func NewWebSocketMetrics(proxyService domain.ProxyService) *WebSocketMetrics {
	return &WebSocketMetrics{
		proxyService: proxyService,
	}
}

func (ws *WebSocketMetrics) SetLoadBalancer(lb *EnterpriseBalancer) {
	ws.loadBalancer = lb
}

func (ws *WebSocketMetrics) collectMetrics() MetricsData {
	metrics := ws.proxyService.GetMetrics()
	serverStats := ws.proxyService.GetServerStats()

	var totalRequests, activeConnections, successfulRequests, failedRequests int64
	for _, server := range serverStats {
		totalRequests += server.TotalRequests
		activeConnections += server.CurrentConns
		successfulRequests += server.TotalRequests - server.FailedRequests
		failedRequests += server.FailedRequests
	}

	errorRate := 0.0
	if totalRequests > 0 {
		errorRate = float64(failedRequests) / float64(totalRequests) * 100
	}

	data := MetricsData{
		Timestamp: time.Now(),
		Servers:   make(map[string]ServerStatus),
	}

	data.Metrics.RequestsPerSecond = metrics.RequestsPerSecond
	data.Metrics.TotalRequests = totalRequests
	data.Metrics.ActiveConnections = activeConnections
	data.Metrics.SuccessfulRequests = successfulRequests
	data.Metrics.FailedRequests = failedRequests
	data.Metrics.AverageResponseTime = metrics.AverageResponseTime.String()
	data.Metrics.ErrorRate = errorRate

	// Obtener servidores drenando
	if ws.loadBalancer != nil {
		data.Draining = ws.loadBalancer.GetDrainingServers()
	}

	// Formatear estadísticas de servidores
	for url, server := range serverStats {
		status := "healthy"
		if !server.Healthy {
			status = "unhealthy"
		}
		if server.CircuitOpen {
			status = "circuit_open"
		}

		draining := false
		for _, drainingURL := range data.Draining {
			if drainingURL == url {
				draining = true
				status = "draining"
				break
			}
		}

		data.Servers[url] = ServerStatus{
			Status:         status,
			Connections:    server.CurrentConns,
			TotalRequests:  server.TotalRequests,
			FailedRequests: server.FailedRequests,
			ResponseTime:   server.ResponseTime.String(),
			Weight:         server.Weight,
			Active:         server.Active,
			Draining:       draining,
		}
	}

	return data
}

func (ws *WebSocketMetrics) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			data := ws.collectMetrics()
			if jsonData, err := json.Marshal(data); err == nil {
				w.Write([]byte("data: "))
				w.Write(jsonData)
				w.Write([]byte("\n\n"))
				flusher.Flush()
			}
		case <-r.Context().Done():
			return
		}
	}
}
