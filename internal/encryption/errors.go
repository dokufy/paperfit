package encryption

import "errors"

var (
	ErrIVwrite = errors.New("could not write initial value")
	ErrIVread  = errors.New("could not read initial value")
)
