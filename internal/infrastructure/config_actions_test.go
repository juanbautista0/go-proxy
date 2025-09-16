package infrastructure

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/juanbautista0/go-proxy/internal/domain"
)

func TestConfigAPI_HandleScaleUp(t *testing.T) {
	api, tempFile := setupTestConfigAPI(t)
	defer os.Remove(tempFile)

	req := httptest.NewRequest("POST", "/actions/scale_up", nil)
	w := httptest.NewRecorder()

	api.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	// Verify server was added
	config := api.configManager.GetConfig()
	if len(config.Backends[0].Servers) != 2 {
		t.Errorf("expected 2 servers after scale up, got %d", len(config.Backends[0].Servers))
	}

	newServer := config.Backends[0].Servers[1]
	if newServer.URL != "http://localhost:3004" {
		t.Errorf("expected new server URL http://localhost:3004, got %s", newServer.URL)
	}

	expectedResponse := `{"status":"scaled_up"}`
	if w.Body.String() != expectedResponse {
		t.Errorf("expected response %s, got %s", expectedResponse, w.Body.String())
	}
}

func TestConfigAPI_HandleScaleDown(t *testing.T) {
	api, tempFile := setupTestConfigAPI(t)
	defer os.Remove(tempFile)

	// First add a server to have multiple servers
	config := api.configManager.GetConfig()
	config.Backends[0].Servers = append(config.Backends[0].Servers, domain.Server{
		URL:    "http://localhost:3004",
		Weight: 1,
		Active: true,
	})
	api.configManager.Update(config)

	req := httptest.NewRequest("POST", "/actions/scale_down", nil)
	w := httptest.NewRecorder()

	api.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	// Verify server was removed
	updatedConfig := api.configManager.GetConfig()
	if len(updatedConfig.Backends[0].Servers) != 1 {
		t.Errorf("expected 1 server after scale down, got %d", len(updatedConfig.Backends[0].Servers))
	}

	expectedResponse := `{"status":"scaled_down"}`
	if w.Body.String() != expectedResponse {
		t.Errorf("expected response %s, got %s", expectedResponse, w.Body.String())
	}
}

func TestConfigAPI_HandleScaleDown_SingleServer(t *testing.T) {
	api, tempFile := setupTestConfigAPI(t)
	defer os.Remove(tempFile)

	// Test scale down with only one server (should not remove)
	req := httptest.NewRequest("POST", "/actions/scale_down", nil)
	w := httptest.NewRecorder()

	api.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	// Verify server was not removed (minimum 1 server)
	config := api.configManager.GetConfig()
	if len(config.Backends[0].Servers) != 1 {
		t.Errorf("expected 1 server to remain, got %d", len(config.Backends[0].Servers))
	}
}

func TestConfigAPI_HandleMorningScale(t *testing.T) {
	api, tempFile := setupTestConfigAPI(t)
	defer os.Remove(tempFile)

	originalWeight := api.configManager.GetConfig().Backends[0].Servers[0].Weight

	req := httptest.NewRequest("POST", "/actions/morning_scale", nil)
	w := httptest.NewRecorder()

	api.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	// Verify weights were increased
	config := api.configManager.GetConfig()
	newWeight := config.Backends[0].Servers[0].Weight
	if newWeight != originalWeight+1 {
		t.Errorf("expected weight %d, got %d", originalWeight+1, newWeight)
	}

	expectedResponse := `{"status":"morning_scaled"}`
	if w.Body.String() != expectedResponse {
		t.Errorf("expected response %s, got %s", expectedResponse, w.Body.String())
	}
}

func TestConfigAPI_HandleEveningScale(t *testing.T) {
	api, tempFile := setupTestConfigAPI(t)
	defer os.Remove(tempFile)

	// Set initial weight > 1
	config := api.configManager.GetConfig()
	config.Backends[0].Servers[0].Weight = 3
	api.configManager.Update(config)

	req := httptest.NewRequest("POST", "/actions/evening_scale", nil)
	w := httptest.NewRecorder()

	api.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	// Verify weights were decreased
	updatedConfig := api.configManager.GetConfig()
	newWeight := updatedConfig.Backends[0].Servers[0].Weight
	if newWeight != 2 {
		t.Errorf("expected weight 2, got %d", newWeight)
	}

	expectedResponse := `{"status":"evening_scaled"}`
	if w.Body.String() != expectedResponse {
		t.Errorf("expected response %s, got %s", expectedResponse, w.Body.String())
	}
}

func TestConfigAPI_HandleEveningScale_MinWeight(t *testing.T) {
	api, tempFile := setupTestConfigAPI(t)
	defer os.Remove(tempFile)

	// Test with weight = 1 (should not decrease below 1)
	req := httptest.NewRequest("POST", "/actions/evening_scale", nil)
	w := httptest.NewRecorder()

	api.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	// Verify weight stayed at 1
	config := api.configManager.GetConfig()
	weight := config.Backends[0].Servers[0].Weight
	if weight != 1 {
		t.Errorf("expected weight to remain 1, got %d", weight)
	}
}

func TestConfigAPI_ActionsInvalidMethods(t *testing.T) {
	api, tempFile := setupTestConfigAPI(t)
	defer os.Remove(tempFile)

	actions := []string{
		"/actions/scale_up",
		"/actions/scale_down", 
		"/actions/morning_scale",
		"/actions/evening_scale",
	}

	for _, action := range actions {
		t.Run(action+"_GET", func(t *testing.T) {
			req := httptest.NewRequest("GET", action, nil)
			w := httptest.NewRecorder()

			api.ServeHTTP(w, req)

			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("expected status 405, got %d", w.Code)
			}
		})
	}
}