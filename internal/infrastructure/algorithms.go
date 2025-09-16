package infrastructure

import (
	"crypto/md5"
	"fmt"
	"math"
	"math/rand"
	"sort"
	"sync/atomic"
	"time"
)

// Adaptive Weighted Round Robin con machine learning
type AdaptiveWeightedRoundRobin struct {
	lastUpdate time.Time
}

func (a *AdaptiveWeightedRoundRobin) SelectServer(servers []*ServerState, clientIP string) *ServerState {
	if len(servers) == 0 {
		return nil
	}

	a.UpdateWeights(servers)
	
	// Smooth weighted round robin (nginx algorithm)
	var selected *ServerState
	totalWeight := 0.0

	for _, server := range servers {
		server.CurrentWeight += server.EffectiveWeight
		totalWeight += server.EffectiveWeight
		
		if selected == nil || server.CurrentWeight > selected.CurrentWeight {
			selected = server
		}
	}

	if selected != nil {
		selected.CurrentWeight -= totalWeight
	}

	return selected
}

func (a *AdaptiveWeightedRoundRobin) UpdateWeights(servers []*ServerState) {
	now := time.Now()
	if now.Sub(a.lastUpdate) < 5*time.Second {
		return
	}
	a.lastUpdate = now

	// Calcular pesos adaptativos basados en performance
	for _, server := range servers {
		baseWeight := server.Weight
		
		// Factor de error rate (0.5 - 1.5)
		errorFactor := 1.0
		if server.Metrics.ErrorRate > 0 {
			errorFactor = math.Max(0.1, 1.0-server.Metrics.ErrorRate*2)
		}
		
		// Factor de response time
		responseFactor := 1.0
		if server.Metrics.P95ResponseTime > 0 {
			baseline := 100 * time.Millisecond
			if server.Metrics.P95ResponseTime > baseline {
				responseFactor = math.Max(0.1, float64(baseline)/float64(server.Metrics.P95ResponseTime))
			} else {
				responseFactor = 1.2 // Bonus para servidores rápidos
			}
		}
		
		// Factor de conexiones activas
		connFactor := 1.0
		activeConns := atomic.LoadInt64(&server.ConnectionPool.ActiveConns)
		if activeConns > 0 {
			connFactor = math.Max(0.1, 1.0-float64(activeConns)/float64(server.ConnectionPool.MaxConnections))
		}
		
		// Factor de health state
		healthFactor := 1.0
		switch server.HealthState {
		case Healthy:
			healthFactor = 1.0
		case Degraded:
			healthFactor = 0.7
		case Recovering:
			healthFactor = 0.5
		case Unhealthy:
			healthFactor = 0.1
		}
		
		// Calcular peso efectivo
		server.EffectiveWeight = baseWeight * errorFactor * responseFactor * connFactor * healthFactor
		server.EffectiveWeight = math.Max(0.1, server.EffectiveWeight) // Mínimo peso
	}
}

// Least Connections con predicción de carga
type LeastConnections struct{}

func (lc *LeastConnections) SelectServer(servers []*ServerState, clientIP string) *ServerState {
	if len(servers) == 0 {
		return nil
	}

	var selected *ServerState
	minScore := math.MaxFloat64

	for _, server := range servers {
		activeConns := atomic.LoadInt64(&server.ConnectionPool.ActiveConns)
		
		// Score = conexiones_activas / peso_efectivo + factor_latencia
		score := float64(activeConns) / server.EffectiveWeight
		
		// Penalizar por alta latencia
		if server.Metrics.P95ResponseTime > 0 {
			latencyPenalty := float64(server.Metrics.P95ResponseTime) / float64(100*time.Millisecond)
			score += latencyPenalty * 0.3
		}
		
		// Penalizar por error rate
		score += server.Metrics.ErrorRate * 10
		
		if score < minScore {
			minScore = score
			selected = server
		}
	}

	return selected
}

func (lc *LeastConnections) UpdateWeights(servers []*ServerState) {
	// Weights updated by adaptive controller
}

// Least Response Time con predicción exponencial
type LeastResponseTime struct{}

func (lrt *LeastResponseTime) SelectServer(servers []*ServerState, clientIP string) *ServerState {
	if len(servers) == 0 {
		return nil
	}

	var selected *ServerState
	minPredictedTime := time.Duration(math.MaxInt64)

	for _, server := range servers {
		// Predicción de tiempo de respuesta basada en carga actual
		baseTime := server.Metrics.P95ResponseTime
		if baseTime == 0 {
			baseTime = 50 * time.Millisecond // Default optimista
		}
		
		// Factor de carga actual
		activeConns := atomic.LoadInt64(&server.ConnectionPool.ActiveConns)
		loadFactor := 1.0 + float64(activeConns)*0.1
		
		// Factor de error rate
		errorFactor := 1.0 + server.Metrics.ErrorRate*2
		
		predictedTime := time.Duration(float64(baseTime) * loadFactor * errorFactor)
		
		if predictedTime < minPredictedTime {
			minPredictedTime = predictedTime
			selected = server
		}
	}

	return selected
}

func (lrt *LeastResponseTime) UpdateWeights(servers []*ServerState) {}

// Consistent Hash con virtual nodes y failover
type ConsistentHash struct {
	ring *ConsistentHashRing
}

func (ch *ConsistentHash) SelectServer(servers []*ServerState, clientIP string) *ServerState {
	if len(servers) == 0 {
		return nil
	}

	// Actualizar ring si es necesario
	ch.ring.UpdateServers(servers)
	
	// Obtener servidor primario
	primary := ch.ring.GetServer(clientIP)
	if primary != nil {
		// Verificar si el servidor primario está disponible
		for _, server := range servers {
			if server.Server.URL == primary.Server.URL {
				// Verificar health y circuit breaker
				if server.HealthState != Unhealthy && server.CircuitBreaker.State != CircuitOpen {
					return server
				}
				break
			}
		}
	}
	
	// Failover: usar least connections
	lc := &LeastConnections{}
	return lc.SelectServer(servers, clientIP)
}

func (ch *ConsistentHash) UpdateWeights(servers []*ServerState) {}

// Power of Two Choices (algoritmo de Google)
type PowerOfTwoChoices struct{}

func (p2c *PowerOfTwoChoices) SelectServer(servers []*ServerState, clientIP string) *ServerState {
	if len(servers) == 0 {
		return nil
	}
	
	if len(servers) == 1 {
		return servers[0]
	}

	// Seleccionar dos servidores aleatorios
	idx1 := rand.Intn(len(servers))
	idx2 := rand.Intn(len(servers))
	for idx2 == idx1 {
		idx2 = rand.Intn(len(servers))
	}

	server1 := servers[idx1]
	server2 := servers[idx2]

	// Calcular score para cada servidor
	score1 := p2c.calculateScore(server1)
	score2 := p2c.calculateScore(server2)

	if score1 <= score2 {
		return server1
	}
	return server2
}

func (p2c *PowerOfTwoChoices) calculateScore(server *ServerState) float64 {
	activeConns := atomic.LoadInt64(&server.ConnectionPool.ActiveConns)
	
	// Score = conexiones / peso + latencia_normalizada + error_rate
	score := float64(activeConns) / server.EffectiveWeight
	
	if server.Metrics.P95ResponseTime > 0 {
		score += float64(server.Metrics.P95ResponseTime) / float64(100*time.Millisecond)
	}
	
	score += server.Metrics.ErrorRate * 5
	
	return score
}

func (p2c *PowerOfTwoChoices) UpdateWeights(servers []*ServerState) {}

// Weighted Fair Queue con bandwidth allocation
type WeightedFairQueue struct {
	virtualTime map[string]float64
	lastUpdate  time.Time
}

func (wfq *WeightedFairQueue) SelectServer(servers []*ServerState, clientIP string) *ServerState {
	if len(servers) == 0 {
		return nil
	}

	if wfq.virtualTime == nil {
		wfq.virtualTime = make(map[string]float64)
	}

	now := time.Now()
	if now.Sub(wfq.lastUpdate) > time.Second {
		wfq.updateVirtualTimes(servers)
		wfq.lastUpdate = now
	}

	// Seleccionar servidor con menor virtual time
	var selected *ServerState
	minVirtualTime := math.MaxFloat64

	for _, server := range servers {
		vt, exists := wfq.virtualTime[server.Server.URL]
		if !exists {
			vt = 0
			wfq.virtualTime[server.Server.URL] = vt
		}

		if vt < minVirtualTime {
			minVirtualTime = vt
			selected = server
		}
	}

	// Actualizar virtual time del servidor seleccionado
	if selected != nil {
		packetSize := 1.0 / selected.EffectiveWeight
		wfq.virtualTime[selected.Server.URL] += packetSize
	}

	return selected
}

func (wfq *WeightedFairQueue) updateVirtualTimes(servers []*ServerState) {
	// Normalizar virtual times para evitar overflow
	minVT := math.MaxFloat64
	for _, vt := range wfq.virtualTime {
		if vt < minVT {
			minVT = vt
		}
	}

	if minVT > 1000 {
		for url := range wfq.virtualTime {
			wfq.virtualTime[url] -= minVT
		}
	}
}

func (wfq *WeightedFairQueue) UpdateWeights(servers []*ServerState) {}

// Consistent Hash Ring implementation
type ConsistentHashRing struct {
	ring         map[uint32]string
	sortedHashes []uint32
	virtualNodes int
	servers      map[string]*ServerState
}

func NewConsistentHashRing(virtualNodes int) *ConsistentHashRing {
	return &ConsistentHashRing{
		ring:         make(map[uint32]string),
		virtualNodes: virtualNodes,
		servers:      make(map[string]*ServerState),
	}
}

func (chr *ConsistentHashRing) UpdateServers(servers []*ServerState) {
	// Limpiar ring
	chr.ring = make(map[uint32]string)
	chr.servers = make(map[string]*ServerState)
	chr.sortedHashes = nil

	// Agregar servidores con virtual nodes
	for _, server := range servers {
		chr.servers[server.Server.URL] = server
		
		for i := 0; i < chr.virtualNodes; i++ {
			virtualKey := fmt.Sprintf("%s:%d", server.Server.URL, i)
			hash := chr.hash(virtualKey)
			chr.ring[hash] = server.Server.URL
			chr.sortedHashes = append(chr.sortedHashes, hash)
		}
	}

	sort.Slice(chr.sortedHashes, func(i, j int) bool {
		return chr.sortedHashes[i] < chr.sortedHashes[j]
	})
}

func (chr *ConsistentHashRing) GetServer(key string) *ServerState {
	if len(chr.sortedHashes) == 0 {
		return nil
	}

	hash := chr.hash(key)
	
	// Buscar el primer hash mayor o igual
	idx := sort.Search(len(chr.sortedHashes), func(i int) bool {
		return chr.sortedHashes[i] >= hash
	})

	// Si no encontramos, usar el primero (wrap around)
	if idx == len(chr.sortedHashes) {
		idx = 0
	}

	serverURL := chr.ring[chr.sortedHashes[idx]]
	return chr.servers[serverURL]
}

func (chr *ConsistentHashRing) hash(key string) uint32 {
	h := md5.Sum([]byte(key))
	return uint32(h[0])<<24 | uint32(h[1])<<16 | uint32(h[2])<<8 | uint32(h[3])
}

// Ring Buffer para métricas de response time
func NewRingBuffer(size int) *RingBuffer {
	return &RingBuffer{
		buffer: make([]time.Duration, size),
		size:   size,
	}
}

func (rb *RingBuffer) Add(value time.Duration) {
	rb.mu.Lock()
	defer rb.mu.Unlock()
	
	rb.buffer[rb.index] = value
	rb.index = (rb.index + 1) % rb.size
	if rb.index == 0 {
		rb.full = true
	}
}

func (rb *RingBuffer) GetAll() []time.Duration {
	rb.mu.RLock()
	defer rb.mu.RUnlock()
	
	if !rb.full {
		result := make([]time.Duration, rb.index)
		copy(result, rb.buffer[:rb.index])
		return result
	}
	
	result := make([]time.Duration, rb.size)
	copy(result, rb.buffer[rb.index:])
	copy(result[rb.size-rb.index:], rb.buffer[:rb.index])
	return result
}