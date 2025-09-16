package infrastructure

import (
	"os"
	"testing"
	"time"

	"github.com/juanbautista0/go-proxy/internal/domain"
)

func TestConfigManager_Load(t *testing.T) {
	// Create temp config file
	tempFile, err := os.CreateTemp("", "config_test_*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tempFile.Name())

	configContent := `
proxy:
  port: 8080
backends:
  - name: "test-backend"
    servers:
      - url: "http://localhost:3001"
        weight: 1
        max_connections: 100
    health_check: "/health"
`
	tempFile.WriteString(configContent)
	tempFile.Close()

	manager := NewConfigManager(tempFile.Name())
	config, err := manager.Load()

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if config.Proxy.Port != 8080 {
		t.Errorf("expected port 8080, got %d", config.Proxy.Port)
	}

	if len(config.Backends) != 1 {
		t.Errorf("expected 1 backend, got %d", len(config.Backends))
	}

	if config.Backends[0].Name != "test-backend" {
		t.Errorf("expected backend name 'test-backend', got %s", config.Backends[0].Name)
	}

	// Check server is activated by default
	if !config.Backends[0].Servers[0].Active {
		t.Error("expected server to be active by default")
	}
}

func TestConfigManager_Update(t *testing.T) {
	tempFile, err := os.CreateTemp("", "config_test_*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tempFile.Name())

	manager := NewConfigManager(tempFile.Name())
	
	// Create test config
	config := &domain.Config{
		Proxy: domain.ProxyConfig{Port: 9090},
		Backends: []domain.Backend{
			{
				Name: "updated-backend",
				Servers: []domain.Server{
					{
						URL:    "http://localhost:4001",
						Weight: 2,
						Active: true,
					},
				},
			},
		},
	}

	// Test callback
	callbackCalled := false
	manager.AddCallback(func(c *domain.Config) {
		callbackCalled = true
		if c.Proxy.Port != 9090 {
			t.Errorf("callback received wrong port: %d", c.Proxy.Port)
		}
	})

	err = manager.Update(config)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !callbackCalled {
		t.Error("expected callback to be called")
	}

	// Verify file was written
	retrievedConfig := manager.GetConfig()
	if retrievedConfig.Proxy.Port != 9090 {
		t.Errorf("expected port 9090, got %d", retrievedConfig.Proxy.Port)
	}
}

func TestConfigManager_GetConfig(t *testing.T) {
	tempFile, err := os.CreateTemp("", "config_test_*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tempFile.Name())

	manager := NewConfigManager(tempFile.Name())
	
	// Should return nil before loading
	config := manager.GetConfig()
	if config != nil {
		t.Error("expected nil config before loading")
	}

	// Load config
	configContent := `
proxy:
  port: 8080
backends: []
`
	tempFile.WriteString(configContent)
	tempFile.Close()

	_, err = manager.Load()
	if err != nil {
		t.Fatal(err)
	}

	config = manager.GetConfig()
	if config == nil {
		t.Error("expected config after loading")
	}
}

func TestConfigManager_ConcurrentAccess(t *testing.T) {
	tempFile, err := os.CreateTemp("", "config_test_*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tempFile.Name())

	configContent := `
proxy:
  port: 8080
backends: []
`
	tempFile.WriteString(configContent)
	tempFile.Close()

	manager := NewConfigManager(tempFile.Name())
	manager.Load()

	// Test concurrent reads and writes
	done := make(chan bool, 10)

	// Start multiple readers
	for i := 0; i < 5; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				config := manager.GetConfig()
				if config == nil {
					t.Error("config should not be nil")
				}
				time.Sleep(time.Microsecond)
			}
			done <- true
		}()
	}

	// Start multiple writers
	for i := 0; i < 5; i++ {
		go func(id int) {
			config := &domain.Config{
				Proxy:    domain.ProxyConfig{Port: 8080 + id},
				Backends: []domain.Backend{},
			}
			for j := 0; j < 10; j++ {
				manager.Update(config)
				time.Sleep(time.Microsecond)
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}