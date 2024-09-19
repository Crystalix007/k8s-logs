package api

import (
	"context"
	"errors"
	"os"
	"path"
)

// GetLogs retrieves logs from the log viewing service.
func (a *API) GetLogs(
	ctx context.Context,
	request GetLogsRequestObject,
) (GetLogsResponseObject, error) {
	requestPath := "/"

	if request.Params.Path != nil {
		requestPath = *request.Params.Path
	}

	directory, err := a.getSafePath(requestPath)
	if err != nil {
		return GetLogs400JSONResponse{
			Message: "Invalid path",
		}, nil
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
