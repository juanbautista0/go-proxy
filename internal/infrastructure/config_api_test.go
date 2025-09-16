package infrastructure

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/juanbautista0/go-proxy/internal/domain"
)

func setupTestConfigAPI(t *testing.T) (*ConfigAPI, string) {
	tempFile, err := os.CreateTemp("", "config_api_test_*.yaml")
	if err != nil {
		t.Fatal(err)
	}

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
`
	tempFile.WriteString(configContent)
	tempFile.Close()

	manager := NewConfigManager(tempFile.Name())
	manager.Load()
	
	return NewConfigAPI(manager), tempFile.Name()
}

func TestConfigAPI_GetConfig(t *testing.T) {
	api, tempFile := setupTestConfigAPI(t)
	defer os.Remove(tempFile)

	req := httptest.NewRequest("GET", "/config", nil)
	w := httptest.NewRecorder()

	api.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var config domain.Config
	err := json.Unmarshal(w.Body.Bytes(), &config)
	if err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if config.Proxy.Port != 8080 {
		t.Errorf("expected port 8080, got %d", config.Proxy.Port)
	}
}

func TestConfigAPI_AddServer(t *testing.T) {
	api, tempFile := setupTestConfigAPI(t)
	defer os.Remove(tempFile)

	addReq := AddServerRequest{
		BackendName:         "web-servers",
		URL:                 "http://localhost:3004",
		Weight:              2,
		MaxConnections:      150,
		HealthCheckEndpoint: "/status",
	}

	body, _ := json.Marshal(addReq)
	req := httptest.NewRequest("POST", "/servers", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	api.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d", w.Code)
	}

	// Verify server was added
	config := api.configManager.GetConfig()
	if len(config.Backends[0].Servers) != 2 {
		t.Errorf("expected 2 servers, got %d", len(config.Backends[0].Servers))
	}

	newServer := config.Backends[0].Servers[1]
	if newServer.URL != "http://localhost:3004" {
		t.Errorf("expected URL http://localhost:3004, got %s", newServer.URL)
	}
	if newServer.Weight != 2 {
		t.Errorf("expected weight 2, got %d", newServer.Weight)
	}
}

func TestConfigAPI_UpdateServer(t *testing.T) {
	api, tempFile := setupTestConfigAPI(t)
	defer os.Remove(tempFile)

	updateReq := UpdateServerRequest{
		BackendName:         "web-servers",
		OldURL:              "http://localhost:3001",
		URL:                 "http://localhost:3005",
		Weight:              3,
		MaxConnections:      200,
		HealthCheckEndpoint: "/new-health",
	}

	body, _ := json.Marshal(updateReq)
	req := httptest.NewRequest("PUT", "/servers", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	api.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	// Verify server was updated
	config := api.configManager.GetConfig()
	server := config.Backends[0].Servers[0]
	if server.URL != "http://localhost:3005" {
		t.Errorf("expected URL http://localhost:3005, got %s", server.URL)
	}
	if server.Weight != 3 {
		t.Errorf("expected weight 3, got %d", server.Weight)
	}
}

func TestConfigAPI_RemoveServer(t *testing.T) {
	api, tempFile := setupTestConfigAPI(t)
	defer os.Remove(tempFile)

	// First add a second server
	addReq := AddServerRequest{
		BackendName: "web-servers",
		URL:         "http://localhost:3004",
		Weight:      1,
	}
	body, _ := json.Marshal(addReq)
	req := httptest.NewRequest("POST", "/servers", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	api.ServeHTTP(httptest.NewRecorder(), req)

	// Now remove the original server
	removeReq := RemoveServerRequest{
		BackendName: "web-servers",
		ServerURL:   "http://localhost:3001",
	}

	body, _ = json.Marshal(removeReq)
	req = httptest.NewRequest("DELETE", "/servers", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	api.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	// Verify server was removed
	config := api.configManager.GetConfig()
	if len(config.Backends[0].Servers) != 1 {
		t.Errorf("expected 1 server, got %d", len(config.Backends[0].Servers))
	}
	if config.Backends[0].Servers[0].URL != "http://localhost:3004" {
		t.Errorf("wrong server remained: %s", config.Backends[0].Servers[0].URL)
	}
}

func TestConfigAPI_InvalidRequests(t *testing.T) {
	api, tempFile := setupTestConfigAPI(t)
	defer os.Remove(tempFile)

	tests := []struct {
		name           string
		method         string
		path           string
		body           string
		expectedStatus int
	}{
		{
			name:           "invalid JSON",
			method:         "POST",
			path:           "/servers",
			body:           "invalid json",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "nonexistent backend",
			method:         "POST",
			path:           "/servers",
			body:           `{"backend_name":"nonexistent","url":"http://test"}`,
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "method not allowed",
			method:         "PATCH",
			path:           "/servers",
			body:           "",
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			name:           "not found path",
			method:         "GET",
			path:           "/invalid",
			body:           "",
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			api.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}