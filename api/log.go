package api

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"strings"

	"github.com/oapi-codegen/runtime/types"
)

// ErrUnsafePath is returned when a path is unsafe, i.e. it escapes the working
// directory.
var ErrUnsafePath = errors.New("api: path is unsafe")

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

	path, err := a.getSafePath(request.Params.Path)
	if err != nil {
		return GetLog400JSONResponse{
			Message: "Invalid path",
		}, nil
	}

	fileInfo, err := os.Stat(path)
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

const pageSize = 50

func (a *API) GetLogPage(
	ctx context.Context,
	request GetLogPageRequestObject,
) (GetLogPageResponseObject, error) {
	if request.Params.Path == "" {
		return GetLogPage400JSONResponse{
			Message: "Requires a non-empty log path",
		}, nil
	}

	path, err := a.getSafePath(request.Params.Path)
	if err != nil {
		return GetLogPage400JSONResponse{
			Message: "Invalid path",
		}, nil
	}

	file, err := os.Open(path)
	if err != nil {
		return GetLogPage400JSONResponse{
			Message: "Failed to open file",
		}, nil
	}

	defer file.Close()

	bs, err := io.ReadAll(file)
	if err != nil {
		return GetLogPage400JSONResponse{
			Message: "Failed to read file",
		}, nil
	}

	lines := bytes.Split(bs, []byte("\n"))
	var contents types.File

	var page int

	if request.Params.Page != nil {
		page = *request.Params.Page
	}

	if len(lines) > pageSize*page {
		startIndex := pageSize * page
		endIndex := pageSize * (page + 1)

		if endIndex > len(lines) {
			endIndex = len(lines)
		}

		contents.InitFromBytes(bytes.Join(lines[startIndex:endIndex], []byte("\n")), path)
	}

	var (
		previousPage *int
		nextPage     *int
	)

	if page > 1 {
		previousPage = new(int)
		*previousPage = page - 1
	}

	if page+1 < len(lines)/pageSize {
		nextPage = new(int)
		*nextPage = page + 1
	}

	return GetLogPage200JSONResponse{
		PreviousPage: previousPage,
		Page:         page,
		NextPage:     nextPage,
		Contents:     contents,
		Path:         request.Params.Path,
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

	path, err := a.getSafePath(request.Params.Path)
	if err != nil {
		return GetLogRaw400JSONResponse{
			Message: "Invalid path",
		}, nil
	}

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

func (a *API) getSafePath(
	requestPath string,
) (string, error) {
	joinedPath := path.Join(a.workingDirectory, path.Clean(requestPath))

	workDirectory := path.Clean(a.workingDirectory)
	workDirectoryPrefix := fmt.Sprintf("%s/", workDirectory)

	if joinedPath != workDirectory && !strings.HasPrefix(joinedPath, workDirectoryPrefix) {
		return "", ErrUnsafePath
	}

	return joinedPath, nil
}
