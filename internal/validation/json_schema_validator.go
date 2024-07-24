package validation

import (
	"bytes"
	"io"

	"github.com/santhosh-tekuri/jsonschema"
)

func Validate(data io.Reader, schema io.Reader) (io.Reader, error) {
	var buf bytes.Buffer
	tee := io.TeeReader(schema, &buf)
	c := jsonschema.NewCompiler()

	err := c.AddResource("schema.json", tee)
	if err != nil {
		return nil, err
	}

	sch, err := c.Compile("schema.json")
	if err != nil {
		return nil, err
	}

	return &buf, sch.Validate(data)
}
