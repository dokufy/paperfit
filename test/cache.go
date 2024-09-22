package test

import (
	"io"

	"github.com/stretchr/testify/assert"
)

func AssertReaderNotEmpty(assert *assert.Assertions, reader io.Reader) {
	if reader == nil {
		assert.Fail("Reader is empty")
		return
	}

	buf := make([]byte, 1)

	cont, err := io.ReadAtLeast(reader, buf, 1)
	if err != nil {
		assert.Fail("Failed to read from empty reader", err)
		return
	}

	assert.NotEmpty(cont)
}
