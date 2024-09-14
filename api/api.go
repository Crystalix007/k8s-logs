// Package api provides the API for the log viewing service.
package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

//go:generate go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen --config=oapi-codegen.yaml api.yaml

// API is the API for the log viewing service.
type API struct {
	router           http.Handler
	workingDirectory string
}

// New creates a new instance of the API, with all the necessary routes and
// handlers.
func New(opts ...Option) (*API, error) {
	var a API

	for _, opt := range opts {
		opt(&a)
	}

	mux := chi.NewRouter()

	mux.Get("/api/openapi.json", a.GetOpenAPISpec)
	mux.Get("/api", a.RenderDocs)

	strictServer := NewStrictHandler(&a, nil)
	a.router = HandlerFromMuxWithBaseURL(strictServer, mux, "/api")

	if err := a.setDefaults(); err != nil {
		return nil, err
	}

	return &a, nil
}

// ServeHTTP responds to HTTP requests, implementing the [http.Handler]
// interface.
func (a *API) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.router.ServeHTTP(w, r)
}
