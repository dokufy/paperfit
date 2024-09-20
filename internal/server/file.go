package server

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/dokufy/paperfit/internal/apigen"
	"github.com/dokufy/paperfit/internal/service"
	"github.com/dokufy/paperfit/internal/validation"
	"github.com/labstack/echo/v4"
)

var (
	ErrNoFileProvided   = errors.New("at least one file must be provided")
	ErrFailedToReadFile = errors.New("failed to read file")
)

func (s *Server) CreateDocumentFromFile(c echo.Context) error {
	form, err := c.MultipartForm()
	if err != nil {
		return sendError(c, http.StatusBadRequest, ErrInvalidInputFormat.Error(), err)
	}

	files := form.File["files"]

	if len(files) == 0 {
		return sendError(c, http.StatusBadRequest, ErrNoFileProvided.Error(), err)
	}

	var in apigen.Input

	if inputStr := c.FormValue("input"); inputStr != "" {
		err := json.Unmarshal([]byte(inputStr), &in)
		if err != nil {
			return sendError(c, http.StatusBadRequest, ErrInvalidInputFormat.Error(), err)
		}
	}

	project := service.NewProject([]*service.File{})

	for _, file := range files {
		if file.Size > 0 {
			of, err := file.Open()
			if err != nil {
				return sendError(c, http.StatusBadRequest, ErrFailedToReadFile.Error(), err)
			}

			project.AppendFile(file.Filename, of, file.Size)
		}
	}

	vars, err := json.Marshal(in.Variables)
	if err != nil {
		return sendError(c, http.StatusBadRequest, ErrInvalidVariables.Error(), err)
	}

	if vars != nil && len(vars) > 0 {
		err := project.AppendVariables(string(vars))
		if err != nil {
			return sendError(c, http.StatusBadRequest, ErrInvalidVariables.Error(), err)
		}
	}

	if in.Validate {
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

	res, _, err := s.pdfService.Generate(&project)
	if err != nil {
		if errors.Is(err, service.ErrInvalidRequest) {
			return sendError(c, http.StatusBadRequest, ErrBadRequest.Error(), err)
		}

		if errors.Is(err, service.ErrTimeout) {
			return sendError(c, http.StatusServiceUnavailable, ErrServiceUnavailable.Error(), err)
		}

		return sendError(c, http.StatusInternalServerError, ErrFailedGeneratingPdf.Error(), err)
	}

	err = c.Stream(http.StatusOK, "application/pdf", res)
	if err != nil {
		return sendError(c, http.StatusInternalServerError, ErrFailedWritingResponse.Error(), err)
	}

	return nil
}
