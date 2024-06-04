package service

import (
	"bytes"
	"io"
	"slices"
)

type Project struct {
	files []*File
}

func NewProject(files []*File) Project {
	return Project{
		files,
	}
}

func (p *Project) GetTemplate() *File {
	index := slices.IndexFunc(p.files, func(f *File) bool { return f.Name() == "index.html" })

	if index == -1 {
		return nil
	}

	return p.files[index]
}

func (p *Project) GetJSONschema() *File {
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

func (p *Project) AppendArgumentsFile(arguments string) error {
	content := []byte("window.test = " + arguments + ";\n")
	size := int64(len(content))
	file := NewFile("arguments.js", bytes.NewReader(content), size)

	p.AppendFiles(&file)

	return nil
}

func (p *Project) AppendFiles(files ...*File) {
	p.files = append(p.files, files...)
}
