package infrastructure

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/juanbautista0/go-proxy/internal/domain"
)

func TestHTTPActionExecutor_NetworkError(t *testing.T) {
	executor := NewHTTPActionExecutor()
	config := domain.ActionConfig{
		URL:    "http://localhost:99999/unreachable",
		Method: "POST",
	}

	// HTTPActionExecutor executes asynchronously and always returns nil
	err := executor.Execute("network_error", config)
	if err != nil {
		t.Errorf("expected no error for async execution, got %v", err)
	}
}

func TestHTTPActionExecutor_Timeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	executor := NewHTTPActionExecutor()
	executor.client = &http.Client{
		Timeout: 50 * time.Millisecond,
	}
	
	config := domain.ActionConfig{
		URL:    server.URL + "/slow",
		Method: "POST",
	}

	// HTTPActionExecutor executes asynchronously and always returns nil
	err := executor.Execute("timeout_action", config)
	if err != nil {
		t.Errorf("expected no error for async execution, got %v", err)
	}
}

func TestHTTPActionExecutor_DifferentMethods(t *testing.T) {
	methods := []string{"GET", "POST", "PUT", "DELETE"}
	
	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != method {
					t.Errorf("expected method %s, got %s", method, r.Method)
				}
				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()

			executor := NewHTTPActionExecutor()
			config := domain.ActionConfig{
				URL:    server.URL + "/test",
				Method: method,
			}

			err := executor.Execute("test_action", config)
			if err != nil {
				t.Errorf("expected no error for %s method, got %v", method, err)
			}
		})
	}
}