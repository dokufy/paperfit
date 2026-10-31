package test

import (
	"bytes"
	"io"
	"os"
	"path/filepath"

	"github.com/dokufy/paperfit/config"
	"github.com/dokufy/paperfit/internal/service"
	"github.com/ledongthuc/pdf"
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

func PdfToString(r io.Reader) string {
	data, err := io.ReadAll(r)
	if err != nil {
		panic(err)
	}

	reader, err := pdf.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		panic(err)
	}

	text, err := reader.GetPlainText()
	if err != nil {
		panic(err)
	}

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, text); err != nil {
		panic(err)
	}

	return buf.String()
}
