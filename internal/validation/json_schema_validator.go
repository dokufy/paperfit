package validation

import (
	"bytes"
	"io"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

func Validate(document string, schema io.Reader) (io.Reader, error) {
	var buf bytes.Buffer
	tee := io.TeeReader(schema, &buf)

	schemaDoc, err := jsonschema.UnmarshalJSON(tee)
	if err != nil {
		return nil, err
	}

	c := jsonschema.NewCompiler()

	if err := c.AddResource("schema.json", schemaDoc); err != nil {
		return nil, err
	}

	compiled, err := c.Compile("schema.json")
	if err != nil {
		return nil, err
	}

	doc, err := jsonschema.UnmarshalJSON(strings.NewReader(document))
	if err != nil {
		return &buf, err
	}

	return &buf, compiled.Validate(doc)
}
