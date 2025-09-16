package application

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/juanbautista0/go-proxy/internal/domain"
)

type TriggerServiceImpl struct {
	config          *domain.Config
	metrics         *domain.TrafficMetrics
	executor        domain.ActionExecutor
	stopCh          chan struct{}
	lastHighTrigger time.Time
	lastLowTrigger  time.Time
	cooldownPeriod  time.Duration
	currentState    string // "normal", "high", "low"
}

func NewTriggerService(executor domain.ActionExecutor) *TriggerServiceImpl {
	return &TriggerServiceImpl{
		executor:       executor,
		cooldownPeriod: 30 * time.Second, // Evitar triggers repetidos
		currentState:   "normal",
	}
}

func (t *TriggerServiceImpl) Start(config *domain.Config, metrics *domain.TrafficMetrics) error {
	t.config = config
	t.metrics = metrics
	t.stopCh = make(chan struct{})

	go t.monitorTraffic()
	go t.monitorSchedule()

	return nil
}

func (t *TriggerServiceImpl) Stop() error {
	if t.stopCh != nil {
		close(t.stopCh)
	}
	return nil
}

func (t *TriggerServiceImpl) monitorTraffic() {
	ticker := time.NewTicker(5 * time.Second) // Revisar cada 5 segundos
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			rps := t.metrics.RequestsPerSecond
			trigger := t.config.Triggers.Traffic
			now := time.Now()

			// Trigger de tráfico alto
			if rps >= trigger.HighThreshold && t.currentState != "high" {
				if now.Sub(t.lastHighTrigger) > t.cooldownPeriod {
					if action, exists := t.config.Actions[trigger.HighAction]; exists {
						fmt.Printf("🔥 HIGH TRAFFIC TRIGGER: %d RPS >= %d threshold, executing %s\n", 
							rps, trigger.HighThreshold, trigger.HighAction)
						t.executor.Execute(trigger.HighAction, action)
						t.lastHighTrigger = now
						t.currentState = "high"
					}
				}
			}

			// Trigger de tráfico bajo
			if rps <= trigger.LowThreshold && t.currentState != "low" {
				if now.Sub(t.lastLowTrigger) > t.cooldownPeriod {
					if action, exists := t.config.Actions[trigger.LowAction]; exists {
						fmt.Printf("📉 LOW TRAFFIC TRIGGER: %d RPS <= %d threshold, executing %s\n", 
							rps, trigger.LowThreshold, trigger.LowAction)
						t.executor.Execute(trigger.LowAction, action)
						t.lastLowTrigger = now
						t.currentState = "low"
					}
				}
			}

			// Resetear estado si el tráfico vuelve a normal
			if rps > trigger.LowThreshold && rps < trigger.HighThreshold {
				if t.currentState != "normal" {
					fmt.Printf("✅ TRAFFIC NORMALIZED: %d RPS (between %d and %d)\n", 
						rps, trigger.LowThreshold, trigger.HighThreshold)
					t.currentState = "normal"
				}
			}

		case <-t.stopCh:
			return
		}
	}
}

func (t *TriggerServiceImpl) monitorSchedule() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			now := time.Now()
			currentTime := fmt.Sprintf("%02d:%02d", now.Hour(), now.Minute())

			for _, schedule := range t.config.Triggers.Schedule {
				if t.timeMatches(currentTime, schedule.Time) {
					if action, exists := t.config.Actions[schedule.Action]; exists {
						t.executor.Execute(schedule.Action, action)
					}
				}
			}
		case <-t.stopCh:
			return
		}
	}
}

func (t *TriggerServiceImpl) timeMatches(current, target string) bool {
	currentParts := strings.Split(current, ":")
	targetParts := strings.Split(target, ":")

	if len(currentParts) != 2 || len(targetParts) != 2 {
		return false
	}

	currentHour, _ := strconv.Atoi(currentParts[0])
	currentMin, _ := strconv.Atoi(currentParts[1])
	targetHour, _ := strconv.Atoi(targetParts[0])
	targetMin, _ := strconv.Atoi(targetParts[1])

	return currentHour == targetHour && currentMin == targetMin
}