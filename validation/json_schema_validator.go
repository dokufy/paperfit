package validation

import (
	"io"

	"github.com/santhosh-tekuri/jsonschema"
)

func Validate(data io.Reader, schema io.Reader) error {
	c := jsonschema.NewCompiler()

	err := c.AddResource("schema.json", schema)
	if err != nil {
		return err
	}

	sch, err := c.Compile("schema.json")
	if err != nil {
		return err
	}

	return sch.Validate(data)
}
