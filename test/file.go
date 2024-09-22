package test

import (
	"bytes"
	"io"
	"os"
	"path/filepath"

	"code.sajari.com/docconv/v2"
	"github.com/dokufy/paperfit/config"
	"github.com/dokufy/paperfit/internal/service"
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

	return filepath.Join(config.AppConfig.BasePath, "test/files", relPath)
}

func PdfToString(pdf io.Reader) string {
	res, _, err := docconv.ConvertPDF(pdf)
	if err != nil {
		panic(err)
	}

	return res
}
