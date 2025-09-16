package application

import (
	"log"
	"time"

	"github.com/juanbautista0/go-proxy/internal/domain"
)

// HybridTriggerService - Wrapper que integra SmartTrigger con el sistema existente
type HybridTriggerService struct {
	smartTrigger *SmartTriggerService
	executor     domain.ActionExecutor
	config       *domain.Config
	stopCh       chan struct{}
	running      bool
}

func NewHybridTriggerService(smartTrigger *SmartTriggerService, executor domain.ActionExecutor) *HybridTriggerService {
	return &HybridTriggerService{
		smartTrigger: smartTrigger,
		executor:     executor,
	}
}

func (h *HybridTriggerService) Start(config *domain.Config, metrics *domain.TrafficMetrics) error {
	h.config = config
	h.stopCh = make(chan struct{})
	h.running = true

	// Configurar SmartTrigger con parámetros del config
	h.configureSmartTrigger(config)

	// Iniciar monitoreo inteligente
	go h.smartMonitorLoop()

	log.Printf("🧠 Smart Trigger Service started - Interval: %v, Cooldown: %v",
		config.Triggers.Smart.EvaluationInterval,
		config.Triggers.Smart.Cooldown)

	return nil
}

func (h *HybridTriggerService) Stop() error {
	if h.running {
		h.running = false
		close(h.stopCh)
		log.Println("🛑 Smart Trigger Service stopped")
	}
	return nil
}

// configureSmartTrigger - Configura el SmartTrigger con parámetros del YAML
func (h *HybridTriggerService) configureSmartTrigger(config *domain.Config) {
	smart := config.Triggers.Smart

	// Actualizar configuración del SmartTrigger
	h.smartTrigger.thresholds.ScaleUp = smart.ScaleUpScore
	h.smartTrigger.thresholds.ScaleDown = smart.ScaleDownScore
	h.smartTrigger.cooldownPeriod = smart.Cooldown

	// Recrear ventanas de tiempo con nueva configuración
	shortSamples := int(smart.ShortWindow.Seconds() / smart.EvaluationInterval.Seconds())
	longSamples := int(smart.LongWindow.Seconds() / (smart.EvaluationInterval.Seconds() * 6)) // 6x menos frecuente

	h.smartTrigger.shortWindow = NewTimeWindow(smart.ShortWindow, max(shortSamples, 3))
	h.smartTrigger.longWindow = NewTimeWindow(smart.LongWindow, max(longSamples, 3))

	log.Printf("📊 Smart Trigger configured - Short: %v (%d samples), Long: %v (%d samples)",
		smart.ShortWindow, shortSamples, smart.LongWindow, longSamples)
}

// smartMonitorLoop - Loop principal del monitoreo inteligente
func (h *HybridTriggerService) smartMonitorLoop() {
	ticker := time.NewTicker(h.config.Triggers.Smart.EvaluationInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			h.evaluateAndExecute()
		case <-h.stopCh:
			return
		}
	}
}

// evaluateAndExecute - Evalúa y ejecuta acciones basadas en SmartTrigger
func (h *HybridTriggerService) evaluateAndExecute() {
	// Obtener score detallado para debugging
	scoreDetail := h.smartTrigger.CalculateScore()
	decision := h.smartTrigger.EvaluateTrigger()

	// Log detallado de componentes del score
	log.Printf("📊 Score Components: RPS=%.3f, Latency=%.3f, Error=%.3f, Conn=%.3f, Total=%.3f",
		scoreDetail.RPSScore, scoreDetail.LatencyScore, scoreDetail.ErrorScore, scoreDetail.ConnScore, scoreDetail.TotalScore)

	// Log de decisión para debugging
	log.Printf("🔍 Smart Decision: Action=%s, Score=%.3f, Trend=%s, Stability=%.3f, Confidence=%.3f, CanTrigger=%v",
		decision.Action, decision.Score, decision.Trend, decision.Stability, decision.Confidence, decision.CanTrigger)

	// Log de thresholds para comparación
	log.Printf("⚖️  Thresholds: ScaleUp=%.3f, ScaleDown=%.3f, StabilityMin=0.6",
		h.smartTrigger.thresholds.ScaleUp, h.smartTrigger.thresholds.ScaleDown)

	// Ejecutar acción si es necesario
	if decision.Action != "none" && decision.CanTrigger {
		h.executeSmartAction(decision)
	} else {
		log.Printf("ℹ️  No action: %s", decision.Reason)
	}
}

// executeSmartAction - Ejecuta la acción determinada por el SmartTrigger
func (h *HybridTriggerService) executeSmartAction(decision *TriggerDecision) {
	var actionName string
	var emoji string

	switch decision.Action {
	case "scale_up":
		actionName = h.config.Triggers.Traffic.HighAction
		emoji = "🚀"
	case "scale_down":
		actionName = h.config.Triggers.Traffic.LowAction
		emoji = "📉"
	default:
		return
	}

	// Buscar configuración de la acción
	actionConfig, exists := h.config.Actions[actionName]
	if !exists {
		log.Printf("❌ Action '%s' not found in config", actionName)
		return
	}

	// Ejecutar acción
	err := h.executor.Execute(actionName, actionConfig)
	if err != nil {
		log.Printf("❌ Failed to execute %s: %v", actionName, err)
		return
	}

	// Actualizar estado del SmartTrigger
	h.smartTrigger.lastTrigger = decision.Timestamp
	h.smartTrigger.lastAction = decision.Action

	// Log exitoso
	log.Printf("%s SMART TRIGGER: %s executed (Score: %.3f, Confidence: %.3f, Reason: %s)",
		emoji, actionName, decision.Score, decision.Confidence, decision.Reason)
}

// max - Función helper para obtener el máximo de dos enteros
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
