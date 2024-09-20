package validation

import (
	"bytes"
	"io"
	"strings"

	"github.com/santhosh-tekuri/jsonschema"
)

func Validate(json string, schema io.Reader) (io.Reader, error) {
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

	r := strings.NewReader(string(json))

	return &buf, sch.Validate(r)
}
