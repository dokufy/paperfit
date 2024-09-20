package test

import (
	"io"
	"testing"
)

func StringBody(t *testing.T, body io.Reader) string {
	r, err := io.ReadAll(body)
	if err != nil {
		t.Fatal(err)
	}

	return string(r)
}
