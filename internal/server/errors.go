package server

import (
	"errors"
	"net/http"

	"github.com/dokufy/paperfit/internal/apigen"
	"github.com/dokufy/paperfit/internal/logger"
	"github.com/labstack/echo/v4"
)

var ErrMissingParam = errors.New("missing parameter")

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

func sendError(ctx echo.Context, code int, message string, err error) error {
	logger.Log.Debug(message, err)

	return ctx.JSON(code, message)
}

func (s *Server) writeError(ctx echo.Context, code int, message interface{}) {
	err := ctx.JSON(code, message)
	if err != nil {
		s.logger.Error("Unexpected error when writing request", "err", err)
	}
}

func (s *Server) writeUnexpectedError(ctx echo.Context) {
	s.writeError(ctx, http.StatusInternalServerError, apigen.UnknownError{
		Message: "Unexpected error",
	})
}

func (s *Server) writeBadRequestError(ctx echo.Context, message string) {
	// TODO: bad request error type
	s.writeError(ctx, http.StatusBadRequest, apigen.ValidationError{
		Message: message,
	})
}
