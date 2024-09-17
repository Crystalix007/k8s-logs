package api

import (
	"context"
	"errors"
	"os"
	"path"
	"path/filepath"
)

// GetLogs retrieves logs from the log viewing service.
func (a *API) GetLogs(
	ctx context.Context,
	request GetLogsRequestObject,
) (GetLogsResponseObject, error) {
	requestPath := "/"
	directory := a.workingDirectory

	if request.Params.Path != nil {
		requestPath = filepath.Clean(*request.Params.Path)
		directory = filepath.Join(directory, requestPath)
	}

	direntries, err := os.ReadDir(directory)
	if errors.Is(err, os.ErrNotExist) {
		return GetLogs404JSONResponse{
			Message: "The specified path does not exist",
		}, nil
	}

	response := GetLogs200JSONResponse{
		Logfiles: make([]LogFile, len(direntries)),
	}

	for i, direntry := range direntries {
		response.Logfiles[i] = LogFile{
			Name: direntry.Name(),
			Path: path.Join(requestPath, direntry.Name()),
			Dir:  direntry.IsDir(),
		}
	}

	return response, nil
}
