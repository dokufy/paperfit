package archiveurlgen_test

import (
	"context"
	"testing"

	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/dokufy/paperfit/config"
	"github.com/dokufy/paperfit/internal/cache"
	"github.com/dokufy/paperfit/internal/httptool"
	"github.com/dokufy/paperfit/internal/server/operation/archiveurlgen"
	"github.com/dokufy/paperfit/internal/service/pdfgenerator"
	"github.com/dokufy/paperfit/test/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type CachedTestSuite struct {
	suite.Suite
	ctx           context.Context
	api           humatest.TestAPI
	cacheInstance *cache.Cache
	assert        *assert.Assertions
}

func (s *CachedTestSuite) SetupSuite() {
	config.Load()
	mocks.PassPDFgenerator()

	archiveProjectServiceMock := new(mocks.ArchiveProjectServiceMock)
	archiveProjectServiceMock.On("GetProject", mock.Anything).Return()

	pdfProvider := pdfgenerator.NewHTTPpdfGenerator(config.Config.PDFgeneratorEndpoint)

	cacheInstance, err := cache.New(config.Config.CacheConfig.CacheUrl)
	if err != nil {
		s.T().Fatal("failed to create cache")
	}

	_, api := humatest.New(s.T())

	archiveurlgen.RegisterOperation(api, "/pdf/generate/url", archiveProjectServiceMock, pdfProvider, cacheInstance)

	s.ctx = context.Background()
	s.api = api
	s.cacheInstance = cacheInstance
	s.assert = assert.New(s.T())
}

func (s *CachedTestSuite) Test_Success_Generate_200() {
	resp := s.api.Post(
		"/pdf/generate/url",
		map[string]any{
			"url": "existing_template_non_existing_schema",
		})

	s.assert.Equal(200, resp.Code)

	cacheKey, _ := cache.Key([][]byte{[]byte("existing_template_non_existing_schema")})

	reader, err := s.cacheInstance.GetPDF(s.ctx, cacheKey)
	if err != nil {
		s.T().Fatal(err)
	}

	assert.NotEmpty(s.T(), reader)

	s.cacheInstance.DeletePDF(s.ctx, cacheKey)
}

func (s *CachedTestSuite) Test_Success_Generate_With_Variables_200() {
	variables := `{"title": "this is a title"}`
	resp := s.api.Post(
		"/pdf/generate/url",
		map[string]any{
			"url":       "existing_template_existing_schema",
			"variables": variables,
		},
	)

	s.assert.Equal(200, resp.Code)

	cacheKey, _ := cache.Key([][]byte{[]byte("existing_template_existing_schema"), []byte(variables)})

	reader, err := s.cacheInstance.GetPDF(s.ctx, cacheKey)
	if err != nil {
		s.T().Fatal(err)
	}

	assert.NotEmpty(s.T(), reader)

	s.cacheInstance.DeletePDF(s.ctx, cacheKey)
}

func (s *CachedTestSuite) Test_Success_Generate_With_Validation_And_Variables_200() {
	variables := `{"title": "this is a title"}`
	resp := s.api.Post(
		"/pdf/generate/url",
		map[string]any{
			"url":       "existing_template_existing_schema",
			"variables": variables,
			"validate":  true,
		},
	)

	s.assert.Equal(200, resp.Code)

	cacheKey, _ := cache.Key([][]byte{[]byte("existing_template_existing_schema"), []byte(variables)})

	reader, err := s.cacheInstance.GetPDF(s.ctx, cacheKey)
	if err != nil {
		s.T().Fatal(err)
	}

	assert.NotEmpty(s.T(), reader)

	s.cacheInstance.DeletePDF(s.ctx, cacheKey)
}

func (s *CachedTestSuite) Test_Success_Generate_Use_Cache_200() {
	headers := map[string][]string{
		"Authorization": {"Bearer test"},
		"Content-Type":  {"application/pdf"},
	}
	variables := `{"title": "this is a title"}`

	resp1 := s.api.Post(
		"/pdf/generate/url",
		map[string]any{
			"url":       "existing_template_non_existing_schema",
			"headers":   headers,
			"variables": variables,
		})

	s.assert.Equal(200, resp1.Code)

	cacheKey, _ := cache.Key([][]byte{[]byte("existing_template_non_existing_schema"), httptool.FlattenHeaders(headers), []byte(variables)})

	reader, err := s.cacheInstance.GetPDF(s.ctx, cacheKey)
	if err != nil {
		s.T().Fatal(err)
	}

	assert.NotEmpty(s.T(), reader)

	resp2 := s.api.Post(
		"/pdf/generate/url",
		map[string]any{
			"url":       "existing_template_non_existing_schema",
			"headers":   headers,
			"variables": variables,
		})

	s.assert.Equal(200, resp2.Code)

	cacheKey, _ = cache.Key([][]byte{[]byte("existing_template_non_existing_schema"), httptool.FlattenHeaders(headers), []byte(variables)})

	reader, err = s.cacheInstance.GetPDF(s.ctx, cacheKey)
	if err != nil {
		s.T().Fatal(err)
	}

	assert.NotEmpty(s.T(), reader)

	s.cacheInstance.DeletePDF(s.ctx, cacheKey)
}

func (s *CachedTestSuite) Test_Success_Generate_Different_Headers_200() {
	headers1 := map[string][]string{
		"Authorization": {"Bearer test"},
		"Content-Type":  {"application/pdf"},
	}

	resp1 := s.api.Post(
		"/pdf/generate/url",
		map[string]any{
			"url":     "existing_template_non_existing_schema",
			"headers": headers1,
		})

	s.assert.Equal(200, resp1.Code)

	cacheKey1, err := cache.Key([][]byte{[]byte("existing_template_non_existing_schema"), httptool.FlattenHeaders(headers1)})

	reader, err := s.cacheInstance.GetPDF(s.ctx, cacheKey1)
	if err != nil {
		s.T().Fatal(err)
	}

	assert.NotEmpty(s.T(), reader)

	headers2 := map[string][]string{
		"Authorization": {"Bearer newtest"},
		"Content-Type":  {"application/pdf"},
	}

	resp2 := s.api.Post(
		"/pdf/generate/url",
		map[string]any{
			"url":     "existing_template_non_existing_schema",
			"headers": headers2,
		})

	s.assert.Equal(200, resp2.Code)

	cacheKey2, _ := cache.Key([][]byte{[]byte("existing_template_non_existing_schema"), httptool.FlattenHeaders(headers2)})

	reader, err = s.cacheInstance.GetPDF(s.ctx, cacheKey2)
	if err != nil {
		s.T().Fatal(err)
	}

	assert.NotEmpty(s.T(), reader)

	s.cacheInstance.DeletePDF(s.ctx, cacheKey1)
	s.cacheInstance.DeletePDF(s.ctx, cacheKey2)
}

func (s *CachedTestSuite) Test_Success_Generate_Different_Variables_200() {
	variables1 := `{"title": "this is a title"}`

	resp1 := s.api.Post(
		"/pdf/generate/url",
		map[string]any{
			"url":       "existing_template_non_existing_schema",
			"variables": variables1,
		})

	s.assert.Equal(200, resp1.Code)

	cacheKey1, err := cache.Key([][]byte{[]byte("existing_template_non_existing_schema"), []byte(variables1)})

	reader, err := s.cacheInstance.GetPDF(s.ctx, cacheKey1)
	if err != nil {
		s.T().Fatal(err)
	}

	assert.NotEmpty(s.T(), reader)

	variables2 := `{"title": "new title"}`

	resp2 := s.api.Post(
		"/pdf/generate/url",
		map[string]any{
			"url":       "existing_template_non_existing_schema",
			"variables": variables2,
		})

	s.assert.Equal(200, resp2.Code)

	cacheKey2, _ := cache.Key([][]byte{[]byte("existing_template_non_existing_schema"), []byte(variables2)})

	reader, err = s.cacheInstance.GetPDF(s.ctx, cacheKey2)
	if err != nil {
		s.T().Fatal(err)
	}

	assert.NotEmpty(s.T(), reader)

	s.cacheInstance.DeletePDF(s.ctx, cacheKey1)
	s.cacheInstance.DeletePDF(s.ctx, cacheKey2)
}

func (s *CachedTestSuite) Test_Fail_Generate_No_Url_400() {
	resp := s.api.Post("/pdf/generate/url", map[string]any{
		"variables": `{"test": "some_value"}`,
	})

	s.assert.Equal(400, resp.Code)
	s.assert.Contains(resp.Body.String(), "No url provided")
}

func (s *CachedTestSuite) Test_Fail_Generate_Bad_Url_404() {
	resp := s.api.Post("/pdf/generate/url", map[string]any{
		"url":       "non_existing",
		"variables": `{"test": "some_value"}`,
	})

	s.assert.Equal(404, resp.Code)
	s.assert.Contains(resp.Body.String(), "Archive not found")
}

func (s *CachedTestSuite) Test_Fail_Generate_With_Validation_No_Variables_400() {
	resp := s.api.Post("/pdf/generate/url", map[string]any{
		"url":      "existing_template_existing_schema",
		"validate": true,
	})

	s.assert.Equal(400, resp.Code)
	s.assert.Contains(resp.Body.String(), "Provided variables do not conform to the json schema")
}

func (s *CachedTestSuite) Test_Fail_Generate_With_Validation_And_Variables_Empty_Schema_400() {
	resp := s.api.Post("/pdf/generate/url", map[string]any{
		"url":       "existing_template_existing_empty_schema",
		"variables": `{"test": "some_value"}`,
		"validate":  true,
	})

	s.assert.Equal(400, resp.Code)
	s.assert.Contains(resp.Body.String(), "No json schema found in the archive, please make sure that a json schema exists and it is not empty or disable variables validation")

	cacheKey, _ := cache.Key([][]byte{[]byte("existing_template_existing_empty_schema")})

	reader, err := s.cacheInstance.GetPDF(s.ctx, cacheKey)
	if err != nil {
		s.T().Fatal(err)
	}

	s.assert.Empty(reader)
}

func (s *CachedTestSuite) Test_Fail_Generate_Bad_Variables_400() {
	resp := s.api.Post("/pdf/generate/url", map[string]any{
		"url":       "existing_template_existing_schema",
		"variables": "bad variables",
		"validate":  true,
	})

	s.assert.Equal(400, resp.Code)
	s.assert.Contains(resp.Body.String(), "Failed to parse variables")
}

func (s *CachedTestSuite) Test_Fail_Generate_Variables_Validation_Fails_400() {
	resp := s.api.Post("/pdf/generate/url", map[string]any{
		"url":       "existing_template_existing_schema",
		"variables": `{"wrong": "missing!!"}`,
		"validate":  true,
	})

	s.assert.Equal(400, resp.Code)
	s.assert.Contains(resp.Body.String(), "Provided variables do not conform to the json schema")

	cacheKey, _ := cache.Key([][]byte{[]byte("existing_template_existing_schema")})

	reader, err := s.cacheInstance.GetPDF(s.ctx, cacheKey)
	if err != nil {
		s.T().Fatal(err)
	}

	s.assert.Empty(reader)
}

func TestCached(t *testing.T) {
	t.Parallel()

	suite.Run(t, new(CachedTestSuite))
}
