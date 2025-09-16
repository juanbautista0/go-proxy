package infrastructure

import (
	"math"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/juanbautista0/go-proxy/internal/domain"
)

type EnterpriseBalancer struct {
	mu                    sync.RWMutex
	servers               map[string]*ServerState
	algorithms            map[string]Algorithm
	currentAlgorithm      string
	adaptiveController    *AdaptiveController
	consistentHashRing    *ConsistentHashRing
	requestCounter        int64
	performanceMonitor    *PerformanceMonitor
}

type ServerState struct {
	Server           *domain.Server
	Metrics          *ServerMetrics
	HealthState      HealthState
	CircuitBreaker   *CircuitBreaker
	ConnectionPool   *ConnectionPool
	LastHealthCheck  time.Time
	ConsecutiveFails int
	Weight           float64
	EffectiveWeight  float64
	CurrentWeight    float64
}

type ServerMetrics struct {
	RequestCount     int64
	SuccessCount     int64
	FailureCount     int64
	ResponseTimes    *RingBuffer
	ActiveConns      int64
	TotalLatency     int64
	P95ResponseTime  time.Duration
	P99ResponseTime  time.Duration
	ThroughputRPS    float64
	ErrorRate        float64
	LastUpdate       time.Time
}

type HealthState int

const (
	Healthy HealthState = iota
	Degraded
	Unhealthy
	Recovering
)

type CircuitState int

const (
	CircuitClosed CircuitState = iota
	CircuitOpen
	CircuitHalfOpen
)

type CircuitBreaker struct {
	State            CircuitState
	FailureCount     int64
	SuccessCount     int64
	LastFailureTime  time.Time
	NextRetryTime    time.Time
	FailureThreshold int
	RecoveryTimeout  time.Duration
	HalfOpenRequests int
}

type ConnectionPool struct {
	MaxConnections int
	ActiveConns    int64
	WaitingConns   int64
}

type Algorithm interface {
	SelectServer(servers []*ServerState, clientIP string) *ServerState
	UpdateWeights(servers []*ServerState)
}

type AdaptiveController struct {
	mu                 sync.RWMutex
	performanceHistory map[string]*PerformanceWindow
	algorithmScores    map[string]float64
	switchThreshold    float64
	evaluationWindow   time.Duration
	lastSwitch         time.Time
}

type PerformanceWindow struct {
	samples    []float64
	timestamps []time.Time
	maxSize    int
}



type PerformanceMonitor struct {
	globalMetrics *GlobalMetrics
	alertThresholds *AlertThresholds
}

type GlobalMetrics struct {
	TotalRequests    int64
	SuccessfulReqs   int64
	FailedReqs       int64
	AvgResponseTime  time.Duration
	P95ResponseTime  time.Duration
	P99ResponseTime  time.Duration
	ErrorRate        float64
	ThroughputRPS    float64
}

type AlertThresholds struct {
	MaxErrorRate     float64
	MaxResponseTime  time.Duration
	MinThroughput    float64
}

type RingBuffer struct {
	buffer []time.Duration
	size   int
	index  int
	full   bool
	mu     sync.RWMutex
}

func NewEnterpriseBalancer() *EnterpriseBalancer {
	eb := &EnterpriseBalancer{
		servers:            make(map[string]*ServerState),
		algorithms:         make(map[string]Algorithm),
		currentAlgorithm:   "adaptive_weighted",
		consistentHashRing: NewConsistentHashRing(150),
		performanceMonitor: &PerformanceMonitor{
			globalMetrics: &GlobalMetrics{},
			alertThresholds: &AlertThresholds{
				MaxErrorRate:    0.05,
				MaxResponseTime: 500 * time.Millisecond,
				MinThroughput:   100,
			},
		},
		adaptiveController: &AdaptiveController{
			performanceHistory: make(map[string]*PerformanceWindow),
			algorithmScores:    make(map[string]float64),
			switchThreshold:    0.15,
			evaluationWindow:   30 * time.Second,
		},
	}

	// Registrar algoritmos avanzados
	eb.algorithms["adaptive_weighted"] = &AdaptiveWeightedRoundRobin{}
	eb.algorithms["least_connections"] = &LeastConnections{}
	eb.algorithms["response_time"] = &LeastResponseTime{}
	eb.algorithms["consistent_hash"] = &ConsistentHash{ring: eb.consistentHashRing}
	eb.algorithms["power_of_two"] = &PowerOfTwoChoices{}
	eb.algorithms["weighted_fair_queue"] = &WeightedFairQueue{}

	return eb
}

func (eb *EnterpriseBalancer) SelectServer(backend *domain.Backend, clientIP string) *domain.Server {
	// Inicializar servidores si es necesario (con write lock)
	eb.mu.Lock()
	eb.initializeServers(backend.Servers)
	eb.mu.Unlock()

	eb.mu.RLock()
	defer eb.mu.RUnlock()

	// Obtener servidores disponibles
	availableServers := eb.getAvailableServers()
	if len(availableServers) == 0 {
		return nil
	}

	// Seleccionar algoritmo adaptativo
	algorithm := eb.selectOptimalAlgorithm()
	
	// Seleccionar servidor usando el algoritmo
	selectedState := algorithm.SelectServer(availableServers, clientIP)
	if selectedState == nil {
		return nil
	}

	// Actualizar métricas de selección
	atomic.AddInt64(&selectedState.Metrics.RequestCount, 1)
	atomic.AddInt64(&selectedState.ConnectionPool.ActiveConns, 1)

	return selectedState.Server
}

func (eb *EnterpriseBalancer) UpdateServers(servers []domain.Server) {
	// Crear mapa de servidores actuales
	currentServers := make(map[string]bool)
	for i := range servers {
		server := &servers[i]
		currentServers[server.URL] = true
		
		if _, exists := eb.servers[server.URL]; !exists {
			// Agregar servidor nuevo
			eb.servers[server.URL] = &ServerState{
				Server: server,
				Metrics: &ServerMetrics{
					ResponseTimes: NewRingBuffer(1000),
					LastUpdate:    time.Now(),
				},
				HealthState: Healthy,
				CircuitBreaker: &CircuitBreaker{
					State:            CircuitClosed,
					FailureThreshold: 10,
					RecoveryTimeout:  30 * time.Second,
				},
				ConnectionPool: &ConnectionPool{
					MaxConnections: 1000,
				},
				Weight:          float64(server.Weight),
				EffectiveWeight: float64(server.Weight),
				CurrentWeight:   0,
			}
		} else {
			// Actualizar servidor existente
			eb.servers[server.URL].Server = server
			eb.servers[server.URL].Weight = float64(server.Weight)
			eb.servers[server.URL].EffectiveWeight = float64(server.Weight)
		}
	}
	
	// Eliminar servidores que ya no existen
	for url := range eb.servers {
		if !currentServers[url] {
			delete(eb.servers, url)
		}
	}
}

func (eb *EnterpriseBalancer) initializeServers(servers []domain.Server) {
	eb.UpdateServers(servers)
}

func (eb *EnterpriseBalancer) getAvailableServers() []*ServerState {
	var available []*ServerState
	now := time.Now()

	for _, state := range eb.servers {
		// Circuit breaker logic
		if state.CircuitBreaker.State == CircuitOpen {
			if now.After(state.CircuitBreaker.NextRetryTime) {
				state.CircuitBreaker.State = CircuitHalfOpen
				state.CircuitBreaker.HalfOpenRequests = 0
			} else {
				continue
			}
		}

		// Health check
		if state.HealthState == Unhealthy && now.Sub(state.LastHealthCheck) < 10*time.Second {
			continue
		}

		// Connection limit
		if state.ConnectionPool.ActiveConns >= int64(state.ConnectionPool.MaxConnections) {
			continue
		}

		available = append(available, state)
	}

	return available
}

func (eb *EnterpriseBalancer) selectOptimalAlgorithm() Algorithm {
	// Evaluación adaptativa de algoritmos
	eb.adaptiveController.mu.RLock()
	lastSwitch := eb.adaptiveController.lastSwitch
	eb.adaptiveController.mu.RUnlock()
	
	if time.Since(lastSwitch) > eb.adaptiveController.evaluationWindow {
		bestAlgorithm := eb.evaluateAlgorithms()
		if bestAlgorithm != eb.currentAlgorithm {
			eb.adaptiveController.mu.RLock()
			currentScore := eb.adaptiveController.algorithmScores[eb.currentAlgorithm]
			bestScore := eb.adaptiveController.algorithmScores[bestAlgorithm]
			eb.adaptiveController.mu.RUnlock()
			
			if bestScore-currentScore > eb.adaptiveController.switchThreshold {
				eb.currentAlgorithm = bestAlgorithm
				eb.adaptiveController.mu.Lock()
				eb.adaptiveController.lastSwitch = time.Now()
				eb.adaptiveController.mu.Unlock()
			}
		}
	}

	return eb.algorithms[eb.currentAlgorithm]
}

func (eb *EnterpriseBalancer) evaluateAlgorithms() string {
	bestAlgorithm := eb.currentAlgorithm
	bestScore := 0.0

	eb.adaptiveController.mu.Lock()
	defer eb.adaptiveController.mu.Unlock()
	
	for name, _ := range eb.algorithms {
		score := eb.calculateAlgorithmScore(name)
		eb.adaptiveController.algorithmScores[name] = score
		
		if score > bestScore {
			bestScore = score
			bestAlgorithm = name
		}
	}

	return bestAlgorithm
}

func (eb *EnterpriseBalancer) calculateAlgorithmScore(algorithmName string) float64 {
	// Score basado en múltiples métricas
	errorRateScore := (1.0 - eb.performanceMonitor.globalMetrics.ErrorRate) * 0.3
	
	responseTimeScore := 0.0
	if eb.performanceMonitor.globalMetrics.AvgResponseTime > 0 {
		responseTimeScore = math.Max(0, 1.0-float64(eb.performanceMonitor.globalMetrics.AvgResponseTime)/float64(time.Second)) * 0.3
	}
	
	throughputScore := math.Min(1.0, eb.performanceMonitor.globalMetrics.ThroughputRPS/1000.0) * 0.2
	
	balanceScore := eb.calculateLoadBalanceScore() * 0.2
	
	return errorRateScore + responseTimeScore + throughputScore + balanceScore
}

func (eb *EnterpriseBalancer) calculateLoadBalanceScore() float64 {
	if len(eb.servers) < 2 {
		return 1.0
	}

	var loads []float64
	for _, state := range eb.servers {
		load := float64(atomic.LoadInt64(&state.Metrics.RequestCount))
		loads = append(loads, load)
	}

	// Calcular coeficiente de variación
	mean := 0.0
	for _, load := range loads {
		mean += load
	}
	mean /= float64(len(loads))

	if mean == 0 {
		return 1.0
	}

	variance := 0.0
	for _, load := range loads {
		variance += (load - mean) * (load - mean)
	}
	variance /= float64(len(loads))

	cv := math.Sqrt(variance) / mean
	return math.Max(0, 1.0-cv)
}

func (eb *EnterpriseBalancer) UpdateStats(server *domain.Server, responseTime time.Duration, success bool) {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	state, exists := eb.servers[server.URL]
	if !exists {
		return
	}

	// Actualizar métricas del servidor
	state.Metrics.ResponseTimes.Add(responseTime)
	atomic.AddInt64(&state.Metrics.TotalLatency, int64(responseTime))
	atomic.AddInt64(&state.ConnectionPool.ActiveConns, -1)

	if success {
		atomic.AddInt64(&state.Metrics.SuccessCount, 1)
		state.CircuitBreaker.SuccessCount++
		
		// Reset circuit breaker si está en half-open
		if state.CircuitBreaker.State == CircuitHalfOpen {
			state.CircuitBreaker.HalfOpenRequests++
			if state.CircuitBreaker.HalfOpenRequests >= 5 {
				state.CircuitBreaker.State = CircuitClosed
				state.CircuitBreaker.FailureCount = 0
			}
		}
		
		state.ConsecutiveFails = 0
		if state.HealthState == Degraded || state.HealthState == Recovering {
			state.HealthState = Healthy
		}
	} else {
		atomic.AddInt64(&state.Metrics.FailureCount, 1)
		state.CircuitBreaker.FailureCount++
		state.CircuitBreaker.LastFailureTime = time.Now()
		state.ConsecutiveFails++

		// Circuit breaker logic
		if state.CircuitBreaker.FailureCount >= int64(state.CircuitBreaker.FailureThreshold) {
			state.CircuitBreaker.State = CircuitOpen
			state.CircuitBreaker.NextRetryTime = time.Now().Add(state.CircuitBreaker.RecoveryTimeout)
		}

		// Health state degradation
		if state.ConsecutiveFails >= 3 {
			state.HealthState = Degraded
		}
		if state.ConsecutiveFails >= 10 {
			state.HealthState = Unhealthy
		}
	}

	// Actualizar métricas calculadas
	eb.updateCalculatedMetrics(state)
	eb.updateGlobalMetrics()
}

func (eb *EnterpriseBalancer) updateCalculatedMetrics(state *ServerState) {
	totalReqs := atomic.LoadInt64(&state.Metrics.RequestCount)
	if totalReqs > 0 {
		successReqs := atomic.LoadInt64(&state.Metrics.SuccessCount)
		state.Metrics.ErrorRate = 1.0 - (float64(successReqs) / float64(totalReqs))
		
		// Calcular percentiles
		times := state.Metrics.ResponseTimes.GetAll()
		if len(times) > 0 {
			sort.Slice(times, func(i, j int) bool { return times[i] < times[j] })
			
			p95Index := int(float64(len(times)) * 0.95)
			p99Index := int(float64(len(times)) * 0.99)
			
			if p95Index < len(times) {
				state.Metrics.P95ResponseTime = times[p95Index]
			}
			if p99Index < len(times) {
				state.Metrics.P99ResponseTime = times[p99Index]
			}
		}
	}
	
	state.Metrics.LastUpdate = time.Now()
}

func (eb *EnterpriseBalancer) updateGlobalMetrics() {
	var totalReqs, successReqs, failedReqs int64
	var totalLatency int64
	
	for _, state := range eb.servers {
		totalReqs += atomic.LoadInt64(&state.Metrics.RequestCount)
		successReqs += atomic.LoadInt64(&state.Metrics.SuccessCount)
		failedReqs += atomic.LoadInt64(&state.Metrics.FailureCount)
		totalLatency += atomic.LoadInt64(&state.Metrics.TotalLatency)
	}
	
	eb.performanceMonitor.globalMetrics.TotalRequests = totalReqs
	eb.performanceMonitor.globalMetrics.SuccessfulReqs = successReqs
	eb.performanceMonitor.globalMetrics.FailedReqs = failedReqs
	
	if totalReqs > 0 {
		eb.performanceMonitor.globalMetrics.ErrorRate = float64(failedReqs) / float64(totalReqs)
		eb.performanceMonitor.globalMetrics.AvgResponseTime = time.Duration(totalLatency / totalReqs)
	}
}

func (eb *EnterpriseBalancer) GetServerMetrics() map[string]*domain.Server {
	eb.mu.RLock()
	defer eb.mu.RUnlock()
	
	metrics := make(map[string]*domain.Server)
	for url, state := range eb.servers {
		// Crear una copia del servidor con métricas actualizadas
		server := &domain.Server{
			URL:            state.Server.URL,
			Weight:         state.Server.Weight,
			MaxConnections: state.Server.MaxConnections,
			Active:         state.Server.Active,
			Healthy:        state.HealthState == Healthy,
			CircuitOpen:    state.CircuitBreaker.State == CircuitOpen,
			TotalRequests:  atomic.LoadInt64(&state.Metrics.RequestCount),
			FailedRequests: atomic.LoadInt64(&state.Metrics.FailureCount),
			CurrentConns:   atomic.LoadInt64(&state.ConnectionPool.ActiveConns),
			ResponseTime:   state.Metrics.P95ResponseTime,
		}
		metrics[url] = server
	}
	return metrics
}