package api

import (
	"encoding/json"
	"html/template"
	"log/slog"
	"net/http"
	"path"
	"strings"
)

// AbsoluteURL modifies requests so that the URL is absolute.
func AbsoluteURL(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Scheme == "" {
			r.URL.Scheme = "http"
		}

		r.URL.Host = r.Host

		next.ServeHTTP(w, r)
	})
}

// RenderMiddleware makes a request to the API for the given path, if not
// already an API request, and renders the response.
func RenderMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cleanPath := path.Clean(r.URL.Path)

		// If the request is for the API, pass it through.
		if strings.HasPrefix(cleanPath, "/api/") {
			next.ServeHTTP(w, r)
			return
		}

		slog.InfoContext(
			r.Context(),
			"rendering template",
			slog.String("path", cleanPath),
		)

		// If the request has a specific file extension, return a 404.
		if path.Ext(cleanPath) != "" {
			http.Error(w, "Not Found", http.StatusNotFound)

			return
		}

		if r.Method != http.MethodGet {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)

			return
		}

		// Try the untemplated version first.
		untemplated, err := GetFirstTemplate(cleanPath+".html", path.Join(cleanPath, "index.html"))
		if err == nil {
			w.Write(untemplated)

			return
		}

		// If the templated version doesn't exist, return a 404.
		tmpl, err := GetFirstTemplate(cleanPath+".tmpl.html", path.Join(cleanPath, "index.tmpl.html"))
		if err != nil {
			http.Error(w, "Not Found", http.StatusNotFound)

			return
		}

		// Build the response template.
		template, err := template.New("response").Parse(string(tmpl))
		if err != nil {
			slog.ErrorContext(
				r.Context(),
				"failed to parse template",
				slog.Any("error", err),
			)

			http.Error(w, "Failed to parse template", http.StatusInternalServerError)

			return
		}

		apiRequest := r.Clone(r.Context())
		apiRequest.RequestURI = ""
		apiRequest.URL.Scheme = "http"
		apiRequest.URL.Path = path.Join("/api", cleanPath)

		// Make the request to the API.
		apiResponse, err := http.DefaultClient.Do(apiRequest)
		if err != nil {
			slog.ErrorContext(
				r.Context(),
				"failed to make API request",
				slog.Any("error", err),
			)

			http.Error(w, "Failed to make API request", http.StatusInternalServerError)

			return
		}

		defer apiResponse.Body.Close()

		contentType := apiResponse.Header.Get("Content-Type")
		if contentType != "application/json" {
			slog.ErrorContext(
				r.Context(),
				"API response is not JSON",
				slog.String("content_type", contentType),
			)

			http.Error(w, "API response is not JSON", http.StatusInternalServerError)
		}

		// Must decode to a map, to allow rendering the template.
		var decodedResponse map[string]any

		if err := json.NewDecoder(apiResponse.Body).Decode(&decodedResponse); err != nil {
			slog.ErrorContext(
				r.Context(),
				"failed to decode API response",
				slog.Any("error", err),
			)

			http.Error(w, "Failed to decode API response", http.StatusInternalServerError)

			return
		}

		// Add some useful utilities.
		decodedResponse["Request"] = r.URL

		// Render the response template.
		if err := template.Execute(w, decodedResponse); err != nil {
			slog.ErrorContext(
				r.Context(),
				"failed to render template",
				slog.Any("error", err),
			)

			http.Error(w, "Failed to render template", http.StatusInternalServerError)

			return
		}
	})
}
