package archiveurlgen_test

import (
	"context"
	"testing"

	"github.com/atoncetti/combiner/internal/cache"
	"github.com/atoncetti/combiner/internal/server/operation/archiveurlgen"
	"github.com/atoncetti/combiner/test/mocks"
	"github.com/danielgtaylor/huma/v2/humatest"
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
	_, api := humatest.New(s.T())

	s.ctx = context.Background()
	s.api = api
	s.cacheInstance = nil
	s.assert = assert.New(s.T())
}

func (s *TestSuite) Test_Success_Generate_PDF_No_Cache_Returns_200() {
	archiveProjectServiceMock := new(mocks.ArchiveProjectServiceMock)
	archiveProjectServiceMock.On("GetProject", mock.Anything).Return()

	pdfServiceMock := new(mocks.PDFserviceMock)
	pdfServiceMock.On("Generate", mock.Anything).Return()

	archiveurlgen.RegisterOperation(s.api, "/pdf/generate/archive", archiveProjectServiceMock, pdfServiceMock, nil)

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

	// archiveProjectServiceMock.AssertExpectations(s.T())
	// pdfServiceMock.AssertExpectations(s.T())
}

func (s *TestSuite) Test_Generate_PDF_No_Cache_With_Validate_Variables_Returns_200() {
	archiveProjectServiceMock := new(mocks.ArchiveProjectServiceMock)
	archiveProjectServiceMock.On("GetProject", mock.Anything).Return()

	pdfServiceMock := new(mocks.PDFserviceMock)
	pdfServiceMock.On("Generate", mock.Anything).Return()

	archiveurlgen.RegisterOperation(s.api, "/pdf/generate/archive", archiveProjectServiceMock, pdfServiceMock, nil)

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
}

func (s *TestSuite) Test_Success_Generate_PDF_No_Cache_With_Validate_Variables_Empty_Schema_Returns_200() {
	archiveProjectServiceMock := new(mocks.ArchiveProjectServiceMock)
	archiveProjectServiceMock.On("GetProject", mock.Anything).Return()

	pdfServiceMock := new(mocks.PDFserviceMock)
	pdfServiceMock.On("Generate", mock.Anything).Return()

	archiveurlgen.RegisterOperation(s.api, "/pdf/generate/archive", archiveProjectServiceMock, pdfServiceMock, nil)

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
}

func (s *TestSuite) Test_Fail_Generate_PDF_No_Cache_URL_Not_Exists_Returns_404() {
	archiveProjectServiceMock := new(mocks.ArchiveProjectServiceMock)
	archiveProjectServiceMock.On("GetProject", mock.Anything).Return()

	pdfServiceMock := new(mocks.PDFserviceMock)
	pdfServiceMock.On("Generate", mock.Anything).Return()

	archiveurlgen.RegisterOperation(s.api, "/pdf/generate/archive", archiveProjectServiceMock, pdfServiceMock, nil)

	resp := s.api.Post("/pdf/generate/archive", map[string]any{
		"url": "non_existing",
	})

	// archiveProjectServiceMock.AssertExpectations(s.T())
	pdfServiceMock.AssertNotCalled(s.T(), "Generate", mock.Anything)

	s.assert.Equal(404, resp.Code)
}

func (s *TestSuite) Test_Fail_Generate_PDF_No_Cache_URL_Not_Provided_Returns_400() {
	archiveProjectServiceMock := new(mocks.ArchiveProjectServiceMock)
	pdfServiceMock := new(mocks.PDFserviceMock)

	archiveurlgen.RegisterOperation(s.api, "/pdf/generate/archive", archiveProjectServiceMock, pdfServiceMock, nil)

	resp := s.api.Post("/pdf/generate/archive", map[string]any{
		"wrong": "no_url",
	})

	archiveProjectServiceMock.AssertNotCalled(s.T(), "GetProject", mock.Anything)
	pdfServiceMock.AssertNotCalled(s.T(), "Generate", mock.Anything)

	s.assert.Equal(400, resp.Code)
}

// func (s *TestSuite) Test_Fail_Generate_PDF_No_Cache_JSONschema_Required_And_NotFound_Returns_400() {
// 	archiveProjectServiceMock := new(mocks.ArchiveProjectServiceMock)
// 	pdfServiceMock := new(mocks.PDFserviceMock)

// 	archiveProjectServiceMock.On("GetProject", mock.Anything).Return()

// 	archiveurlgen.RegisterOperation(s.api, "/pdf/generate/archive", archiveProjectServiceMock, pdfServiceMock, nil)

// 	resp := s.api.Post("/pdf/generate/archive", map[string]any{
// 		"url": "existing_template_non_existing_schema",
// 		"input": map[string]any{
// 			"variables": "{\"test\": \"some_value\"}",
// 			"validate":  true,
// 		},
// 	})

// 	// archiveProjectServiceMock.AssertExpectations(s.T())
// 	// pdfServiceMock.AssertNotCalled(s.T(), "Generate", mock.Anything)

// 	s.assert.Equal(400, resp.Code)
// }

func (s *TestSuite) Test_Fail_Generate_PDF_No_Cache_Variables_Validation_Fails_Returns_400() {
	archiveProjectServiceMock := new(mocks.ArchiveProjectServiceMock)
	archiveProjectServiceMock.On("GetProject", mock.Anything).Return()

	pdfServiceMock := new(mocks.PDFserviceMock)

	archiveurlgen.RegisterOperation(s.api, "/pdf/generate/archive", archiveProjectServiceMock, pdfServiceMock, nil)

	resp := s.api.Post("/pdf/generate/archive", map[string]any{
		"url": "existing_template_existing_schema",
		"input": map[string]any{
			"variables": "{\"test\": \"some_value\"}",
			"validate":  true,
		},
	})

	// archiveProjectServiceMock.AssertExpectations(s.T())
	pdfServiceMock.AssertNotCalled(s.T(), "Generate", mock.Anything)

	s.assert.Equal(400, resp.Code)
}

func TestSuccessfulGeneration(t *testing.T) {
	t.Parallel()

	suite.Run(t, new(TestSuite))
}
