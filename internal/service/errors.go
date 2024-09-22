package service

import "errors"

var (
	ErrInvalidRequest   = errors.New("invalid request")
	ErrTimeout          = errors.New("request timed out")
	ErrInternalError    = errors.New("internal error")
	ErrIndexFileMissing = errors.New("index.html is required")
)
