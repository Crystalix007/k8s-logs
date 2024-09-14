package api

import (
	"fmt"
	"os"
)

// Option represents a value that can be configured on an API.
type Option func(a *API)

// WithWorkingDirectory sets the working directory on the API.
func WithWorkingDirectory(workingDirectory string) Option {
	return func(a *API) {
		a.workingDirectory = workingDirectory
	}
}

// setDefaults sets the default values on the API.
func (a *API) setDefaults() error {
	var err error

	if a.workingDirectory == "" {
		a.workingDirectory, err = os.Getwd()
		if err != nil {
			return fmt.Errorf(
				"api: getting working directory: %w",
				err,
			)
		}
	}

	return nil
}
