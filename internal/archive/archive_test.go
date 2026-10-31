package archive

import (
	"bytes"
	"context"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/dokufy/paperfit/internal/logger"
	"github.com/dokufy/paperfit/internal/service"
)

func TestMain(m *testing.M) {
	logger.Initialize()

	os.Exit(m.Run())
}

func TestExtractToProject(t *testing.T) {
	file, err := os.Open("../../test/files/archive/valid/simple.tar.gz")
	if err != nil {
		t.Fatalf("failed to open archive: %v", err)
	}
	defer file.Close()

	project, err := ExtractToProject(context.Background(), "simple.tar.gz", file)
	if err != nil {
		t.Fatalf("failed to extract archive: %v", err)
	}

	files := project.GetFiles()
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}

	if files[0].Name() != "index.html" {
		t.Errorf("expected index.html, got %q", files[0].Name())
	}

	content, err := io.ReadAll(files[0].Content)
	if err != nil {
		t.Fatalf("failed to read extracted file: %v", err)
	}

	if !bytes.Contains(content, []byte("<html")) {
		t.Errorf("expected extracted html content, got %q", content)
	}
}

func TestCompressArchiveProjectRoundTrip(t *testing.T) {
	const html = "<html><body>hello</body></html>"

	file := service.NewFile("index.html", strings.NewReader(html), int64(len(html)))
	project := service.NewProject([]*service.File{&file})

	compressed, err := CompressArchiveProject(context.Background(), &project)
	if err != nil {
		t.Fatalf("failed to compress project: %v", err)
	}

	extracted, err := ExtractToProject(context.Background(), "compressed", compressed.Content)
	if err != nil {
		t.Fatalf("failed to extract compressed project: %v", err)
	}

	files := extracted.GetFiles()
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}

	if files[0].Name() != "index.html" {
		t.Errorf("expected index.html, got %q", files[0].Name())
	}

	content, err := io.ReadAll(files[0].Content)
	if err != nil {
		t.Fatalf("failed to read extracted file: %v", err)
	}

	if string(content) != html {
		t.Errorf("unexpected extracted content: %q", content)
	}
}
