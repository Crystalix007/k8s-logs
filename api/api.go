// Package api provides the API for the log viewing service.
package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	kmiddleware "github.com/crystalix007/log-viewer/middleware"
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
	mux.Use(middleware.Logger)
	mux.Use(middleware.RedirectSlashes)
	mux.Use(kmiddleware.AbsoluteURL)

	mux.Get("/api/openapi.json", a.GetOpenAPISpec)
	mux.Get("/api", a.RenderDocs)

	// Apply the render middleware to all other routes.
	mux.Group(func(r chi.Router) {
		r.Use(kmiddleware.RenderMiddleware(templates, "templates"))

		// If the request is not handled by the Render middleware, return a 404.
		r.Handle("/*", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "Not Found", http.StatusNotFound)
		}))
	})

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
