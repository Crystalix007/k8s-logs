package api

import (
	"context"
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
		return GetLog404JSONResponse{
			Message: "The specified path does not exist",
		}, nil
	}

	path := path.Join(
		a.workingDirectory,
		path.Clean(request.Params.Path),
	)

	file, err := os.Open(path)
	if err != nil {
		return GetLog400JSONResponse{
			Message: "Failed to open file",
		}, nil
	}

	defer file.Close()

	bs, err := io.ReadAll(file)
	if err != nil {
		return GetLog400JSONResponse{
			Message: "Failed to read file",
		}, nil
	}

	var responseFile types.File

	responseFile.InitFromBytes(bs, path)

	return GetLog200JSONResponse{
		Contents: responseFile,
	}, nil
}
