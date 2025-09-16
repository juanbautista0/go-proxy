package infrastructure

import (
	"net/http"
	"os"
	"path/filepath"
)

type SwaggerHandler struct {
	swaggerFile string
}

func NewSwaggerHandler() *SwaggerHandler {
	return &SwaggerHandler{
		swaggerFile: "api-docs-en.yaml",
	}
}

func (s *SwaggerHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/swagger":
		s.serveSwaggerUI(w, r)
	case "/api-docs.yaml":
		s.serveSwaggerSpec(w, r)
	default:
		http.NotFound(w, r)
	}
}

func (s *SwaggerHandler) serveSwaggerUI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	html := `<!DOCTYPE html>
<html>
<head>
    <title>Go-Proxy API Documentation</title>
    <link rel="stylesheet" type="text/css" href="https://unpkg.com/swagger-ui-dist@4.15.5/swagger-ui.css" />
    <style>
        html { box-sizing: border-box; overflow: -moz-scrollbars-vertical; overflow-y: scroll; }
        *, *:before, *:after { box-sizing: inherit; }
        body { margin:0; background: #fafafa; }
    </style>
</head>
<body>
    <div id="swagger-ui"></div>
    <script src="https://unpkg.com/swagger-ui-dist@4.15.5/swagger-ui-bundle.js"></script>
    <script src="https://unpkg.com/swagger-ui-dist@4.15.5/swagger-ui-standalone-preset.js"></script>
    <script>
        window.onload = function() {
            const ui = SwaggerUIBundle({
                url: '/api-docs.yaml',
                dom_id: '#swagger-ui',
                deepLinking: true,
                presets: [
                    SwaggerUIBundle.presets.apis,
                    SwaggerUIStandalonePreset
                ],
                plugins: [
                    SwaggerUIBundle.plugins.DownloadUrl
                ],
                layout: "StandaloneLayout"
            });
        };
    </script>
</body>
</html>`
	w.Write([]byte(html))
}

func (s *SwaggerHandler) serveSwaggerSpec(w http.ResponseWriter, r *http.Request) {
	// Get current working directory
	wd, err := os.Getwd()
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Construct path to swagger file
	swaggerPath := filepath.Join(wd, s.swaggerFile)
	
	// Check if file exists
	if _, err := os.Stat(swaggerPath); os.IsNotExist(err) {
		http.Error(w, "Swagger spec not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/yaml")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	http.ServeFile(w, r, swaggerPath)
}