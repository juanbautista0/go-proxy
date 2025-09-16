package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

type StatsResponse struct {
	Timestamp time.Time              `json:"timestamp"`
	Metrics   interface{}            `json:"metrics"`
	Servers   map[string]interface{} `json:"servers"`
}

func main() {
	http.HandleFunc("/stats", func(w http.ResponseWriter, r *http.Request) {
		// Simulación de estadísticas (en producción vendría del proxy)
		stats := StatsResponse{
			Timestamp: time.Now(),
			Metrics: map[string]interface{}{
				"requests_per_second":   45,
				"total_requests":        12450,
				"active_connections":    23,
				"average_response_time": "120ms",
				"error_rate":            0.02,
			},
			Servers: map[string]interface{}{
				"http://localhost:3001": map[string]interface{}{
					"status":          "healthy",
					"connections":     12,
					"total_requests":  7200,
					"failed_requests": 15,
					"response_time":   "110ms",
					"weight":          3,
				},
				"http://localhost:3002": map[string]interface{}{
					"status":          "healthy",
					"connections":     11,
					"total_requests":  5250,
					"failed_requests": 8,
					"response_time":   "130ms",
					"weight":          2,
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(stats)
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `
<!DOCTYPE html>
<html>
<head>
    <title>HAProxy Stats</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        .stats { display: flex; gap: 20px; margin-bottom: 20px; }
        .card { border: 1px solid #ddd; padding: 15px; border-radius: 5px; flex: 1; }
        .server { margin: 10px 0; padding: 10px; border-left: 4px solid #4CAF50; }
        .healthy { border-left-color: #4CAF50; }
        .unhealthy { border-left-color: #f44336; }
    </style>
</head>
<body>
    <h1>🚀 Go-Proxy Stats</h1>
    
    <div class="stats">
        <div class="card">
            <h3>📊 Traffic Metrics</h3>
            <p><strong>RPS:</strong> <span id="rps">45</span></p>
            <p><strong>Total Requests:</strong> <span id="total">12,450</span></p>
            <p><strong>Active Connections:</strong> <span id="active">23</span></p>
            <p><strong>Error Rate:</strong> <span id="error">2%</span></p>
        </div>
        
        <div class="card">
            <h3>⚡ Performance</h3>
            <p><strong>Avg Response Time:</strong> <span id="response">120ms</span></p>
            <p><strong>Load Balance:</strong> Weighted Round Robin</p>
            <p><strong>Health Checks:</strong> ✅ Active</p>
            <p><strong>Circuit Breaker:</strong> ✅ Enabled</p>
        </div>
    </div>

    <div class="card">
        <h3>🖥️ Backend Servers</h3>
        <div class="server healthy">
            <strong>Server 1 (localhost:3001)</strong> - Weight: 3<br>
            Status: ✅ Healthy | Connections: 12 | Requests: 7,200 | Response: 110ms
        </div>
        <div class="server healthy">
            <strong>Server 2 (localhost:3002)</strong> - Weight: 2<br>
            Status: ✅ Healthy | Connections: 11 | Requests: 5,250 | Response: 130ms
        </div>
    </div>

    <script>
        setInterval(() => {
            fetch('/stats')
                .then(r => r.json())
                .then(data => {
                    document.getElementById('rps').textContent = data.metrics.requests_per_second;
                    document.getElementById('total').textContent = data.metrics.total_requests.toLocaleString();
                    document.getElementById('active').textContent = data.metrics.active_connections;
                    document.getElementById('error').textContent = (data.metrics.error_rate * 100).toFixed(1) + '%';
                    document.getElementById('response').textContent = data.metrics.average_response_time;
                });
        }, 1000);
    </script>
</body>
</html>`)
	})

	log.Println("Stats server running on http://localhost:8081")
	log.Fatal(http.ListenAndServe(":8081", nil))
}
