package infrastructure

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/juanbautista0/go-proxy/internal/domain"
)

type MetricsServer struct {
	proxyService domain.ProxyService
}

func NewMetricsServer(proxyService domain.ProxyService) *MetricsServer {
	return &MetricsServer{
		proxyService: proxyService,
	}
}

func (ms *MetricsServer) Start(port int) error {
	http.HandleFunc("/metrics", ms.handleMetrics)
	http.HandleFunc("/", ms.handleDashboard)

	addr := fmt.Sprintf(":%d", port)
	return http.ListenAndServe(addr, nil)
}

func (ms *MetricsServer) handleMetrics(w http.ResponseWriter, r *http.Request) {
	metrics := ms.proxyService.GetMetrics()
	serverStats := ms.proxyService.GetServerStats()

	totalRequests := int64(0)
	activeConnections := int64(0)
	successfulRequests := int64(0)
	failedRequests := int64(0)

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

	response := map[string]interface{}{
		"timestamp": time.Now(),
		"metrics": map[string]interface{}{
			"requests_per_second":   metrics.RequestsPerSecond,
			"total_requests":        totalRequests,
			"active_connections":    activeConnections,
			"successful_requests":   successfulRequests,
			"failed_requests":       failedRequests,
			"average_response_time": metrics.AverageResponseTime.String(),
			"error_rate":            errorRate,
		},
		"servers": ms.formatServerStats(serverStats),
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Cache-Control", "no-cache")
	json.NewEncoder(w).Encode(response)
}

func (ms *MetricsServer) formatServerStats(serverStats map[string]*domain.Server) map[string]interface{} {
	formatted := make(map[string]interface{})

	for url, server := range serverStats {
		status := "healthy"
		if !server.Healthy {
			status = "unhealthy"
		}
		if server.CircuitOpen {
			status = "circuit_open"
		}

		formatted[url] = map[string]interface{}{
			"status":          status,
			"connections":     server.CurrentConns,
			"total_requests":  server.TotalRequests,
			"failed_requests": server.FailedRequests,
			"response_time":   server.ResponseTime.String(),
			"weight":          server.Weight,
			"active":          server.Active,
		}
	}

	return formatted
}

func (ms *MetricsServer) handleDashboard(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprint(w, `<!DOCTYPE html>
<html>
<head>
    <title>Go-Proxy Dashboard</title>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body { font-family: Arial, sans-serif; background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); min-height: 100vh; color: #333; }
        .container { max-width: 1200px; margin: 0 auto; padding: 20px; }
        .header { text-align: center; color: white; margin-bottom: 30px; }
        .header h1 { font-size: 2.5em; margin-bottom: 10px; }
        .status-bar { background: rgba(255,255,255,0.1); padding: 15px; border-radius: 10px; margin-bottom: 20px; text-align: center; color: white; }
        .grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(300px, 1fr)); gap: 20px; margin-bottom: 20px; }
        .card { background: white; border-radius: 15px; padding: 25px; box-shadow: 0 8px 32px rgba(0,0,0,0.1); }
        .card h3 { color: #4a5568; margin-bottom: 20px; font-size: 1.3em; }
        .metric { display: flex; justify-content: space-between; align-items: center; margin: 15px 0; padding: 10px; background: #f7fafc; border-radius: 8px; }
        .metric-label { font-weight: 500; color: #4a5568; }
        .metric-value { font-weight: bold; font-size: 1.2em; color: #2d3748; }
        .metric-value.success { color: #38a169; }
        .metric-value.warning { color: #d69e2e; }
        .metric-value.error { color: #e53e3e; }
        .server { margin: 15px 0; padding: 15px; border-radius: 10px; border-left: 5px solid; }
        .healthy { background: #e6fffa; border-color: #38a169; }
        .unhealthy { background: #fed7d7; border-color: #e53e3e; }
        .circuit_open { background: #fefcbf; border-color: #d69e2e; }
        .server-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 10px; }
        .server-url { font-weight: bold; font-size: 1.1em; }
        .server-status { padding: 4px 12px; border-radius: 20px; font-size: 0.8em; font-weight: bold; text-transform: uppercase; }
        .status-healthy { background: #38a169; color: white; }
        .status-unhealthy { background: #e53e3e; color: white; }
        .status-circuit_open { background: #d69e2e; color: white; }
        .server-stats { display: grid; grid-template-columns: repeat(auto-fit, minmax(120px, 1fr)); gap: 10px; font-size: 0.9em; }
        .stat { text-align: center; padding: 8px; background: rgba(255,255,255,0.7); border-radius: 6px; }
        .stat-label { display: block; font-size: 0.8em; color: #666; margin-bottom: 4px; }
        .stat-value { font-weight: bold; color: #2d3748; }
        .live-indicator { display: inline-block; width: 10px; height: 10px; background: #38a169; border-radius: 50%; margin-right: 8px; animation: pulse 1s infinite; }
        @keyframes pulse { 0%, 100% { opacity: 1; } 50% { opacity: 0.7; } }
        .footer { text-align: center; color: rgba(255,255,255,0.8); margin-top: 30px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>🚀 Go-Proxy Dashboard</h1>
            <p>Intelligent Load Balancer</p>
        </div>
        
        <div class="status-bar">
            <span class="live-indicator"></span>
            <strong>LIVE</strong> | Last Update: <span id="lastUpdate">-</span> | 
            Algorithm: Adaptive Weighted | 
            Uptime: <span id="uptime">-</span>
        </div>

        <div class="grid">
            <div class="card">
                <h3>📊 Traffic Metrics</h3>
                <div class="metric">
                    <span class="metric-label">Requests/Second</span>
                    <span class="metric-value success" id="rps">0</span>
                </div>
                <div class="metric">
                    <span class="metric-label">Total Requests</span>
                    <span class="metric-value" id="total">0</span>
                </div>
                <div class="metric">
                    <span class="metric-label">Active Connections</span>
                    <span class="metric-value" id="active">0</span>
                </div>
                <div class="metric">
                    <span class="metric-label">Success Rate</span>
                    <span class="metric-value success" id="success">100%</span>
                </div>
            </div>

            <div class="card">
                <h3>⚡ Performance</h3>
                <div class="metric">
                    <span class="metric-label">Avg Response Time</span>
                    <span class="metric-value" id="response">0ms</span>
                </div>
                <div class="metric">
                    <span class="metric-label">Error Rate</span>
                    <span class="metric-value" id="error">0%</span>
                </div>
                <div class="metric">
                    <span class="metric-label">Circuit Breakers</span>
                    <span class="metric-value" id="circuits">0 Open</span>
                </div>
                <div class="metric">
                    <span class="metric-label">Load Balance</span>
                    <span class="metric-value success">Optimal</span>
                </div>
            </div>
        </div>

        <div class="card">
            <h3>🖥️ Backend Servers</h3>
            <div id="servers">Loading server stats...</div>
        </div>
        
        <div class="footer">
            <p>🔄 Auto-refreshing every second</p>
        </div>
    </div>

    <script>
        let startTime = Date.now();
        
        function formatNumber(num) {
            return new Intl.NumberFormat().format(num);
        }
        
        function formatUptime(ms) {
            const seconds = Math.floor(ms / 1000);
            const minutes = Math.floor(seconds / 60);
            const hours = Math.floor(minutes / 60);
            return hours > 0 ? hours + 'h ' + (minutes % 60) + 'm' : minutes + 'm ' + (seconds % 60) + 's';
        }
        
        function updateStats() {
            fetch('/metrics')
                .then(r => r.json())
                .then(data => {
                    document.getElementById('rps').textContent = data.metrics.requests_per_second || 0;
                    document.getElementById('total').textContent = formatNumber(data.metrics.total_requests || 0);
                    document.getElementById('active').textContent = data.metrics.active_connections || 0;
                    
                    const errorRate = data.metrics.error_rate || 0;
                    const errorEl = document.getElementById('error');
                    errorEl.textContent = errorRate.toFixed(2) + '%';
                    errorEl.className = 'metric-value ' + (errorRate > 5 ? 'error' : errorRate > 1 ? 'warning' : 'success');
                    
                    const successRate = 100 - errorRate;
                    document.getElementById('success').textContent = successRate.toFixed(1) + '%';
                    
                    document.getElementById('response').textContent = data.metrics.average_response_time || '0ms';
                    document.getElementById('uptime').textContent = formatUptime(Date.now() - startTime);
                    
                    const serversDiv = document.getElementById('servers');
                    serversDiv.innerHTML = '';
                    
                    let circuitCount = 0;
                    
                    for (const [url, server] of Object.entries(data.servers || {})) {
                        if (server.status === 'circuit_open') circuitCount++;
                        
                        const serverDiv = document.createElement('div');
                        serverDiv.className = 'server ' + (server.status || 'healthy');
                        
                        const statusClass = 'status-' + (server.status || 'healthy');
                        const statusText = (server.status || 'healthy').replace('_', ' ').toUpperCase();
                        
                        serverDiv.innerHTML = 
                            '<div class="server-header">' +
                                '<span class="server-url">' + url + '</span>' +
                                '<span class="server-status ' + statusClass + '">' + statusText + '</span>' +
                            '</div>' +
                            '<div class="server-stats">' +
                                '<div class="stat">' +
                                    '<span class="stat-label">Connections</span>' +
                                    '<span class="stat-value">' + (server.connections || 0) + '</span>' +
                                '</div>' +
                                '<div class="stat">' +
                                    '<span class="stat-label">Requests</span>' +
                                    '<span class="stat-value">' + formatNumber(server.total_requests || 0) + '</span>' +
                                '</div>' +
                                '<div class="stat">' +
                                    '<span class="stat-label">Failed</span>' +
                                    '<span class="stat-value">' + formatNumber(server.failed_requests || 0) + '</span>' +
                                '</div>' +
                                '<div class="stat">' +
                                    '<span class="stat-label">Response</span>' +
                                    '<span class="stat-value">' + (server.response_time || '0ms') + '</span>' +
                                '</div>' +
                                '<div class="stat">' +
                                    '<span class="stat-label">Weight</span>' +
                                    '<span class="stat-value">' + (server.weight || 1) + '</span>' +
                                '</div>' +
                            '</div>';
                        
                        serversDiv.appendChild(serverDiv);
                    }
                    
                    document.getElementById('circuits').textContent = circuitCount + ' Open';
                    document.getElementById('lastUpdate').textContent = new Date().toLocaleTimeString();
                })
                .catch(err => {
                    console.error('Error fetching metrics:', err);
                    document.getElementById('lastUpdate').textContent = 'Error loading data';
                });
        }
        
        updateStats();
        setInterval(updateStats, 1000);
    </script>
</body>
</html>`)
}