package middleware

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"path"
	"strings"
	"text/template"
)

// ErrNoTemplateFound is returned when no matching templates are found.
var ErrNoTemplateFound = errors.New("middleware: no templates found")

// RenderMiddleware makes a request to the API for the given path, if not
// already an API request, and renders the response.
func RenderMiddleware(
	templates fs.FS,
	templateDir string,
) func(next http.Handler) http.Handler {
	// GetTemplate reads a template from the templates directory.
	GetTemplate := func(name string) ([]byte, error) {
		templatePath := path.Clean(name)

		if templateDir != "" {
			templatePath = path.Join(templateDir, templatePath)
		}

		templateFile, err := templates.Open(templatePath)
		if err != nil {
			return nil, fmt.Errorf(
				"middleware: opening template file %s: %w",
				templatePath,
				err,
			)
		}

		defer templateFile.Close()

		template, err := io.ReadAll(templateFile)
		if err != nil {
			return nil, fmt.Errorf(
				"middleware: reading template %s: %w",
				name,
				err,
			)
		}

		return template, nil
	}

	// GetFirstTemplate reads the first template that exists from a list of
	// names.
	GetFirstTemplate := func(names ...string) ([]byte, error) {
		for _, name := range names {
			template, err := GetTemplate(name)
			if err == nil {
				return template, nil
			}
		}

		return nil, ErrNoTemplateFound
	}

	// StandardError responds with a standard error message.
	StandardError := func(w http.ResponseWriter, status int) {
		http.Error(w, http.StatusText(status), status)
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cleanPath := path.Clean(r.URL.Path)

			// If the request is for the API, pass it through.
			if strings.HasPrefix(cleanPath, "/api/") {
				next.ServeHTTP(w, r)
				return
			}

			// If the request has a specific file extension, return a 404.
			if path.Ext(cleanPath) != "" {
				StandardError(w, http.StatusNotFound)

				return
			}

			if r.Method != http.MethodGet {
				StandardError(w, http.StatusMethodNotAllowed)

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
				StandardError(w, http.StatusNotFound)

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

			if apiResponse.StatusCode == http.StatusNotFound {
				http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)

				return
			}

			if apiResponse.StatusCode < http.StatusOK || apiResponse.StatusCode >= http.StatusMultipleChoices {
				slog.ErrorContext(
					r.Context(),
					"API request failed",
					slog.Any("status_code", apiResponse.StatusCode),
				)

				http.Error(w, "API request failed", apiResponse.StatusCode)

				return
			}

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
}
