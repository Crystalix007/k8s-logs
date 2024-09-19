package api

import (
	"context"
	"errors"
	"io"
	"os"
	"path"

	"github.com/oapi-codegen/runtime/types"
)

func (a *API) GetLog(
	ctx context.Context,
	request GetLogRequestObject,
) (GetLogResponseObject, error) {
	if request.Params.Path == "" {
		return GetLog400JSONResponse{
			Message: "Requires a non-empty log path",
		}, nil
	}

	name := path.Base(request.Params.Path)

	fileInfo, err := os.Stat(a.getLogPath(request.Params.Path))
	if errors.Is(err, os.ErrNotExist) {
		return GetLog404JSONResponse{
			Message: "The specified path does not exist",
		}, nil
	} else if err != nil {
		return GetLog400JSONResponse{
			Message: "Invalid path",
		}, nil
	}

	return GetLog200JSONResponse{
		Name:     name,
		Path:     request.Params.Path,
		FileSize: int(fileInfo.Size()),
	}, nil
}

func (a *API) GetLogRaw(
	ctx context.Context,
	request GetLogRawRequestObject,
) (GetLogRawResponseObject, error) {
	if request.Params.Path == "" {
		return GetLogRaw404JSONResponse{
			Message: "The specified path does not exist",
		}, nil
	}

	path := a.getLogPath(request.Params.Path)

	file, err := os.Open(path)
	if err != nil {
		return GetLogRaw400JSONResponse{
			Message: "Failed to open file",
		}, nil
	}

	defer file.Close()

	bs, err := io.ReadAll(file)
	if err != nil {
		return GetLogRaw400JSONResponse{
			Message: "Failed to read file",
		}, nil
	}

	var responseFile types.File

	responseFile.InitFromBytes(bs, path)

	return GetLogRaw200JSONResponse{
		Contents: responseFile,
	}, nil
}

func (a *API) getLogPath(
	requestPath string,
) string {
	return path.Join(a.workingDirectory, path.Clean(requestPath))
}
