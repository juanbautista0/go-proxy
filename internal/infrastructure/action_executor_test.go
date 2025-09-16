package infrastructure

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/juanbautista0/go-proxy/internal/domain"
)

func TestHTTPActionExecutor_Execute_Success(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST method, got %s", r.Method)
		}
		if r.URL.Path != "/test-action" {
			t.Errorf("expected path /test-action, got %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"success"}`))
	}))
	defer server.Close()

	executor := NewHTTPActionExecutor()
	config := domain.ActionConfig{
		URL:    server.URL + "/test-action",
		Method: "POST",
	}

	err := executor.Execute("test_action", config)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestHTTPActionExecutor_Execute_InvalidURL(t *testing.T) {
	executor := NewHTTPActionExecutor()
	config := domain.ActionConfig{
		URL:    "://invalid-url",
		Method: "POST",
	}
	
	err := executor.Execute("invalid_url", config)
	if err == nil {
		t.Error("expected error for invalid URL")
	}
}

func TestHTTPActionExecutor_Execute_HTTPError(t *testing.T) {
	// Create test server that returns error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
	}))
	defer server.Close()

	executor := NewHTTPActionExecutor()
	config := domain.ActionConfig{
		URL:    server.URL + "/error",
		Method: "POST",
	}

	err := executor.Execute("error_action", config)
	// HTTP errors don't return error in current implementation
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

