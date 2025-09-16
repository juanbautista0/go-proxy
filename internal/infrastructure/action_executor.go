package infrastructure

import (
	"bytes"
	"net/http"

	"github.com/juanbautista0/go-proxy/internal/domain"
)

type HTTPActionExecutor struct {
	client *http.Client
}

func NewHTTPActionExecutor() *HTTPActionExecutor {
	return &HTTPActionExecutor{
		client: &http.Client{},
	}
}

func (e *HTTPActionExecutor) Execute(actionName string, config domain.ActionConfig) error {
	req, err := http.NewRequest(config.Method, config.URL, bytes.NewBuffer([]byte{}))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	_, err = e.client.Do(req)
	return err
}