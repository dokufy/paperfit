package service

import (
	"bytes"
	"io"
	"slices"

	"github.com/dokufy/paperfit/internal/logger"
	"golang.org/x/net/html"
)

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
		logger.Log.Debug("Index file not found.")

		return nil
	}

	return p.files[index]
}

func (p *Project) GetJSONschemaFile() *File {
	index := slices.IndexFunc(p.files, func(f *File) bool { return f.Name() == "json_schema.json" })

	if index == -1 {
		logger.Log.Debug("JSON schema file not found.")

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

	modifiedIndex, len, err := renderHTML(doc)
	if err != nil {
		return err
	}

	index.Content = modifiedIndex
	index.size = int64(len)

	return nil
}

func buildJSscript(jsonVars string) string {
	return "window.dokufy=" + jsonVars + ";"
}

func appendScriptToHead(doc *html.Node, script string) {
	if doc == nil {
		logger.Log.Debug("No document is nil, cannot append script to head.")

		return
	}
	scriptNode := &html.Node{
		Type: html.ElementNode,
		Data: "script",
		Attr: []html.Attribute{{Key: "type", Val: "text/javascript"}},
	}

	scriptText := &html.Node{
		Type: html.TextNode,
		Data: script,
	}

	scriptNode.AppendChild(scriptText)

	appendScript(doc, scriptNode)
}

func appendScript(node *html.Node, script *html.Node) {
	if node == nil || script == nil {
		return
	}

	if node.Type == html.ElementNode && node.Data == "head" {
		node.AppendChild(script)

		return
	}

	for c := node.FirstChild; c != nil; c = c.NextSibling {
		appendScript(c, script)
	}
}

func renderHTML(doc *html.Node) (io.Reader, int, error) {
	var buf bytes.Buffer
	if err := html.Render(&buf, doc); err != nil {
		return nil, 0, err
	}

	return &buf, buf.Len(), nil
}

func (p *Project) AppendFiles(files ...*File) {
	p.files = append(p.files, files...)
}
