package archive

import "errors"

var (
	ErrTimeoutExtracting  = errors.New("timeout while extracting archive into project")
	ErrTimeoutCompressing = errors.New("timeout while compressing project into archive")
)
