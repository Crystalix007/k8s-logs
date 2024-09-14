package api

import (
	_ "embed"
	"encoding/json"
	"net/http"
	"path"

	"github.com/getkin/kin-openapi/openapi3"
)

// renderDocs stores the HTML for the API documentation.
//
//go:embed docs.html
var renderDocs []byte

// GetOpenAPISpec returns the OpenAPI specification for the API.
func (a *API) GetOpenAPISpec(w http.ResponseWriter, r *http.Request) {
	spec, err := GetSwagger()
	if err != nil {
		http.Error(w, "failed to get OpenAPI spec", http.StatusInternalServerError)

		return
	}

	// Add the server URL to the OpenAPI spec, by taking the current URL and
	// removing the last path component.
	serverURL := r.URL
	serverURL.Path = path.Dir(serverURL.Path)

	spec.Servers = append([]*openapi3.Server{
		{
			URL: serverURL.String(),
		},
	}, spec.Servers...)

	jsonSpec, err := json.Marshal(spec)
	if err != nil {
		http.Error(w, "failed to marshal OpenAPI spec", http.StatusInternalServerError)

		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonSpec)
}

// RenderDocs renders the API documentation.
func (a *API) RenderDocs(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	w.Write(renderDocs)
}
