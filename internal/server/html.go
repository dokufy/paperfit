package server

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/dokufy/paperfit/config"
	"github.com/dokufy/paperfit/internal/apigen"
	"github.com/dokufy/paperfit/internal/cache"
	"github.com/dokufy/paperfit/internal/service"
	"github.com/dokufy/paperfit/internal/validation"
	"github.com/labstack/echo/v5"
)

type DocumentHtmlInput struct {
	Html       string                 `json:"html"`
	Assets     map[string]string      `json:"assets,omitempty"`
	JSONSchema interface{}            `json:"jsonSchema,omitempty"`
	Input      apigen.Input           `json:"input"`
	RootPath   *string                `json:"rootPath,omitempty"`
	Options    *service.RenderOptions `json:"options,omitempty"`
}

func (s *Server) CreateDocumentFromHtml(c *echo.Context) error {
	ctx := c.Request().Context()

	var in DocumentHtmlInput
	if err := c.Bind(&in); err != nil {
		return sendError(c, http.StatusBadRequest, ErrInvalidInputFormat.Error(), err)
	}

	if in.Html == "" {
		return sendError(c, http.StatusBadRequest, ErrBadRequest.Error(), errors.New("html is required"))
	}

	project := service.NewProject([]*service.File{})

	// index.html
	_ = project.AppendFile("index.html", bytes.NewReader([]byte(in.Html)), int64(len(in.Html)))

	// assets
	for name := range in.Assets {
		if name == "index.html" || name == "json_schema.json" {
			return sendError(c, http.StatusBadRequest, ErrBadRequest.Error(), errors.New("invalid asset name"))
		}
		if strings.Contains(name, "..") {
			return sendError(c, http.StatusBadRequest, ErrBadRequest.Error(), errors.New("invalid asset name"))
		}
	}
	for name, content := range in.Assets {
		_ = project.AppendFile(name, bytes.NewReader([]byte(content)), int64(len(content)))
	}

	// json schema
	if in.JSONSchema != nil {
		var schemaBytes []byte
		var err error
		switch v := in.JSONSchema.(type) {
		case string:
			schemaBytes = []byte(v)
		default:
			schemaBytes, err = json.Marshal(v)
			if err != nil {
				return sendError(c, http.StatusBadRequest, ErrInvalidVariables.Error(), err)
			}
		}
		_ = project.AppendFile("json_schema.json", bytes.NewReader(schemaBytes), int64(len(schemaBytes)))
	}

	project.RenderOptions = in.Options

	var vars string
	if len(in.Input.Variables) > 0 {
		marshalledVars, err := json.Marshal(in.Input.Variables)
		if err != nil {
			return sendError(c, http.StatusBadRequest, ErrInvalidVariables.Error(), err)
		}
		vars = string(marshalledVars)
	}

	pdfPath := ""
	archivePath := ""
	if s.cacheInstance != nil {
		if config.AppConfig.CacheConfig != nil {
			pdfPath = config.AppConfig.CacheConfig.DefaultPDFpath
			archivePath = config.AppConfig.CacheConfig.DefaultArchivePath
		}
		if pdfPath == "" {
			pdfPath = "pdf"
		}
	}
	if in.RootPath != nil {
		if err := cache.ValidateRootPath(*in.RootPath); err != nil {
			return sendError(c, http.StatusBadRequest, ErrBadRequest.Error(), err)
		}
		pdfPath = *in.RootPath
	}

	cacheKeyInput := in.Html
	if len(in.Assets) > 0 {
		// include assets in key
		ab, _ := json.Marshal(in.Assets)
		cacheKeyInput += string(ab)
	}
	if in.Options != nil {
		ob, _ := json.Marshal(in.Options)
		cacheKeyInput += string(ob)
	}
	pdfCacheKey, err := cache.Key(cacheKeyInput, vars)
	if err != nil {
		return sendError(c, http.StatusInternalServerError, ErrFailedCreatingCacheKey.Error(), err)
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
		jsonSchemaContent, err := validation.Validate(vars, jsonSchema.Content)
		if err != nil {
			return sendError(c, http.StatusBadRequest, ErrBadVariables.Error(), err)
		}
		jsonSchema.Content = jsonSchemaContent
	}

	if s.cacheInstance != nil {
		cachedPdf, err := s.cacheInstance.Get(ctx, pdfCacheKey, pdfPath)
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

	res, size, err := s.pdfService.Generate(ctx, &project)
	if err != nil {
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
		err = s.cacheInstance.Set(ctx, pdfCacheKey, tee, size, pdfPath)
		if err != nil {
			return sendError(c, http.StatusInternalServerError, ErrFailedWritingCache.Error(), err)
		}
		res = buf
	}

	err = c.Stream(http.StatusOK, echo.MIMEApplicationJSON, res)
	if err != nil {
		return sendError(c, http.StatusInternalServerError, ErrFailedWritingResponse.Error(), err)
	}
	_ = archivePath
	return nil
}
