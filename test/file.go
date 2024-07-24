package test

import (
	"bytes"
	"io"
	"os"
	"path/filepath"

	"github.com/atoncetti/combiner/config"
	"github.com/atoncetti/combiner/internal/service"
)

func File(relPath string) service.File {
	archive, archive_size := getFile(relPath)

	return service.NewFile(filepath.Base(relPath), archive, archive_size)
}

func getFile(relPath string) (io.Reader, int64) {
	path := FilePath(relPath)

	content, err := os.ReadFile(path)
	if err != nil {
		panic("failed to read file")
	}

	return bytes.NewReader(content), int64(len(content))
}

func FilePath(relPath string) string {
	config.Load()

	return filepath.Join(config.Config.BasePath, "test/files", relPath)
}
