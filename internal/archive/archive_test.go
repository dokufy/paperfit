package archive

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
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
	defer func() { _ = file.Close() }()

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

func TestExtractToProjectNestedAndRewritesPaths(t *testing.T) {
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)

	files := map[string]string{
		"assets/app.js": "console.log('app');",
		"css/style.css": "body { color: red; }",
		"index.html":    `<!DOCTYPE html><html><head><link rel="stylesheet" href="./css/style.css"><script src="assets/app.js"></script></head><body><p>hi</p></body></html>`,
	}
	// write nested dirs implicitly via names
	for name, content := range files {
		hdr := &tar.Header{
			Name: name,
			Mode: 0o644,
			Size: int64(len(content)),
		}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	_ = tw.Close()
	_ = gw.Close()

	project, err := ExtractToProject(context.Background(), "nested.tar.gz", bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatalf("extract failed: %v", err)
	}

	gotFiles := map[string]bool{}
	for _, f := range project.GetFiles() {
		gotFiles[f.Name()] = true
	}
	for _, want := range []string{"index.html", "app.js", "style.css"} {
		if !gotFiles[want] {
			t.Fatalf("missing flattened file %s; got %v", want, gotFiles)
		}
	}

	// check rewritten index.html
	var indexContent string
	for _, f := range project.GetFiles() {
		if f.Name() == "index.html" {
			b, _ := io.ReadAll(f.Content)
			indexContent = string(b)
		}
	}
	if !strings.Contains(indexContent, `href="style.css"`) {
		t.Fatalf("css path not rewritten: %s", indexContent)
	}
	if !strings.Contains(indexContent, `src="app.js"`) {
		t.Fatalf("js path not rewritten: %s", indexContent)
	}
}

func TestExtractToProjectCollidingBasenames(t *testing.T) {
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)

	files := map[string]string{
		"a/style.css": "body { color: red; }",
		"b/style.css": "body { color: blue; }",
		"index.html":  `<!DOCTYPE html><html><head><link rel="stylesheet" href="a/style.css"><link rel="stylesheet" href="b/style.css"></head><body></body></html>`,
	}
	for name, content := range files {
		hdr := &tar.Header{Name: name, Mode: 0o644, Size: int64(len(content))}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatal(err)
		}
		_, _ = tw.Write([]byte(content))
	}
	_ = tw.Close()
	_ = gw.Close()

	project, err := ExtractToProject(context.Background(), "collide.tar.gz", bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatalf("extract failed: %v", err)
	}

	names := map[string]bool{}
	for _, f := range project.GetFiles() {
		names[f.Name()] = true
	}
	if !names["style.css"] {
		t.Fatalf("missing style.css: %v", names)
	}
	if !names["style_1.css"] {
		t.Fatalf("missing style_1.css: %v", names)
	}

	var indexContent string
	for _, f := range project.GetFiles() {
		if f.Name() == "index.html" {
			b, _ := io.ReadAll(f.Content)
			indexContent = string(b)
		}
	}
	if !strings.Contains(indexContent, "style.css") || !strings.Contains(indexContent, "style_1.css") {
		t.Fatalf("references not rewritten: %s", indexContent)
	}
}

func TestCompressArchiveProjectIdempotent(t *testing.T) {
	const html = "<html><body>hello</body></html>"
	file := service.NewFile("index.html", strings.NewReader(html), int64(len(html)))
	project := service.NewProject([]*service.File{&file})

	c1, err := CompressArchiveProject(context.Background(), &project)
	if err != nil {
		t.Fatal(err)
	}
	c2, err := CompressArchiveProject(context.Background(), &project)
	if err != nil {
		t.Fatalf("second compress failed: %v", err)
	}

	for _, c := range []service.File{c1, c2} {
		extracted, err := ExtractToProject(context.Background(), "compressed", c.Content)
		if err != nil {
			t.Fatal(err)
		}
		if len(extracted.GetFiles()) != 1 {
			t.Fatalf("expected 1 file, got %d", len(extracted.GetFiles()))
		}
		b, _ := io.ReadAll(extracted.GetFiles()[0].Content)
		if string(b) != html {
			t.Fatalf("content mismatch: %q", string(b))
		}
	}
}
