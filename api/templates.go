package api

import (
	"embed"
	"errors"
	"fmt"
	"path"
)

//go:embed templates/*
var templates embed.FS

// ErrNoTemplateFound is returned when no templates are found.
var ErrNoTemplateFound = errors.New("api: no templates found")

// GetTemplate reads a template from the templates directory.
func GetTemplate(name string) ([]byte, error) {
	path := path.Join("templates", path.Clean(name))

	template, err := templates.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to read template %s: %w",
			name,
			err,
		)
	}

	return template, nil
}

// GetFirstTemplate reads the first template that exists from a list of names.
func GetFirstTemplate(names ...string) ([]byte, error) {
	for _, name := range names {
		template, err := GetTemplate(name)
		if err == nil {
			return template, nil
		}
	}

	return nil, ErrNoTemplateFound
}
