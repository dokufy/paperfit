package archiveurlgen_test

import (
	"context"
	"testing"

	"github.com/atoncetti/combiner/config"
	"github.com/atoncetti/combiner/internal/cache"
	"github.com/atoncetti/combiner/internal/httptool"
	"github.com/atoncetti/combiner/internal/server/operation/archiveurlgen"
	"github.com/atoncetti/combiner/test/mocks"
	"github.com/danielgtaylor/huma/v2/humatest"
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
	cacheInstance, err := cache.New(config.Config.CacheConfig.CacheUrl)
	if err != nil {
		s.T().Fatal("failed to create cache")
	}

	_, api := humatest.New(s.T())

	s.ctx = context.Background()
	s.api = api
	s.cacheInstance = cacheInstance
	s.assert = assert.New(s.T())
}

func (s *CachedTestSuite) Test_Success_Generate_PDF_With_Cache_Returns_200() {
	archiveProjectServiceMock := new(mocks.ArchiveProjectServiceMock)
	pdfServiceMock := new(mocks.PDFserviceMock)

	archiveProjectServiceMock.On("GetProject", mock.Anything).Return()
	pdfServiceMock.On("Generate", mock.Anything).Return()

	archiveurlgen.RegisterOperation(s.api, "/pdf/generate/archive", archiveProjectServiceMock, pdfServiceMock, s.cacheInstance)

	resp1 := s.api.Post(
		"/pdf/generate/archive",
		map[string]any{
			"url": "existing_template_non_existing_schema",
		})

	s.assert.Equal(200, resp1.Code)

	resp2 := s.api.Post(
		"/pdf/generate/archive",
		map[string]any{
			"url": "existing_template_non_existing_schema",
		})

	s.assert.Equal(200, resp2.Code)

	// TODO: Figure out because https://github.com/stretchr/testify/issues/944
	// archiveProjectServiceMock.AssertExpectations(s.T())
	// pdfServiceMock.AssertExpectations(s.T())

	cacheKey, err := cache.Key([][]byte{[]byte("existing_template_non_existing_schema")})
	if err != nil {
		s.T().Fatal(err)
	}

	s.cacheInstance.DeletePDF(s.ctx, cacheKey)
}

func (s *CachedTestSuite) Test_Success_Generate_PDF_With_Cache_Different_Headers_Returns_200() {
	archiveProjectServiceMock := new(mocks.ArchiveProjectServiceMock)
	pdfServiceMock := new(mocks.PDFserviceMock)

	archiveProjectServiceMock.On("GetProject", mock.Anything).Return()
	pdfServiceMock.On("Generate", mock.Anything).Return()

	archiveurlgen.RegisterOperation(s.api, "/pdf/generate/archive", archiveProjectServiceMock, pdfServiceMock, s.cacheInstance)

	headers1 := map[string][]string{
		"Authorization": {"Bearer test"},
		"Content-Type":  {"application/pdf"},
	}

	resp1 := s.api.Post(
		"/pdf/generate/archive",
		map[string]any{
			"url":     "existing_template_non_existing_schema",
			"headers": headers1,
		})

	s.assert.Equal(200, resp1.Code)

	headers2 := map[string][]string{
		"Authorization": {"Bearer newtest"},
		"Content-Type":  {"application/pdf"},
	}

	resp2 := s.api.Post(
		"/pdf/generate/archive",
		map[string]any{
			"url":     "existing_template_non_existing_schema",
			"headers": headers2,
		})

	s.assert.Equal(200, resp2.Code)

	// TODO: Figure out because https://github.com/stretchr/testify/issues/944
	// archiveProjectServiceMock.AssertExpectations(s.T())
	// pdfServiceMock.AssertExpectations(s.T())

	cacheKey1, err := cache.Key([][]byte{[]byte("existing_template_non_existing_schema"), httptool.FlattenHeaders(headers1)})
	if err != nil {
		s.T().Fatal(err)
	}

	cacheKey2, err := cache.Key([][]byte{[]byte("existing_template_non_existing_schema"), httptool.FlattenHeaders(headers1)})
	if err != nil {
		s.T().Fatal(err)
	}

	s.cacheInstance.DeletePDF(s.ctx, cacheKey1)
	s.cacheInstance.DeletePDF(s.ctx, cacheKey2)
}

func (s *CachedTestSuite) Test_Success_Generate_PDF_With_Cache_With_Validate_Arguments_Returns_200() {
	archiveProjectServiceMock := new(mocks.ArchiveProjectServiceMock)
	pdfServiceMock := new(mocks.PDFserviceMock)

	archiveProjectServiceMock.On("GetProject", mock.Anything).Return()
	pdfServiceMock.On("Generate", mock.Anything).Return()

	archiveurlgen.RegisterOperation(s.api, "/pdf/generate/archive", archiveProjectServiceMock, pdfServiceMock, s.cacheInstance)

	resp1 := s.api.Post(
		"/pdf/generate/archive",
		map[string]any{
			"url": "existing_template_existing_schema",
			"argumentOptions": map[string]any{
				"variables": "{\"title\": \"this is a title\"}",
				"validate":  true,
			},
		},
	)

	s.assert.Equal(200, resp1.Code)

	resp2 := s.api.Post(
		"/pdf/generate/archive",
		map[string]any{
			"url": "existing_template_existing_schema",
			"argumentOptions": map[string]any{
				"variables": "{\"title\": \"this is a title\"}",
				"validate":  true,
			},
		},
	)

	s.assert.Equal(200, resp2.Code)

	// TODO: Figure out because https://github.com/stretchr/testify/issues/944
	// archiveProjectServiceMock.AssertExpectations(s.T())
	// pdfServiceMock.AssertExpectations(s.T())

	cacheKey, err := cache.Key([][]byte{[]byte("existing_template_existing_schema")})
	if err != nil {
		s.T().Fatal(err)
	}

	s.cacheInstance.DeletePDF(s.ctx, cacheKey)
}

func (s *CachedTestSuite) Test_Success_Generate_PDF_With_Cache_With_Validate_Arguments_Empty_Schema_Returns_200() {
	archiveProjectServiceMock := new(mocks.ArchiveProjectServiceMock)
	pdfServiceMock := new(mocks.PDFserviceMock)

	archiveProjectServiceMock.On("GetProject", mock.Anything).Return()
	pdfServiceMock.On("Generate", mock.Anything).Return()

	archiveurlgen.RegisterOperation(s.api, "/pdf/generate/archive", archiveProjectServiceMock, pdfServiceMock, s.cacheInstance)

	resp1 := s.api.Post("/pdf/generate/archive", map[string]any{
		"url": "existing_template_existing_empty_schema",
		"input": map[string]any{
			"variables": "{\"test\": \"some_value\"}",
			"validate":  true,
		},
	})

	s.assert.Equal(200, resp1.Code)

	resp2 := s.api.Post("/pdf/generate/archive", map[string]any{
		"url": "existing_template_existing_empty_schema",
		"input": map[string]any{
			"variables": "{\"test\": \"some_value\"}",
			"validate":  true,
		},
	})

	s.assert.Equal(200, resp2.Code)

	// TODO: Figure out because https://github.com/stretchr/testify/issues/944
	// archiveProjectServiceMock.AssertExpectations(s.T())
	// pdfServiceMock.AssertExpectations(s.T())

	cacheKey, err := cache.Key([][]byte{[]byte("existing_template_existing_empty_schema")})
	if err != nil {
		s.T().Fatal(err)
	}

	s.cacheInstance.DeletePDF(s.ctx, cacheKey)
}

func TestCached(t *testing.T) {
	t.Parallel()

	suite.Run(t, new(CachedTestSuite))
}
