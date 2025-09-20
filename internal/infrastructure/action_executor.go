package infrastructure

import (
	"bytes"
	"context"
	"net/http"
	"time"

	"github.com/juanbautista0/go-proxy/internal/domain"
)

type HTTPActionExecutor struct {
	client *http.Client
}

func NewHTTPActionExecutor() *HTTPActionExecutor {
	return &HTTPActionExecutor{
		client: &http.Client{
			Timeout: 5 * time.Second, // Timeout corto para no bloquear
		},
	}
}

func (e *HTTPActionExecutor) Execute(actionName string, config domain.ActionConfig) error {
	// Ejecutar de forma completamente asíncrona sin bloquear
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		
		req, err := http.NewRequestWithContext(ctx, config.Method, config.URL, bytes.NewBuffer([]byte{}))
		if err != nil {
			return
		}
		req.Header.Set("Content-Type", "application/json")
		e.client.Do(req) // Ignorar respuesta y errores para no bloquear
	}()
	return nil // Retornar inmediatamente
}