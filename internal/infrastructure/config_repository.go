package infrastructure

import (
	"os"

	"github.com/fsnotify/fsnotify"
	"github.com/juanbautista0/go-proxy/internal/domain"
	"gopkg.in/yaml.v3"
)

type FileConfigRepository struct {
	configPath string
}

func NewFileConfigRepository(configPath string) *FileConfigRepository {
	return &FileConfigRepository{configPath: configPath}
}

func (r *FileConfigRepository) Load() (*domain.Config, error) {
	data, err := os.ReadFile(r.configPath)
	if err != nil {
		return nil, err
	}

	var config domain.Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	// Activar todos los servidores por defecto
	for i := range config.Backends {
		for j := range config.Backends[i].Servers {
			config.Backends[i].Servers[j].Active = true
		}
	}

	return &config, nil
}

func (r *FileConfigRepository) Watch(callback func(*domain.Config)) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}

	go func() {
		defer watcher.Close()
		for {
			select {
			case event := <-watcher.Events:
				// Solo recargar si es el archivo config.yaml específico
				if event.Name == r.configPath && event.Op&fsnotify.Write == fsnotify.Write {
					if config, err := r.Load(); err == nil {
						callback(config)
					}
				}
			case <-watcher.Errors:
				return
			}
		}
	}()

	return watcher.Add(r.configPath)
}
