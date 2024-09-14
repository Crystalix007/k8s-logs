package api

import (
	_ "embed"
	"net/http"
)

// renderDocs stores the HTML for the API documentation.
//
//go:embed docs.html
var renderDocs []byte

// GetOpenAPISpec returns the OpenAPI specification for the API.
func (a *API) GetOpenAPISpec(w http.ResponseWriter, r *http.Request) {
	spec, err := rawSpec()
	if err != nil {
		http.Error(w, "failed to get OpenAPI spec", http.StatusInternalServerError)

		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(spec)
}

// RenderDocs renders the API documentation.
func (a *API) RenderDocs(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	w.Write(renderDocs)
}
