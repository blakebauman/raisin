package httpapi

import (
	"net/http"

	"github.com/blakebauman/raisin/packages/openapi"
)

const docsHTML = `<!doctype html>
<html lang="en">
  <head>
    <meta charset="utf-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1" />
    <title>Raisin API Reference</title>
    <meta name="description" content="Raisin Email API — OpenAPI reference" />
    <style>
      body { margin: 0; }
    </style>
  </head>
  <body>
    <script
      id="api-reference"
      data-url="/openapi.yaml"
      data-configuration='{"theme":"kepler","hideModels":true,"defaultHttpClient":{"targetKey":"shell","clientKey":"curl"},"operationsSorter":"alpha"}'
    ></script>
    <script src="https://cdn.jsdelivr.net/npm/@scalar/api-reference@1.28.11"></script>
  </body>
</html>
`

func (s *Server) serveOpenAPIYAML(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/yaml; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=60")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(openapi.SpecYAML)
}

func (s *Server) serveAPIDocs(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=60")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(docsHTML))
}
