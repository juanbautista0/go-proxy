package infrastructure

import (
	"encoding/json"
	"net/http"

	"github.com/juanbautista0/go-proxy/internal/domain"
)

type ConfigAPI struct {
	configManager *ConfigManager
}

func NewConfigAPI(configManager *ConfigManager) *ConfigAPI {
	return &ConfigAPI{configManager: configManager}
}

func (api *ConfigAPI) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/servers":
		if !api.authenticate(r) {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		switch r.Method {
		case http.MethodPost:
			api.addServer(w, r)
		case http.MethodPut:
			api.updateServer(w, r)
		case http.MethodDelete:
			api.removeServer(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	case "/config":
		switch r.Method {
		case http.MethodGet:
			api.getConfig(w, r)
		case http.MethodPut:
			if !api.authenticate(r) {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
			api.updateConfig(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	case "/security":
		switch r.Method {
		case http.MethodGet:
			if !api.authenticateAdmin(r) {
				http.Error(w, "Admin access required", http.StatusForbidden)
				return
			}
			api.getSecurity(w, r)
		case http.MethodPut:
			if !api.authenticateAdmin(r) {
				http.Error(w, "Admin access required", http.StatusForbidden)
				return
			}
			api.updateSecurity(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	case "/actions/scale_up":
		api.handleScaleUp(w, r)
	case "/actions/scale_down":
		api.handleScaleDown(w, r)
	case "/actions/morning_scale":
		api.handleMorningScale(w, r)
	case "/actions/evening_scale":
		api.handleEveningScale(w, r)
	case "/swagger":
		swaggerHandler := NewSwaggerHandler()
		swaggerHandler.ServeHTTP(w, r)
	case "/api-docs.yaml":
		swaggerHandler := NewSwaggerHandler()
		swaggerHandler.ServeHTTP(w, r)
	default:
		http.Error(w, "Not found", http.StatusNotFound)
	}
}

func (api *ConfigAPI) authenticate(r *http.Request) bool {
	apiKey := r.Header.Get("X-API-KEY")
	if apiKey == "" {
		return false
	}
	
	config := api.configManager.GetConfig()
	// Verificar admin keys primero
	for _, validKey := range config.Security.AdminAPIKeys {
		if apiKey == validKey {
			return true
		}
	}
	// Verificar keys regulares
	for _, validKey := range config.Security.APIKeys {
		if apiKey == validKey {
			return true
		}
	}
	return false
}

func (api *ConfigAPI) authenticateAdmin(r *http.Request) bool {
	apiKey := r.Header.Get("X-API-KEY")
	if apiKey == "" {
		return false
	}
	
	config := api.configManager.GetConfig()
	// Solo admin keys pueden acceder
	for _, validKey := range config.Security.AdminAPIKeys {
		if apiKey == validKey {
			return true
		}
	}
	return false
}

func (api *ConfigAPI) getConfig(w http.ResponseWriter, r *http.Request) {
	config := *api.configManager.GetConfig()
	
	// Ocultar API keys por seguridad
	for i := range config.Security.APIKeys {
		config.Security.APIKeys[i] = "***"
	}
	for i := range config.Security.AdminAPIKeys {
		config.Security.AdminAPIKeys[i] = "***"
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(config)
}

func (api *ConfigAPI) getSecurity(w http.ResponseWriter, r *http.Request) {
	config := api.configManager.GetConfig()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(config.Security)
}

func (api *ConfigAPI) updateSecurity(w http.ResponseWriter, r *http.Request) {
	var newSecurity domain.SecurityConfig
	if err := json.NewDecoder(r.Body).Decode(&newSecurity); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Actualizar solo la configuración de seguridad
	config := *api.configManager.GetConfig()
	config.Security = newSecurity

	if err := api.configManager.Update(&config); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (api *ConfigAPI) updateConfig(w http.ResponseWriter, r *http.Request) {
	var newConfig domain.Config
	if err := json.NewDecoder(r.Body).Decode(&newConfig); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Preservar puerto original del proxy
	currentConfig := api.configManager.GetConfig()
	newConfig.Proxy.Port = currentConfig.Proxy.Port

	if err := api.configManager.Update(&newConfig); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

type AddServerRequest struct {
	BackendName           string `json:"backend_name"`
	URL                   string `json:"url"`
	Weight                int    `json:"weight"`
	MaxConnections        int    `json:"max_connections"`
	HealthCheckEndpoint   string `json:"health_check_endpoint"`
}

func (api *ConfigAPI) addServer(w http.ResponseWriter, r *http.Request) {
	var req AddServerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	config := *api.configManager.GetConfig()
	
	// Buscar backend y agregar servidor
	for i := range config.Backends {
		if config.Backends[i].Name == req.BackendName {
			// Validar límite máximo
			currentCount := len(config.Backends[i].Servers)
			if config.Backends[i].MaxServers > 0 && currentCount >= config.Backends[i].MaxServers {
				http.Error(w, "Maximum servers limit reached", http.StatusBadRequest)
				return
			}
			
			server := domain.Server{
				URL:                 req.URL,
				Weight:              req.Weight,
				MaxConnections:      req.MaxConnections,
				HealthCheckEndpoint: req.HealthCheckEndpoint,
				Active:              true,
			}
			config.Backends[i].Servers = append(config.Backends[i].Servers, server)
			
			if err := api.configManager.Update(&config); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			
			w.WriteHeader(http.StatusCreated)
			return
		}
	}
	
	http.Error(w, "Backend not found", http.StatusNotFound)
}

type UpdateServerRequest struct {
	BackendName           string `json:"backend_name"`
	OldURL                string `json:"old_url"`
	URL                   string `json:"url"`
	Weight                int    `json:"weight"`
	MaxConnections        int    `json:"max_connections"`
	HealthCheckEndpoint   string `json:"health_check_endpoint"`
}

type RemoveServerRequest struct {
	BackendName string `json:"backend_name"`
	ServerURL   string `json:"server_url"`
}

func (api *ConfigAPI) removeServer(w http.ResponseWriter, r *http.Request) {
	var req RemoveServerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	config := *api.configManager.GetConfig()
	
	// Buscar backend y remover servidor
	for i := range config.Backends {
		if config.Backends[i].Name == req.BackendName {
			// Validar límite mínimo
			currentCount := len(config.Backends[i].Servers)
			if config.Backends[i].MinServers > 0 && currentCount <= config.Backends[i].MinServers {
				http.Error(w, "Minimum servers limit reached", http.StatusBadRequest)
				return
			}
			
			for j, server := range config.Backends[i].Servers {
				if server.URL == req.ServerURL {
					config.Backends[i].Servers = append(
						config.Backends[i].Servers[:j],
						config.Backends[i].Servers[j+1:]...,
					)
					
					if err := api.configManager.Update(&config); err != nil {
						http.Error(w, err.Error(), http.StatusInternalServerError)
						return
					}
					
					w.WriteHeader(http.StatusOK)
					return
				}
			}
		}
	}
	
	http.Error(w, "Server not found", http.StatusNotFound)
}

func (api *ConfigAPI) updateServer(w http.ResponseWriter, r *http.Request) {
	var req UpdateServerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	config := *api.configManager.GetConfig()
	
	// Buscar backend y actualizar servidor
	for i := range config.Backends {
		if config.Backends[i].Name == req.BackendName {
			for j, server := range config.Backends[i].Servers {
				if server.URL == req.OldURL {
					config.Backends[i].Servers[j] = domain.Server{
						URL:                 req.URL,
						Weight:              req.Weight,
						MaxConnections:      req.MaxConnections,
						HealthCheckEndpoint: req.HealthCheckEndpoint,
						Active:              true,
					}
					
					if err := api.configManager.Update(&config); err != nil {
						http.Error(w, err.Error(), http.StatusInternalServerError)
						return
					}
					
					w.WriteHeader(http.StatusOK)
					return
				}
			}
		}
	}
	
	http.Error(w, "Server not found", http.StatusNotFound)
}