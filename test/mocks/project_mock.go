package mocks

import (
	"strings"

	"github.com/atoncetti/combiner/internal/service"
)

func ProjectMock(content string) service.Project {
	var sb strings.Builder

	sb.WriteString("<!DOCTYPE html><html><body><div style=\"width: 100%; word-break: break-all;\"><p>")
	sb.WriteString(content)
	sb.WriteString("</p></div></body></html>")

	index := strings.NewReader(sb.String())
	index_size := int64(sb.Len())

	template := service.NewFile("index.html", index, index_size)

	return service.NewProject([]*service.File{&template})
}

func ProjectFileMock() service.File {
	content := "test"
	index := strings.NewReader(content)
	index_size := int64(len(content))

	return service.NewFile("index.html", index, index_size)
}
