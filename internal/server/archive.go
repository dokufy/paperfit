package server

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime/multipart"
	"net/http"

	"github.com/dokufy/paperfit/internal/apigen"
	"github.com/dokufy/paperfit/internal/archive"
	"github.com/dokufy/paperfit/internal/cache"
	"github.com/dokufy/paperfit/internal/service"
	"github.com/dokufy/paperfit/internal/validation"
	"github.com/labstack/echo/v5"
)

var ErrMissingArchive = errors.New("archive is missing")

func (s *Server) CreateDocumentFromArchive(c *echo.Context) error {
	ctx := c.Request().Context()

	file, err := c.FormFile("file")
	if err != nil || file == nil {
		return sendError(c, http.StatusBadRequest, ErrMissingArchive.Error(), err)
	}

	var in apigen.Input
	var vars string

	if inputStr := c.FormValue("input"); inputStr != "" {
		err := json.Unmarshal([]byte(inputStr), &in)
		if err != nil {
			return sendError(c, http.StatusBadRequest, ErrInvalidInputFormat.Error(), err)
		}
	}

	project, archiveName, err := archiveToProject(ctx, file)
	if err != nil {
		return sendError(c, http.StatusInternalServerError, ErrFailedExtractingArchive.Error(), err)
	}

	if len(in.Variables) > 0 {
		marshalledVars, err := json.Marshal(in.Variables)
		if err != nil {
			return sendError(c, http.StatusBadRequest, ErrInvalidVariables.Error(), err)
		}

		vars = string(marshalledVars)
	}

	pdfCacheKey, err := cache.Key(archiveName, vars)
	if err != nil {
		return sendError(c, http.StatusInternalServerError, ErrFailedCreatingCacheKey.Error(), err)
	}

	archiveCacheKey, err := cache.Key(archiveName)
	if err != nil {
		return sendError(c, http.StatusInternalServerError, ErrFailedCreatingCacheKey.Error(), err)
	}

	if s.cacheInstance != nil {
		cachedPdf, err := s.cacheInstance.Get(ctx, pdfCacheKey, "pdf")
		if err != nil {
			return sendError(c, http.StatusInternalServerError, ErrFailedReadingCache.Error(), err)
		}

		if cachedPdf != nil {
			err = c.Stream(http.StatusOK, echo.MIMEApplicationJSON, cachedPdf)
			if err != nil {
				return sendError(c, http.StatusInternalServerError, ErrFailedWritingResponse.Error(), err)
			}

			return nil
		}
	}

	if vars != "" {
		err = project.AppendVariables(vars)
		if err != nil {
			return sendError(c, http.StatusBadRequest, ErrInvalidVariables.Error(), err)
		}
	}

	if in.Validate {
		jsonSchema := project.GetJSONschemaFile()

		if jsonSchema == nil || jsonSchema.Size() == 0 {
			return sendError(c, http.StatusBadRequest, ErrNoSchemaFile.Error(), err)
		}

		jsonSchemaContent, err := validation.Validate(vars, jsonSchema.Content)
		if err != nil {
			return sendError(c, http.StatusBadRequest, ErrBadVariables.Error(), err)
		}

		jsonSchema.Content = jsonSchemaContent
	}

	res, size, err := s.pdfService.Generate(project)
	if err != nil {
		if s.cacheInstance != nil {
			if err2 := s.cacheInstance.Delete(ctx, archiveCacheKey, "archive"); err2 != nil {
				return sendError(c, http.StatusInternalServerError, ErrFailedCleanupArchive.Error(), err)
			}
		}

		if errors.Is(err, service.ErrInvalidRequest) {
			return sendError(c, http.StatusBadRequest, ErrBadRequest.Error(), err)
		}

		if errors.Is(err, service.ErrTimeout) {
			return sendError(c, http.StatusServiceUnavailable, ErrServiceUnavailable.Error(), err)
		}

		return sendError(c, http.StatusInternalServerError, ErrFailedGeneratingPdf.Error(), err)
	}

	if s.cacheInstance != nil {
		buf := new(bytes.Buffer)
		tee := io.TeeReader(res, buf)

		err = s.cacheInstance.Set(ctx, pdfCacheKey, tee, size, "pdf")
		if err != nil {
			return sendError(c, http.StatusInternalServerError, ErrFailedWritingCache.Error(), err)
		}

		res = buf
	}

	err = c.Stream(http.StatusOK, echo.MIMEApplicationJSON, res)
	if err != nil {
		return sendError(c, http.StatusInternalServerError, ErrFailedWritingResponse.Error(), err)
	}

	return nil
}

func archiveToProject(ctx context.Context, file *multipart.FileHeader) (*service.Project, string, error) {
	src, err := file.Open()
	if err != nil {
		return nil, "", err
	}

	defer src.Close()

	project, err := archive.ExtractToProject(ctx, file.Filename, src)
	if err != nil {
		return nil, "", err
	}

	return &project, file.Filename, nil
}
