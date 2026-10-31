package validation

import (
	"io"
	"strings"
	"testing"
)

const testSchema = `{
	"type": "object",
	"required": ["title"],
	"properties": {
		"title": {"type": "string"}
	}
}`

func TestValidate(t *testing.T) {
	t.Run("valid document", func(t *testing.T) {
		schema, err := Validate(`{"title": "a title"}`, strings.NewReader(testSchema))
		if err != nil {
			t.Fatalf("expected document to be valid, got %v", err)
		}

		content, err := io.ReadAll(schema)
		if err != nil {
			t.Fatalf("failed to read schema: %v", err)
		}

		if string(content) != testSchema {
			t.Errorf("expected schema to be returned unchanged, got %q", content)
		}
	})

	t.Run("invalid document", func(t *testing.T) {
		if _, err := Validate(`{"title": 42}`, strings.NewReader(testSchema)); err == nil {
			t.Fatal("expected document to be invalid")
		}
	})

	t.Run("invalid document json", func(t *testing.T) {
		if _, err := Validate(`not json`, strings.NewReader(testSchema)); err == nil {
			t.Fatal("expected invalid json to be rejected")
		}
	})

	t.Run("empty schema", func(t *testing.T) {
		if _, err := Validate(`{"title": "a title"}`, strings.NewReader("")); err == nil {
			t.Fatal("expected empty schema to be rejected")
		}
	})
}
