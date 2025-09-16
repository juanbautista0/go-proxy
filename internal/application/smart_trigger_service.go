package application

import (
	"fmt"
	"math"
	"time"

	"github.com/juanbautista0/go-proxy/internal/domain"
)

// SmartTriggerService - Sistema de triggers inteligente basado en scoring compuesto
type SmartTriggerService struct {
	config         *domain.Config
	metrics        *domain.TrafficMetrics
	executor       domain.ActionExecutor
	proxyService   domain.ProxyService
	
	// Configuración de scoring
	weights        ScoreWeights
	thresholds     ScoreThresholds
	
	// Ventanas de tiempo para estabilidad
	shortWindow    *TimeWindow // 30s - Detección rápida
	longWindow     *TimeWindow // 5min - Confirmación
	
	// Control de cooldown
	lastTrigger    time.Time
	lastAction     string
	cooldownPeriod time.Duration
	
	// Estado interno
	lastScore      float64
	lastEvaluation time.Time
}

// ScoreWeights - Pesos para el cálculo del score compuesto
type ScoreWeights struct {
	RPS         float64 // Requests per second
	Latency     float64 // Tiempo de respuesta promedio
	ErrorRate   float64 // Tasa de errores
	Connections float64 // Conexiones activas
}

// ScoreThresholds - Umbrales para decisiones de escalado
type ScoreThresholds struct {
	ScaleUp   float64 // Score para escalar hacia arriba
	ScaleDown float64 // Score para escalar hacia abajo
}

// TriggerScore - Resultado del cálculo de scoring
type TriggerScore struct {
	TotalScore    float64
	RPSScore      float64
	LatencyScore  float64
	ErrorScore    float64
	ConnScore     float64
	Timestamp     time.Time
	ShouldScale   string // "up", "down", "none"
	Confidence    float64
}

// TimeWindow - Buffer circular para ventanas de tiempo
type TimeWindow struct {
	scores    []float64
	timestamps []time.Time
	size      int
	index     int
	full      bool
	duration  time.Duration
}

// TriggerDecision - Decisión final de trigger con contexto temporal
type TriggerDecision struct {
	Action        string    // "scale_up", "scale_down", "none"
	Score         float64   // Score actual
	Trend         string    // "increasing", "stable", "decreasing"
	Confidence    float64   // 0.0 - 1.0
	Stability     float64   // Qué tan estable es la tendencia
	Reason        string    // Razón de la decisión
	CanTrigger    bool      // Si puede disparar (cooldown)
	Timestamp     time.Time
}

func NewSmartTriggerService(executor domain.ActionExecutor, proxyService domain.ProxyService) *SmartTriggerService {
	return &SmartTriggerService{
		executor:     executor,
		proxyService: proxyService,
		
		// Pesos balanceados basados en impacto en performance
		weights: ScoreWeights{
			RPS:         0.30, // 30% - Volumen de tráfico
			Latency:     0.25, // 25% - Performance percibida
			ErrorRate:   0.25, // 25% - Calidad del servicio
			Connections: 0.20, // 20% - Saturación de recursos
		},
		
		// Thresholds por defecto (serán configurados desde YAML)
		thresholds: ScoreThresholds{
			ScaleUp:   0.75,
			ScaleDown: 0.25,
		},
		
		// Ventanas por defecto (serán reconfiguradas)
		shortWindow:    NewTimeWindow(30*time.Second, 6),
		longWindow:     NewTimeWindow(5*time.Minute, 10),
		cooldownPeriod: 3 * time.Minute,
	}
}

// SetConfig - Configura el SmartTrigger con parámetros externos
func (s *SmartTriggerService) SetConfig(config *domain.Config) {
	s.config = config
}

// GetLastDecision - Obtiene la última decisión para debugging
func (s *SmartTriggerService) GetLastDecision() *TriggerDecision {
	return s.EvaluateTrigger()
}

// CalculateScore - Calcula el score compuesto basado en métricas actuales
func (s *SmartTriggerService) CalculateScore() *TriggerScore {
	metrics := s.proxyService.GetMetrics()
	serverStats := s.proxyService.GetServerStats()
	
	now := time.Now()
	
	// Calcular métricas agregadas
	totalRequests := int64(0)
	totalFailures := int64(0)
	totalConnections := int64(0)
	avgLatency := time.Duration(0)
	
	for _, server := range serverStats {
		totalRequests += server.TotalRequests
		totalFailures += server.FailedRequests
		totalConnections += server.CurrentConns
		avgLatency += server.ResponseTime
	}
	
	if len(serverStats) > 0 {
		avgLatency = avgLatency / time.Duration(len(serverStats))
	}
	
	// Calcular scores individuales (0.0 - 1.0)
	rpsScore := s.calculateRPSScore(float64(metrics.RequestsPerSecond))
	latencyScore := s.calculateLatencyScore(avgLatency)
	errorScore := s.calculateErrorScore(totalRequests, totalFailures)
	connScore := s.calculateConnectionScore(totalConnections, len(serverStats))
	
	// Score compuesto ponderado
	totalScore := (rpsScore * s.weights.RPS) +
		(latencyScore * s.weights.Latency) +
		(errorScore * s.weights.ErrorRate) +
		(connScore * s.weights.Connections)
	
	// Determinar acción de escalado
	shouldScale := "none"
	confidence := 0.0
	
	if totalScore >= s.thresholds.ScaleUp {
		shouldScale = "up"
		confidence = (totalScore - s.thresholds.ScaleUp) / (1.0 - s.thresholds.ScaleUp)
	} else if totalScore <= s.thresholds.ScaleDown {
		shouldScale = "down"
		confidence = (s.thresholds.ScaleDown - totalScore) / s.thresholds.ScaleDown
	}
	
	return &TriggerScore{
		TotalScore:   totalScore,
		RPSScore:     rpsScore,
		LatencyScore: latencyScore,
		ErrorScore:   errorScore,
		ConnScore:    connScore,
		Timestamp:    now,
		ShouldScale:  shouldScale,
		Confidence:   confidence,
	}
}

// calculateRPSScore - Score basado en requests per second (0.0 - 1.0)
// Basado en capacidades reales de servidores web
func (s *SmartTriggerService) calculateRPSScore(rps float64) float64 {
	// Rangos basados en benchmarks de servidores Go:
	// Go HTTP server: ~10K RPS en hardware moderno
	// Aplicación típica: 100-1000 RPS por instancia
	// Microservicio: 50-500 RPS típico
	// API REST: 100-2000 RPS según complejidad
	
	switch {
	case rps <= 50: // Tráfico bajo
		return 0.1
	case rps <= 100: // Tráfico normal
		return 0.3
	case rps <= 200: // Tráfico moderado
		return 0.5
	case rps <= 500: // Tráfico alto
		return 0.7
	case rps <= 1000: // Tráfico muy alto
		return 0.9
	default: // >1000 RPS = Saturación
		return 1.0
	}
}

// calculateLatencyScore - Score basado en latencia promedio (0.0 - 1.0)
// Basado en benchmarks de la industria y estándares de UX
func (s *SmartTriggerService) calculateLatencyScore(latency time.Duration) float64 {
	latencyMs := float64(latency.Nanoseconds()) / 1e6
	
	// Rangos basados en estándares de la industria:
	// Google: <100ms excelente, <200ms bueno, >500ms malo
	// AWS ALB: <100ms típico, >300ms requiere acción
	// HTTP/2 spec: <100ms para interactividad
	
	switch {
	case latencyMs <= 100: // Excelente (Google/AWS standard)
		return 0.0
	case latencyMs <= 200: // Bueno (aceptable para web)
		return 0.2
	case latencyMs <= 300: // Moderado (límite AWS)
		return 0.4
	case latencyMs <= 500: // Malo (límite UX crítico)
		return 0.7
	case latencyMs <= 1000: // Crítico (timeout típico)
		return 0.9
	default: // >1s = Inaceptable
		return 1.0
	}
}

// calculateErrorScore - Score basado en tasa de errores (0.0 - 1.0)
func (s *SmartTriggerService) calculateErrorScore(totalReqs, failedReqs int64) float64 {
	if totalReqs == 0 {
		return 0.0
	}
	
	errorRate := float64(failedReqs) / float64(totalReqs)
	
	// 0% errores = 0.0, 1% = 0.2, 5% = 0.6, 10%+ = 1.0
	if errorRate <= 0.01 { // 1%
		return errorRate * 20 // 0-0.2
	} else if errorRate <= 0.05 { // 5%
		return 0.2 + (errorRate-0.01)*10 // 0.2-0.6
	} else if errorRate >= 0.10 { // 10%+
		return 1.0
	}
	
	return 0.6 + (errorRate-0.05)*8 // 0.6-1.0
}

// calculateConnectionScore - Score basado en conexiones activas (0.0 - 1.0)
// Basado en capacidades típicas de servidores Go HTTP
func (s *SmartTriggerService) calculateConnectionScore(totalConns int64, serverCount int) float64 {
	if serverCount == 0 {
		return 1.0
	}
	
	avgConnsPerServer := float64(totalConns) / float64(serverCount)
	
	// Rangos basados en benchmarks Go HTTP server:
	// Go default: ~5000 conns/server teórico
	// Producción típica: 100-500 conns/server
	// Kubernetes: 100 conns/pod típico
	// NGINX: 1024 worker_connections default
	
	switch {
	case avgConnsPerServer <= 50: // Bajo uso
		return 0.0
	case avgConnsPerServer <= 100: // Uso normal (K8s típico)
		return 0.3
	case avgConnsPerServer <= 200: // Uso alto
		return 0.6
	case avgConnsPerServer <= 500: // Saturación
		return 0.8
	default: // >500 = Crítico
		return 1.0
	}
}

// NewTimeWindow - Crea una nueva ventana de tiempo
func NewTimeWindow(duration time.Duration, maxSamples int) *TimeWindow {
	return &TimeWindow{
		scores:    make([]float64, maxSamples),
		timestamps: make([]time.Time, maxSamples),
		size:      maxSamples,
		index:     0,
		full:      false,
		duration:  duration,
	}
}

// AddScore - Agrega un score a la ventana
func (tw *TimeWindow) AddScore(score float64, timestamp time.Time) {
	tw.scores[tw.index] = score
	tw.timestamps[tw.index] = timestamp
	tw.index = (tw.index + 1) % tw.size
	if tw.index == 0 {
		tw.full = true
	}
}

// GetAverage - Obtiene el promedio de scores en la ventana
func (tw *TimeWindow) GetAverage() float64 {
	count := tw.size
	if !tw.full {
		count = tw.index
	}
	if count == 0 {
		return 0.0
	}
	
	sum := 0.0
	for i := 0; i < count; i++ {
		sum += tw.scores[i]
	}
	return sum / float64(count)
}

// GetTrend - Calcula la tendencia (slope) de los scores
func (tw *TimeWindow) GetTrend() (string, float64) {
	count := tw.size
	if !tw.full {
		count = tw.index
	}
	if count < 3 {
		return "stable", 0.0
	}
	
	// Cálculo de regresión lineal simple
	n := float64(count)
	sumX, sumY, sumXY, sumX2 := 0.0, 0.0, 0.0, 0.0
	
	for i := 0; i < count; i++ {
		x := float64(i)
		y := tw.scores[i]
		sumX += x
		sumY += y
		sumXY += x * y
		sumX2 += x * x
	}
	
	// Slope = (n*sumXY - sumX*sumY) / (n*sumX2 - sumX*sumX)
	slope := (n*sumXY - sumX*sumY) / (n*sumX2 - sumX*sumX)
	
	// Clasificar tendencia
	if slope > 0.02 {
		return "increasing", slope
	} else if slope < -0.02 {
		return "decreasing", slope
	}
	return "stable", slope
}

// GetStability - Mide qué tan estables son los scores
func (tw *TimeWindow) GetStability() float64 {
	avg := tw.GetAverage()
	count := tw.size
	if !tw.full {
		count = tw.index
	}
	if count < 2 {
		return 0.0
	}
	
	// Calcular varianza
	variance := 0.0
	for i := 0; i < count; i++ {
		diff := tw.scores[i] - avg
		variance += diff * diff
	}
	variance /= float64(count)
	
	// Estabilidad = 1 - varianza normalizada
	return math.Max(0.0, 1.0-variance*4)
}

// EvaluateTrigger - Evaluación inteligente con ventanas de tiempo
func (s *SmartTriggerService) EvaluateTrigger() *TriggerDecision {
	now := time.Now()
	
	// Calcular score actual
	currentScore := s.CalculateScore()
	
	// Agregar a ventanas de tiempo
	s.shortWindow.AddScore(currentScore.TotalScore, now)
	s.longWindow.AddScore(currentScore.TotalScore, now)
	
	// Obtener métricas temporales
	shortAvg := s.shortWindow.GetAverage()
	longAvg := s.longWindow.GetAverage()
	trend, _ := s.shortWindow.GetTrend()
	stability := s.shortWindow.GetStability()
	
	// Verificar cooldown
	canTrigger := now.Sub(s.lastTrigger) > s.cooldownPeriod
	
	// Lógica de decisión inteligente
	decision := &TriggerDecision{
		Action:     "none",
		Score:      currentScore.TotalScore,
		Trend:      trend,
		Confidence: 0.0,
		Stability:  stability,
		CanTrigger: canTrigger,
		Timestamp:  now,
	}
	
	// Solo considerar acción si hay suficiente estabilidad y está fuera de cooldown
	if stability > 0.6 && canTrigger {
		// Scale Up: Score alto Y tendencia creciente Y confirmación
		if shortAvg >= s.thresholds.ScaleUp && longAvg > 0.5 && trend == "increasing" {
			decision.Action = "scale_up"
			decision.Confidence = math.Min(1.0, (shortAvg-s.thresholds.ScaleUp)*2 + stability)
			decision.Reason = fmt.Sprintf("High load: avg=%.2f, trend=%s, stability=%.2f", shortAvg, trend, stability)
		}
		// Scale Down: Score bajo Y tendencia decreciente Y confirmación sostenida
		if shortAvg <= s.thresholds.ScaleDown && longAvg < 0.4 && trend == "decreasing" {
			decision.Action = "scale_down"
			decision.Confidence = math.Min(1.0, (s.thresholds.ScaleDown-shortAvg)*2 + stability)
			decision.Reason = fmt.Sprintf("Low load: avg=%.2f, trend=%s, stability=%.2f", shortAvg, trend, stability)
		}
	} else {
		if !canTrigger {
			decision.Reason = fmt.Sprintf("Cooldown active (%.0fs remaining)", s.cooldownPeriod.Seconds()-now.Sub(s.lastTrigger).Seconds())
		} else {
			decision.Reason = fmt.Sprintf("Insufficient stability: %.2f < 0.6", stability)
		}
	}
	
	return decision
}