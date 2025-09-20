package infrastructure

import (
	"sync"

	"github.com/juanbautista0/go-proxy/internal/domain"
)

type GracefulConfigManager struct {
	configManager *ConfigManager
	mu            sync.RWMutex
}

func NewGracefulConfigManager(configManager *ConfigManager) *GracefulConfigManager {
	return &GracefulConfigManager{
		configManager: configManager,
	}
}

func (gcm *GracefulConfigManager) RemoveServerFromConfig(serverURL string) error {
	gcm.mu.Lock()
	defer gcm.mu.Unlock()

	config := *gcm.configManager.GetConfig()
	
	// Buscar y remover servidor de la configuración
	for i := range config.Backends {
		for j, server := range config.Backends[i].Servers {
			if server.URL == serverURL {
				config.Backends[i].Servers = append(
					config.Backends[i].Servers[:j],
					config.Backends[i].Servers[j+1:]...,
				)
				return gcm.configManager.Update(&config)
			}
		}
	}
	
	return nil // Servidor no encontrado, no es error
}

func (gcm *GracefulConfigManager) GetConfig() *domain.Config {
	return gcm.configManager.GetConfig()
}

func (gcm *GracefulConfigManager) Update(config *domain.Config) error {
	return gcm.configManager.Update(config)
}