package middleware

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"path"
	"path/filepath"
	"strings"
	"text/template"
)

var (
	// ErrNoTemplateFound is returned when no matching templates are found.
	ErrNoTemplateFound = errors.New("middleware: no templates found")

	// extensionMimeType is a map of file extensions to MIME types.
	extensionMimeType = map[string]string{
		"html": "text/html",
		"txt":  "text/plain",
	}
)

// TemplateExtension is the extension used for template files.
//
// I.e. a file named "index.tmpl.html" would be a template file.
const TemplateExtension = "tmpl"

// TemplateSource is an interface that combines the capabilities of fs.ReadDirFS
// and fs.ReadFileFS.
// It represents a source from which templates can be read, allowing directory
// reading and file reading operations.
type TemplateSource interface {
	fs.ReadDirFS
	fs.ReadFileFS
}

// Template represents the properties of a single template.
type Template struct {
	Format    string
	Templated bool
	Path      string
}

// Templates represents a collection of templates, which can be looked up by
// name.
type Templates struct {
	fs            fs.ReadFileFS
	templateNames map[string][]Template
}

// DecodeBase64 decodes a base64-encoded string.
func DecodeBase64(str string) (string, error) {
	bs, err := base64.StdEncoding.DecodeString(str)
	if err != nil {
		return "", fmt.Errorf("middleware: decoding template base64: %w", err)
	}

	return string(bs), nil
}

// NewTemplates creates a new instance of Templates, using the given file system
// as the source of templates, and the given path as the root directory.
func NewTemplates(ts TemplateSource, rootDir string) (*Templates, error) {
	rootDir = path.Clean(rootDir)

	templates := &Templates{
		fs:            ts,
		templateNames: make(map[string][]Template),
	}

	// Walk the directory to get all the templates.
	if err := fs.WalkDir(ts, rootDir, func(fullpath string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Walk all directories.
		if d.IsDir() {
			return nil
		}

		// Get the templates from the directory entry.
		dirTemplates, err := templatesFromDirentry(rootDir, fullpath, d)
		if err != nil {
			return err
		}

		for name, template := range dirTemplates {
			templates.templateNames[name] = append(templates.templateNames[name], template)
		}

		return nil
	}); err != nil {
		return nil, fmt.Errorf("middleware: walking template directory %s: %w", rootDir, err)
	}

	return templates, nil
}

// templatesFromDirentry reads a file directory entry and returns a map of
// templates found within it.
func templatesFromDirentry(
	rootDir string,
	path string,
	direntry fs.DirEntry,
) (map[string]Template, error) {
	if direntry.IsDir() {
		panic("middleware: templatesFromDirentry called with a directory")
	}

	filename := filepath.Base(path)

	name, ext, found := strings.Cut(filename, ".")
	if !found {
		return map[string]Template{}, nil
	}

	relPath, err := filepath.Rel(rootDir, path)
	if err != nil {
		return map[string]Template{}, fmt.Errorf(
			"middleware: getting relative path: %w", err,
		)
	}

	templateName := filepath.Join(filepath.Dir(relPath), name)

	if name == "index" {
		templateName = filepath.Dir(templateName)
	}

	templated := strings.HasPrefix(ext, TemplateExtension+".")
	ext = strings.TrimPrefix(ext, TemplateExtension+".")
	mimeType, ok := extensionMimeType[ext]

	if !ok {
		slog.Info("unknown MIME type", slog.String("extension", ext))
	}

	return map[string]Template{
		templateName: {
			Format:    mimeType,
			Templated: templated,
			Path:      path,
		},
	}, nil
}

// Open opens a template file for reading.
func (t Templates) Open(name string) (*Template, io.ReadCloser, error) {
	templateName, err := filepath.Rel("/", path.Clean(name))
	if err != nil {
		return nil, nil, fmt.Errorf("middleware: getting relative path: %w", err)
	}

	template, ok := t.templateNames[templateName]
	if !ok {
		return nil, nil, ErrNoTemplateFound
	}

	templateFile, err := t.fs.Open(template[0].Path)
	if err != nil {
		return nil, nil, fmt.Errorf(
			"middleware: opening template file %s: %w",
			template[0].Path,
			err,
		)
	}

	return &template[0], templateFile, nil
}

// RenderMiddleware makes a request to the API for the given path, if not
// already an API request, and renders the response.
func RenderMiddleware(
	templateSource TemplateSource,
	templateDir string,
) func(next http.Handler) http.Handler {
	templates, err := NewTemplates(templateSource, templateDir)
	if err != nil {
		panic(err)
	}

	for name, templates := range templates.templateNames {
		for _, template := range templates {
			slog.Info(
				"found template",
				slog.String("name", name),
				slog.String("path", template.Path),
				slog.String("format", template.Format),
			)
		}
	}

	// GetTemplate reads a template from the templates directory.
	GetTemplate := func(name string) (*Template, []byte, error) {
		templatePath := path.Clean(name)

		template, templateFile, err := templates.Open(templatePath)
		if err != nil {
			return nil, nil, fmt.Errorf(
				"middleware: opening template file %s: %w",
				templatePath,
				err,
			)
		}

		defer templateFile.Close()

		templateContents, err := io.ReadAll(templateFile)
		if err != nil {
			return nil, nil, fmt.Errorf(
				"middleware: reading template %s: %w",
				name,
				err,
			)
		}

		return template, templateContents, nil
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

			templateDetails, templateContent, err := GetTemplate(cleanPath)
			if errors.Is(err, ErrNoTemplateFound) {
				StandardError(w, http.StatusNotFound)

				return
			} else if err != nil {
				slog.InfoContext(
					r.Context(),
					"failed to get template",
					slog.Any("error", err),
				)

				StandardError(w, http.StatusInternalServerError)

				return
			}

			// If the template is not a template, return it as is.
			if !templateDetails.Templated {
				w.Header().Set("Content-Type", templateDetails.Format)
				w.Write(templateContent)

				return
			}

			// Build the response template.
			responseTemplate := template.New("response")

			responseTemplate.Funcs(template.FuncMap{
				"from_base64": DecodeBase64,
			})

			_, err = responseTemplate.Parse(string(templateContent))
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

			w.Header().Set("Content-Type", templateDetails.Format)

			// Render the response template.
			if err := responseTemplate.Execute(w, decodedResponse); err != nil {
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
