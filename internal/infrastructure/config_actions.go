package infrastructure

import (
	"net/http"

	"github.com/juanbautista0/go-proxy/internal/domain"
)

func (api *ConfigAPI) handleScaleUp(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	config := *api.configManager.GetConfig()
	for i := range config.Backends {
		if config.Backends[i].Name == "web-servers" {
			// Validar límite máximo
			currentCount := len(config.Backends[i].Servers)
			if config.Backends[i].MaxServers > 0 && currentCount >= config.Backends[i].MaxServers {
				http.Error(w, "Maximum servers limit reached", http.StatusBadRequest)
				return
			}
			
			newServer := domain.Server{
				URL:                 "http://localhost:3004",
				Weight:              1,
				MaxConnections:      100,
				HealthCheckEndpoint: "/health",
				Active:              true,
			}
			config.Backends[i].Servers = append(config.Backends[i].Servers, newServer)
			break
		}
	}
	
	if err := api.configManager.Update(&config); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"scaled_up"}`))
}

func (api *ConfigAPI) handleScaleDown(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	config := *api.configManager.GetConfig()
	for i := range config.Backends {
		if config.Backends[i].Name == "web-servers" {
			// Validar límite mínimo
			currentCount := len(config.Backends[i].Servers)
			if config.Backends[i].MinServers > 0 && currentCount <= config.Backends[i].MinServers {
				http.Error(w, "Minimum servers limit reached", http.StatusBadRequest)
				return
			}
			
			if currentCount > 1 {
				config.Backends[i].Servers = config.Backends[i].Servers[:currentCount-1]
			}
			break
		}
	}
	
	if err := api.configManager.Update(&config); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"scaled_down"}`))
}

func (api *ConfigAPI) handleMorningScale(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	config := *api.configManager.GetConfig()
	for i := range config.Backends {
		if config.Backends[i].Name == "web-servers" {
			for j := range config.Backends[i].Servers {
				config.Backends[i].Servers[j].Weight += 1
			}
			break
		}
	}
	
	if err := api.configManager.Update(&config); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"morning_scaled"}`))
}

func (api *ConfigAPI) handleEveningScale(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	config := *api.configManager.GetConfig()
	for i := range config.Backends {
		if config.Backends[i].Name == "web-servers" {
			for j := range config.Backends[i].Servers {
				if config.Backends[i].Servers[j].Weight > 1 {
					config.Backends[i].Servers[j].Weight -= 1
				}
			}
			break
		}
	}
	
	if err := api.configManager.Update(&config); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"evening_scaled"}`))
}