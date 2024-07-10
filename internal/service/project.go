package service

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"slices"

	"golang.org/x/net/html"
)

var ErrIndexFileMissing = errors.New("index.html is required")

type Project struct {
	files []*File
}

func NewProject(files []*File) Project {
	return Project{
		files,
	}
}

func (p *Project) GetIndexFile() *File {
	index := slices.IndexFunc(p.files, func(f *File) bool { return f.Name() == "index.html" })

	if index == -1 {
		return nil
	}

	return p.files[index]
}

func (p *Project) GetJSONschemaFile() *File {
	index := slices.IndexFunc(p.files, func(f *File) bool { return f.Name() == "json_schema.json" })

	if index == -1 {
		return nil
	}

	return p.files[index]
}

func (p *Project) GetFiles() []*File {
	return p.files
}

func (p *Project) AppendFile(name string, content io.Reader, size int64) error {
	file := NewFile(name, content, size)
	p.AppendFiles(&file)

	return nil
}

func (p *Project) AppendVariables(variables string) error {
	if variables == "" {
		return nil
	}

	var parsedVars map[string]any
	err := json.Unmarshal([]byte(variables), &parsedVars)
	if err != nil {
		return err
	}

	index := p.GetIndexFile()

	if index == nil {
		return ErrIndexFileMissing
	}

	doc, err := html.Parse(index.Content)
	if err != nil {
		return err
	}

	script := buildJSscript(variables)
	appendScriptToHead(doc, script)

	modifiedIndex, err := renderHTML(doc)
	if err != nil {
		return err
	}

	index.Content = modifiedIndex

	return nil
}

func buildJSscript(JSONvariables string) string {
	return "window.dokufy=" + JSONvariables + ";"
}

func appendScriptToHead(doc *html.Node, script string) {
	scriptNode := &html.Node{
		Type: html.ElementNode,
		Data: "script",
	}

	scriptText := &html.Node{
		Type: html.TextNode,
		Data: script,
	}

	scriptNode.AppendChild(scriptText)

	var traverse func(*html.Node)

	traverse = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "head" {
			n.AppendChild(scriptNode)
			return
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			traverse(c)
		}
	}

	traverse(doc)
}

func renderHTML(doc *html.Node) (io.Reader, error) {
	var buf bytes.Buffer
	if err := html.Render(&buf, doc); err != nil {
		return nil, err
	}

	return &buf, nil
}

func (p *Project) AppendFiles(files ...*File) {
	p.files = append(p.files, files...)
}
