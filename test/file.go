package test

import (
	"bytes"
	"io"
	"os"
	"path/filepath"

	"github.com/atoncetti/combiner/config"
)

func File(relPath string) (io.Reader, int64) {
	path := FilePath(relPath)

	content, err := os.ReadFile(path)
	if err != nil {
		panic("failed to read file")
	}

	return bytes.NewReader(content), int64(len(content))
}

func FilePath(relPath string) string {
	config.Load()

	return filepath.Join(config.AppConfig.BasePath, "test/files", relPath)
}
