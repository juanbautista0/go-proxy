package infrastructure

import (
	"os"
	"sync"

	"github.com/juanbautista0/go-proxy/internal/domain"
	"gopkg.in/yaml.v3"
)

type ConfigManager struct {
	configPath string
	mu         sync.RWMutex
	config     *domain.Config
	callbacks  []func(*domain.Config)
}

func NewConfigManager(configPath string) *ConfigManager {
	return &ConfigManager{
		configPath: configPath,
		callbacks:  make([]func(*domain.Config), 0),
	}
}

func (cm *ConfigManager) Load() (*domain.Config, error) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	data, err := os.ReadFile(cm.configPath)
	if err != nil {
		return nil, err
	}

	var config domain.Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	// Activar servidores por defecto
	for i := range config.Backends {
		for j := range config.Backends[i].Servers {
			config.Backends[i].Servers[j].Active = true
		}
	}

	cm.config = &config
	return &config, nil
}

func (cm *ConfigManager) Update(config *domain.Config) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// Escribir archivo primero
	data, err := yaml.Marshal(config)
	if err != nil {
		return err
	}

	if err := os.WriteFile(cm.configPath, data, 0644); err != nil {
		return err
	}

	// Actualizar memoria con copia
	configCopy := *config
	cm.config = &configCopy

	// Notificar callbacks
	for _, callback := range cm.callbacks {
		callback(&configCopy)
	}

	return nil
}

func (cm *ConfigManager) GetConfig() *domain.Config {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	// Crear copia profunda para evitar modificaciones concurrentes
	if cm.config == nil {
		return nil
	}
	configCopy := *cm.config
	return &configCopy
}

func (cm *ConfigManager) AddCallback(callback func(*domain.Config)) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.callbacks = append(cm.callbacks, callback)
}