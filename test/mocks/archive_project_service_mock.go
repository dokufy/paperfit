package mocks

import (
	"context"

	"github.com/atoncetti/combiner/internal/service"
	"github.com/atoncetti/combiner/internal/service/projectprovider"
	"github.com/atoncetti/combiner/test"
	"github.com/stretchr/testify/mock"
)

type ArchiveProjectServiceMock struct {
	mock.Mock
}

func (m *ArchiveProjectServiceMock) GetProject(ctx context.Context, url string, headers map[string][]string) (service.Project, error) {
	m.Called(url)

	index, index_size := test.File("index.html")

	schema, schema_size := test.File("schema.json")

	empty_schema, empty_schema_size := test.File("empty_schema.json")

	index_html := service.NewFile("index.html", index, index_size)
	json_schema := service.NewFile("json_schema.json", schema, schema_size)
	empty_json_schema := service.NewFile("json_schema.json", empty_schema, empty_schema_size)

	switch url {
	case "existing_template_non_existing_schema":
		return service.NewProject([]*service.File{&index_html}), nil
	case "existing_template_existing_schema":
		return service.NewProject([]*service.File{&index_html, &json_schema}), nil
	case "existing_template_existing_empty_schema":
		return service.NewProject([]*service.File{&index_html, &empty_json_schema}), nil
	default:
		return service.Project{}, &projectprovider.RequestError{StatusCode: 404, Message: "does not exist"}
	}
}
