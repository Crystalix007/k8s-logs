package api

import (
	"context"
	"io"
	"os"
	"path"

	"github.com/oapi-codegen/runtime/types"
)

func (a *API) GetLogRaw(
	ctx context.Context,
	request GetLogRawRequestObject,
) (GetLogRawResponseObject, error) {
	if request.Params.Path == "" {
		return GetLogRaw404JSONResponse{
			Message: "The specified path does not exist",
		}, nil
	}

	path := path.Join(
		a.workingDirectory,
		path.Clean(request.Params.Path),
	)

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
