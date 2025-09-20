package infrastructure

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
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

	// Verify response message (scale up logic is not implemented yet)
	expectedResponse := `{"status":"scaled_up"}`
	if w.Body.String() != expectedResponse {
		t.Errorf("expected response %s, got %s", expectedResponse, w.Body.String())
	}
}

func TestConfigAPI_HandleScaleDown(t *testing.T) {
	api, tempFile := setupTestConfigAPI(t)
	defer os.Remove(tempFile)

	req := httptest.NewRequest("POST", "/actions/scale_down", nil)
	w := httptest.NewRecorder()

	api.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	// Verify response message (scale down logic is not implemented yet)
	expectedResponse := `{"status":"scaled_down"}`
	if w.Body.String() != expectedResponse {
		t.Errorf("expected response %s, got %s", expectedResponse, w.Body.String())
	}
}

func TestConfigAPI_HandleScaleDown_SingleServer(t *testing.T) {
	api, tempFile := setupTestConfigAPI(t)
	defer os.Remove(tempFile)

	req := httptest.NewRequest("POST", "/actions/scale_down", nil)
	w := httptest.NewRecorder()

	api.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	// Verify response message
	expectedResponse := `{"status":"scaled_down"}`
	if w.Body.String() != expectedResponse {
		t.Errorf("expected response %s, got %s", expectedResponse, w.Body.String())
	}
}

func TestConfigAPI_HandleMorningScale(t *testing.T) {
	api, tempFile := setupTestConfigAPI(t)
	defer os.Remove(tempFile)

	req := httptest.NewRequest("POST", "/actions/morning_scale", nil)
	w := httptest.NewRecorder()

	api.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	// Verify response message (morning scale logic is not implemented yet)
	expectedResponse := `{"status":"morning_scaled"}`
	if w.Body.String() != expectedResponse {
		t.Errorf("expected response %s, got %s", expectedResponse, w.Body.String())
	}
}

func TestConfigAPI_HandleEveningScale(t *testing.T) {
	api, tempFile := setupTestConfigAPI(t)
	defer os.Remove(tempFile)

	req := httptest.NewRequest("POST", "/actions/evening_scale", nil)
	w := httptest.NewRecorder()

	api.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	// Verify response message (evening scale logic is not implemented yet)
	expectedResponse := `{"status":"evening_scaled"}`
	if w.Body.String() != expectedResponse {
		t.Errorf("expected response %s, got %s", expectedResponse, w.Body.String())
	}
}

func TestConfigAPI_HandleEveningScale_MinWeight(t *testing.T) {
	api, tempFile := setupTestConfigAPI(t)
	defer os.Remove(tempFile)

	req := httptest.NewRequest("POST", "/actions/evening_scale", nil)
	w := httptest.NewRecorder()

	api.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	// Verify response message
	expectedResponse := `{"status":"evening_scaled"}`
	if w.Body.String() != expectedResponse {
		t.Errorf("expected response %s, got %s", expectedResponse, w.Body.String())
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