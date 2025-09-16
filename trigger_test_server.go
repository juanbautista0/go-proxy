package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

type TriggerEvent struct {
	Timestamp time.Time `json:"timestamp"`
	Action    string    `json:"action"`
	Reason    string    `json:"reason"`
}

var events = []TriggerEvent{}

func main() {
	port := "8091"
	http.HandleFunc("/scale/up", handleScaleUp)
	http.HandleFunc("/scale/down", handleScaleDown)
	http.HandleFunc("/morning", handleMorning)
	http.HandleFunc("/evening", handleEvening)
	http.HandleFunc("/events", handleEvents)
	http.HandleFunc("/", handleDashboard)

	fmt.Println("🎯 Trigger Test Server starting on :" + port)
	fmt.Println("📊 Dashboard: http://localhost:" + port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func handleScaleUp(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("📥 Received SCALE UP request from %s\n", r.RemoteAddr)
	
	event := TriggerEvent{
		Timestamp: time.Now(),
		Action:    "SCALE UP",
		Reason:    "High traffic detected",
	}
	events = append(events, event)

	fmt.Printf("🔥 SCALE UP triggered at %s (Total events: %d)\n", event.Timestamp.Format("15:04:05"), len(events))
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "scaled up"})
}

func handleScaleDown(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("📥 Received SCALE DOWN request from %s\n", r.RemoteAddr)
	
	event := TriggerEvent{
		Timestamp: time.Now(),
		Action:    "SCALE DOWN",
		Reason:    "Low traffic detected",
	}
	events = append(events, event)

	fmt.Printf("📉 SCALE DOWN triggered at %s (Total events: %d)\n", event.Timestamp.Format("15:04:05"), len(events))
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "scaled down"})
}

func handleMorning(w http.ResponseWriter, r *http.Request) {
	event := TriggerEvent{
		Timestamp: time.Now(),
		Action:    "MORNING SCALE",
		Reason:    "Scheduled morning scaling",
	}
	events = append(events, event)

	fmt.Printf("🌅 MORNING SCALE triggered at %s\n", event.Timestamp.Format("15:04:05"))
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "morning scaled"})
}

func handleEvening(w http.ResponseWriter, r *http.Request) {
	event := TriggerEvent{
		Timestamp: time.Now(),
		Action:    "EVENING SCALE",
		Reason:    "Scheduled evening scaling",
	}
	events = append(events, event)

	fmt.Printf("🌆 EVENING SCALE triggered at %s\n", event.Timestamp.Format("15:04:05"))
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "evening scaled"})
}

func handleEvents(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	
	// Asegurar que siempre devuelva un array, no null
	if events == nil {
		events = []TriggerEvent{}
	}
	
	json.NewEncoder(w).Encode(events)
}

func handleDashboard(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprint(w, `<!DOCTYPE html>
<html>
<head>   
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>🎯 Trigger Events Dashboard</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; background: #f5f5f5; }
        .container { max-width: 800px; margin: 0 auto; }
        .header { text-align: center; margin-bottom: 30px; }
        .event { background: white; margin: 10px 0; padding: 15px; border-radius: 8px; border-left: 4px solid #007bff; }
        .scale-up { border-color: #dc3545; }
        .scale-down { border-color: #28a745; }
        .morning { border-color: #ffc107; }
        .evening { border-color: #6f42c1; }
        .timestamp { color: #666; font-size: 0.9em; }
        .action { font-weight: bold; font-size: 1.1em; margin: 5px 0; }
        .reason { color: #555; }
        .no-events { text-align: center; color: #666; padding: 40px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>🎯 Trigger Events Dashboard</h1>
            <p>Monitoring Go-Proxy trigger actions in real-time</p>
        </div>
        <div id="events">Loading events...</div>
    </div>

    <script>
        function updateEvents() {
            fetch('/events')
                .then(r => r.json())
                .then(events => {
                    const container = document.getElementById('events');
                    
                    // Manejar casos donde events es null, undefined o array vacío
                    if (!events || !Array.isArray(events) || events.length === 0) {
                        container.innerHTML = '<div class="no-events">No trigger events yet. Generate some traffic to see triggers in action!</div>';
                        return;
                    }
                    
                    container.innerHTML = '';
                    
                    // Crear copia para no mutar el original
                    const sortedEvents = [...events].reverse();
                    
                    sortedEvents.forEach(event => {
                        const div = document.createElement('div');
                        const actionClass = event.action.toLowerCase().replace(/\s+/g, '-');
                        div.className = 'event ' + actionClass;
                        
                        div.innerHTML = 
                            '<div class="timestamp">' + new Date(event.timestamp).toLocaleString() + '</div>' +
                            '<div class="action">' + event.action + '</div>' +
                            '<div class="reason">' + event.reason + '</div>';
                        
                        container.appendChild(div);
                    });
                })
                .catch(err => {
                    console.error('Error fetching events:', err);
                    document.getElementById('events').innerHTML = '<div class="no-events">Error loading events</div>';
                });
        }
        
        // Inicializar
        updateEvents();
        setInterval(updateEvents, 1000);
    </script>
</body>
</html>`)
}
