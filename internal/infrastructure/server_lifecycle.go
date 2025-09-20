package infrastructure

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/juanbautista0/go-proxy/internal/domain"
)

type ServerLifecycle struct {
	mu                sync.RWMutex
	pendingRemovals   map[string]*RemovalState
	drainTimeout      time.Duration
	checkInterval     time.Duration
	onServerRemoved   func(serverURL string)
	onServerDrained   func(serverURL string)
}

type RemovalState struct {
	Server          *domain.Server
	StartTime       time.Time
	DrainDeadline   time.Time
	Context         context.Context
	Cancel          context.CancelFunc
	ConnectionCount *int64
}

func NewServerLifecycle() *ServerLifecycle {
	return &ServerLifecycle{
		pendingRemovals: make(map[string]*RemovalState),
		drainTimeout:    30 * time.Second,
		checkInterval:   time.Second,
	}
}

func (sl *ServerLifecycle) SetCallbacks(onRemoved, onDrained func(string)) {
	sl.onServerRemoved = onRemoved
	sl.onServerDrained = onDrained
}

func (sl *ServerLifecycle) StartGracefulRemoval(server *domain.Server, connectionCount *int64) {
	sl.mu.Lock()
	defer sl.mu.Unlock()

	if _, exists := sl.pendingRemovals[server.URL]; exists {
		return // Ya está en proceso
	}

	ctx, cancel := context.WithTimeout(context.Background(), sl.drainTimeout)
	now := time.Now()

	removal := &RemovalState{
		Server:          server,
		StartTime:       now,
		DrainDeadline:   now.Add(sl.drainTimeout),
		Context:         ctx,
		Cancel:          cancel,
		ConnectionCount: connectionCount,
	}

	sl.pendingRemovals[server.URL] = removal

	// Marcar servidor como inactivo para nuevas conexiones
	server.Active = false

	// Iniciar monitoreo en goroutine
	go sl.monitorDraining(server.URL)
}

func (sl *ServerLifecycle) monitorDraining(serverURL string) {
	ticker := time.NewTicker(sl.checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if sl.checkAndFinalizeDrain(serverURL) {
				return
			}
		}
	}
}

func (sl *ServerLifecycle) checkAndFinalizeDrain(serverURL string) bool {
	sl.mu.Lock()
	removal, exists := sl.pendingRemovals[serverURL]
	if !exists {
		sl.mu.Unlock()
		return true
	}

	connections := atomic.LoadInt64(removal.ConnectionCount)
	now := time.Now()
	
	// Verificar si se completó el drenado o se agotó el tiempo
	if connections == 0 || now.After(removal.DrainDeadline) {
		delete(sl.pendingRemovals, serverURL)
		removal.Cancel()
		sl.mu.Unlock()

		// Callbacks no bloqueantes
		if sl.onServerDrained != nil {
			go sl.onServerDrained(serverURL)
		}
		if sl.onServerRemoved != nil {
			go sl.onServerRemoved(serverURL)
		}
		return true
	}
	
	sl.mu.Unlock()
	return false
}

func (sl *ServerLifecycle) IsServerDraining(serverURL string) bool {
	sl.mu.RLock()
	defer sl.mu.RUnlock()
	_, exists := sl.pendingRemovals[serverURL]
	return exists
}

func (sl *ServerLifecycle) GetDrainingServers() []string {
	sl.mu.RLock()
	defer sl.mu.RUnlock()
	
	servers := make([]string, 0, len(sl.pendingRemovals))
	for url := range sl.pendingRemovals {
		servers = append(servers, url)
	}
	return servers
}

func (sl *ServerLifecycle) CancelRemoval(serverURL string) bool {
	sl.mu.Lock()
	defer sl.mu.Unlock()
	
	removal, exists := sl.pendingRemovals[serverURL]
	if !exists {
		return false
	}
	
	removal.Cancel()
	removal.Server.Active = true // Reactivar servidor
	delete(sl.pendingRemovals, serverURL)
	return true
}

func (sl *ServerLifecycle) ForceRemoval(serverURL string) {
	sl.mu.Lock()
	removal, exists := sl.pendingRemovals[serverURL]
	if exists {
		removal.Cancel()
		delete(sl.pendingRemovals, serverURL)
		sl.mu.Unlock()
		
		if sl.onServerRemoved != nil {
			go sl.onServerRemoved(serverURL)
		}
		return
	}
	sl.mu.Unlock()
}