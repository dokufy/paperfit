package mocks

import (
	"context"

	"github.com/dokufy/paperfit/internal/service"
	"github.com/dokufy/paperfit/internal/service/projectprovider"
	"github.com/dokufy/paperfit/test"
	"github.com/stretchr/testify/mock"
)

type ArchiveProjectServiceMock struct {
	mock.Mock
}

func (m *ArchiveProjectServiceMock) GetProject(ctx context.Context, url string, headers map[string][]string) (service.Project, error) {
	m.Called(url)

	index_html := test.File("index.html")

	json_schema := test.File("valid_json_schema/json_schema.json")

	empty_json_schema := test.File("empty_json_schema/json_schema.json")

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
