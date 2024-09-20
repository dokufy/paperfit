package service

import (
	"errors"
	"io"
)

var (
	ErrInvalidRequest = errors.New("invalid request")
	ErrTimeout        = errors.New("request timed out")
	ErrInternalError  = errors.New("internal error")
)

type PDFserviceGenerator interface {
	Generate(project *Project) (io.Reader, int64, error)
}
