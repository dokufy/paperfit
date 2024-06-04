package service

import (
	"errors"
)

var (
	ErrInvalidRequest = errors.New("invalid request")
	ErrTimeout        = errors.New("request timed out")
	ErrInternalError  = errors.New("internal error")
)

type PDFserviceProvider interface {
	Generate(project *Project) ([]byte, error)
}
