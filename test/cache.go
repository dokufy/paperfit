package test

import (
	"io"

	"github.com/stretchr/testify/assert"
)

func AssertReaderNotEmpty(assert assert.Assertions, reader io.Reader) {
	if reader == nil {
		assert.Fail("Reader is empty")
		return
	}

	pdf, err := io.ReadAll(reader)
	if err != nil {
		assert.Fail("Failed to read from reader", err)
		return
	}

	assert.NotEmpty(pdf)
}
