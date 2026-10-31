package service

import (
	"bytes"
	"io"
	"strings"
	"testing"
)

func TestAppendVariablesInjectsScriptAsFirstChildOfHead(t *testing.T) {
	html := `<html><head><meta charset="utf-8" /><title>T</title></head><body><p>Hi</p></body></html>`
	f := NewFile("index.html", bytes.NewReader([]byte(html)), int64(len(html)))
	p := NewProject([]*File{&f})

	if err := p.AppendVariables(`{"title":"Hello"}`); err != nil {
		t.Fatalf("AppendVariables failed: %v", err)
	}

	idx := p.GetIndexFile()
	if idx == nil {
		t.Fatal("index file missing")
	}
	content, err := io.ReadAll(idx.Content)
	if err != nil {
		t.Fatal(err)
	}
	s := string(content)

	if !strings.Contains(s, "window.__PAPERFIT_INPUT__={\"title\":\"Hello\"}") {
		t.Fatalf("missing injected input: %s", s)
	}
	if strings.Contains(s, "window.dokufy") {
		t.Fatalf("unexpected dokufy injection: %s", s)
	}

	// The script must appear before the meta/title inside <head>.
	headIdx := strings.Index(s, "<head>")
	if headIdx == -1 {
		t.Fatal("no head")
	}
	// find first child after <head>
	afterHead := s[headIdx+len("<head>"):]
	trimmed := strings.TrimLeft(afterHead, " \t\n\r")
	if !strings.HasPrefix(trimmed, "<script") {
		t.Fatalf("script is not first child of head: %s", s)
	}
	if !strings.Contains(trimmed, "window.__PAPERFIT_INPUT__") {
		t.Fatalf("injected script not first: %s", s)
	}
}

func TestAppendVariablesWhenHeadAbsent(t *testing.T) {
	html := `<html><body><p>Hi</p></body></html>`
	f := NewFile("index.html", bytes.NewReader([]byte(html)), int64(len(html)))
	p := NewProject([]*File{&f})

	if err := p.AppendVariables(`{"title":"Hello"}`); err != nil {
		t.Fatalf("AppendVariables failed: %v", err)
	}

	idx := p.GetIndexFile()
	content, _ := io.ReadAll(idx.Content)
	s := string(content)

	if !strings.Contains(s, "<head>") {
		t.Fatalf("head not created: %s", s)
	}
	if !strings.Contains(s, "window.__PAPERFIT_INPUT__") {
		t.Fatalf("missing injected input: %s", s)
	}
	// script should be inside head
	headStart := strings.Index(s, "<head>")
	headEnd := strings.Index(s, "</head>")
	if headStart == -1 || headEnd == -1 || headEnd < headStart {
		t.Fatalf("invalid head: %s", s)
	}
	segment := s[headStart:headEnd]
	if !strings.Contains(segment, "window.__PAPERFIT_INPUT__") {
		t.Fatalf("injected script not in head: %s", s)
	}
}
