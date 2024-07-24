package archiveurlgen_test

import (
	"context"
	"testing"

	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/dokufy/paperfit/config"
	"github.com/dokufy/paperfit/internal/cache"
	"github.com/dokufy/paperfit/internal/server/operation/archiveurlgen"
	"github.com/dokufy/paperfit/internal/service/pdfgenerator"
	"github.com/dokufy/paperfit/test/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type TestSuite struct {
	suite.Suite
	ctx           context.Context
	api           humatest.TestAPI
	cacheInstance *cache.Cache
	assert        *assert.Assertions
}

func (s *TestSuite) SetupSuite() {
	config.Load()
	mocks.PassPDFgenerator()

	archiveProjectServiceMock := new(mocks.ArchiveProjectServiceMock)
	archiveProjectServiceMock.On("GetProject", mock.Anything).Return()

	_, api := humatest.New(s.T())

	pdfProvider := pdfgenerator.NewHTTPpdfGenerator(config.Config.PDFgeneratorEndpoint)
	archiveurlgen.RegisterOperation(api, "/pdf/generate/url", archiveProjectServiceMock, pdfProvider, nil)

	s.ctx = context.Background()
	s.api = api
	s.assert = assert.New(s.T())
}

func (s *TestSuite) Test_Success_Generate_200() {
	resp := s.api.Post(
		"/pdf/generate/url",
		map[string]any{
			"url": "existing_template_non_existing_schema",
		})

	s.assert.Equal(200, resp.Code)
}

func (s *TestSuite) Test_Success_Generate_With_Variables_200() {
	resp := s.api.Post(
		"/pdf/generate/url",
		map[string]any{
			"url":       "existing_template_non_existing_schema",
			"variables": `{"title": "this is a title"}`,
		})

	s.assert.Equal(200, resp.Code)
}

func (s *TestSuite) Test_Generate_With_Validation_And_Variables_200() {
	resp := s.api.Post(
		"/pdf/generate/url",
		map[string]any{
			"url":       "existing_template_existing_schema",
			"variables": `{"title": "this is a title"}`,
			"validate":  true,
		},
	)

	s.assert.Equal(200, resp.Code)
}

func (s *TestSuite) Test_Success_Generate_With_Variables_Empty_Schema_200() {
	resp := s.api.Post("/pdf/generate/url", map[string]any{
		"url":       "existing_template_existing_empty_schema",
		"variables": `{"test": "some_value"}`,
	})

	s.assert.Equal(200, resp.Code)
}

func (s *TestSuite) Test_Fail_Generate_No_Url_400() {
	resp := s.api.Post("/pdf/generate/url", map[string]any{})

	s.assert.Equal(400, resp.Code)
	s.assert.Contains(resp.Body.String(), "No url provided")
}

func (s *TestSuite) Test_Fail_Generate_Bad_Url_404() {
	resp := s.api.Post("/pdf/generate/url", map[string]any{
		"url": "non_existing",
	})

	s.assert.Equal(404, resp.Code)
	s.assert.Contains(resp.Body.String(), "Archive not found")
}

func (s *TestSuite) Test_Fail_Generate_Validation_And_Variables_Empty_Schema_400() {
	resp := s.api.Post("/pdf/generate/url", map[string]any{
		"url":       "existing_template_non_existing_schema",
		"variables": `{"test": "some_value"}`,
		"validate":  true,
	})

	s.assert.Equal(400, resp.Code)
	s.assert.Contains(resp.Body.String(), "No json schema found in the archive, please make sure that a json schema exists and it is not empty or disable variables validation")
}

func (s *TestSuite) Test_Fail_Generate_Bad_Variables_400() {
	resp := s.api.Post("/pdf/generate/url", map[string]any{
		"url":       "existing_template_existing_schema",
		"variables": "bad variables",
		"validate":  true,
	})

	s.assert.Equal(400, resp.Code)
	s.assert.Contains(resp.Body.String(), "Failed to parse variables")
}

func (s *TestSuite) Test_Fail_Generate_Variables_Validation_Fails_400() {
	resp := s.api.Post("/pdf/generate/url", map[string]any{
		"url":       "existing_template_existing_schema",
		"variables": `{"test": "some_value"}`,
		"validate":  true,
	})

	s.assert.Equal(400, resp.Code)
	s.assert.Contains(resp.Body.String(), "Provided variables do not conform to the json schema")
}

func TestSuccessfulGeneration(t *testing.T) {
	t.Parallel()

	suite.Run(t, new(TestSuite))
}
