package server

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/dokufy/paperfit/internal/apigen"
	"github.com/dokufy/paperfit/internal/cache"
	"github.com/dokufy/paperfit/internal/httptool"
	"github.com/dokufy/paperfit/internal/service"
	"github.com/dokufy/paperfit/internal/service/projectprovider"
	"github.com/dokufy/paperfit/internal/validation"
	"github.com/labstack/echo/v4"
)

var (
	ErrNoUrlProvided         = errors.New("no url specified")
	ErrArchiveNotFound       = errors.New("archive not found at the specified url")
	ErrFailedFetchingArchive = errors.New("error while fetching the archive at the specified url")
)

func (s *Server) CreateDocumentFromArchiveUrl(c echo.Context) error {
	ctx := c.Request().Context()

	var in apigen.DocumentArchiveUrlInput
	var vars string
	err := c.Bind(&in)
	if err != nil {
		return sendError(c, http.StatusBadRequest, ErrBadRequest.Error(), err)
	}

	if in.Url == "" {
		return sendError(c, http.StatusBadRequest, ErrNoUrlProvided.Error(), err)
	}

	if in.Input.Variables != nil {
		marshalledVars, err := json.Marshal(in.Input.Variables)
		if err != nil {
			return sendError(c, http.StatusBadRequest, ErrInvalidVariables.Error(), err)
		}

		vars = string(marshalledVars)
	}

	pdfCacheKey, err := cache.Key(in.Url, httptool.FlattenHeaders(in.AdditionalHeaders), vars)
	if err != nil {
		return sendError(c, http.StatusInternalServerError, ErrFailedCreatingCacheKey.Error(), err)
	}

	archiveCacheKey, err := cache.Key(in.Url, httptool.FlattenHeaders(in.AdditionalHeaders))
	if err != nil {
		return sendError(c, http.StatusInternalServerError, ErrFailedCreatingCacheKey.Error(), err)
	}

	if s.cacheInstance != nil {
		cachedPdf, err := s.cacheInstance.GetPDF(ctx, pdfCacheKey)
		if err != nil {
			return sendError(c, http.StatusInternalServerError, ErrFailedReadingCache.Error(), err)
		}

		if cachedPdf != nil {
			err = c.Stream(http.StatusOK, "application/pdf", cachedPdf)
			if err != nil {
				return sendError(c, http.StatusInternalServerError, ErrFailedWritingResponse.Error(), err)
			}
		}
	}

	project, err := s.archiveProjectProvider.GetProject(ctx, in.Url, in.AdditionalHeaders)
	if err != nil {
		if projectErr, ok := err.(*projectprovider.RequestError); ok {
			if projectErr.StatusCode == 404 {
				return sendError(c, http.StatusNotFound, ErrArchiveNotFound.Error(), err)
			}
		}

		return sendError(c, http.StatusInternalServerError, ErrFailedFetchingArchive.Error(), err)
	}

	if vars != "" {
		err = project.AppendVariables(vars)
		if err != nil {
			return sendError(c, http.StatusBadRequest, ErrInvalidVariables.Error(), err)
		}
	}

	if in.Input.Validate {
		jsonSchema := project.GetJSONschemaFile()

		if jsonSchema == nil || jsonSchema.Size() == 0 {
			return sendError(c, http.StatusBadRequest, ErrNoSchemaFile.Error(), err)
		}

		jsonSchemaContent, err := validation.Validate(string(vars), jsonSchema.Content)
		if err != nil {
			return sendError(c, http.StatusBadRequest, ErrBadVariables.Error(), err)
		}

		jsonSchema.Content = jsonSchemaContent
	}

	res, size, err := s.pdfService.Generate(&project)
	if err != nil {
		if s.cacheInstance != nil {
			if err2 := s.cacheInstance.DeleteArchive(ctx, archiveCacheKey); err2 != nil {
				return sendError(c, http.StatusBadRequest, ErrBadRequest.Error(), err)
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
		err = s.cacheInstance.SetPDF(ctx, pdfCacheKey, res, size)
		if err != nil {
			return sendError(c, http.StatusInternalServerError, ErrFailedWritingCache.Error(), err)
		}
	}

	err = c.Stream(http.StatusOK, "application/pdf", res)
	if err != nil {
		return sendError(c, http.StatusInternalServerError, ErrFailedWritingResponse.Error(), err)
	}

	return nil
}
