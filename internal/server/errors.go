package server

import (
	"errors"
	"net/http"

	"github.com/dokufy/paperfit/internal/apigen"
	"github.com/dokufy/paperfit/internal/logger"
	"github.com/labstack/echo/v5"
)

var (
	ErrBadRequest         = errors.New("bad request")
	ErrServiceUnavailable = errors.New("service unavailable")

	ErrInvalidInputFormat = errors.New("invalid input format")
	ErrBadVariables       = errors.New("provided variables do not conform to the json schema")
	ErrInvalidVariables   = errors.New("invalid variables format")

	ErrFailedExtractingArchive = errors.New("failed while extracting archive")

	ErrNoSchemaFile = errors.New("no json schema found in the archive, please make sure that a json schema exists and it is not empty or disable variables validation")

	ErrFailedCreatingCacheKey = errors.New("failed to create cache key")
	ErrFailedWritingCache     = errors.New("failed to write cache")
	ErrFailedReadingCache     = errors.New("failed to read cache")

	ErrFailedWritingResponse = errors.New("unexpected error when writing response")
	ErrFailedCleanupArchive  = errors.New("failed to cleanup existing archive")

	ErrFailedGeneratingPdf = errors.New("failed to generate pdf")
)

type TranslatedError struct {
	msg string
	err error
}

func NewTranslatedError(msg string, err error) TranslatedError {
	return TranslatedError{
		msg: msg,
		err: err,
	}
}

func (e TranslatedError) Error() string {
	return e.msg
}

func (e TranslatedError) Unwrap() error {
	return e.err
}

func sendError(ctx *echo.Context, code int, message string, err error) error {
	logger.Log.Debug(message, err)

	return ctx.JSON(code, message)
}

func (s *Server) writeError(ctx *echo.Context, code int, message interface{}) {
	err := ctx.JSON(code, message)
	if err != nil {
		s.logger.Error("Unexpected error when writing request", "err", err)
	}
}

func (s *Server) writeUnexpectedError(ctx *echo.Context) {
	s.writeError(ctx, http.StatusInternalServerError, apigen.UnknownError{
		Message: "Unexpected error",
	})
}

func (s *Server) writeBadRequestError(ctx *echo.Context, message string) {
	// TODO: bad request error type
	s.writeError(ctx, http.StatusBadRequest, apigen.ValidationError{
		Message: message,
	})
}
