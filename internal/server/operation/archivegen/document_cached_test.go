package archivegen_test

import (
	"context"
	"testing"

	"github.com/atoncetti/combiner/config"
	"github.com/atoncetti/combiner/internal/cache"

	// "github.com/atoncetti/combiner/internal/httptool"
	"github.com/atoncetti/combiner/internal/server/operation/archivegen"
	"github.com/atoncetti/combiner/internal/service"
	"github.com/atoncetti/combiner/internal/service/pdfgenerator"
	"github.com/atoncetti/combiner/test"
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
	pdfServiceMock := new(mocks.PDFserviceMock)
	pdfServiceMock.On("Generate", mock.Anything).Return()

	archivegen.RegisterOperation(s.api, "/pdf/generate/url", pdfServiceMock, s.cacheInstance)

	archive, archive_size := test.File("archive/valid/simple.tar.gz")
	file := service.NewFile("simple.tar.gz", archive, archive_size)

	multipart, _ := pdfgenerator.MultipartForm([]*service.File{&file}, nil)
	resp1 := s.api.Post(
		"/pdf/generate/url",
		"Content-type: "+multipart.FormDataContentType(),
		multipart.Body(),
	)

	s.assert.Equal(200, resp1.Code)

	archive, archive_size = test.File("archive/valid/simple.tar.gz")
	file = service.NewFile("simple.tar.gz", archive, archive_size)
	multipart, _ = pdfgenerator.MultipartForm([]*service.File{&file}, nil)
	resp2 := s.api.Post(
		"/pdf/generate/url",
		"Content-type: "+multipart.FormDataContentType(),
		multipart.Body(),
	)

	s.assert.Equal(200, resp2.Code)

	// TODO: Figure out because https://github.com/stretchr/testify/issues/944
	// archiveProjectServiceMock.AssertExpectations(s.T())
	// pdfServiceMock.AssertExpectations(s.T())

	cacheKey, err := cache.Key([][]byte{[]byte("simple.tar.gz")})
	if err != nil {
		s.T().Fatal(err)
	}

	s.cacheInstance.DeletePDF(s.ctx, cacheKey)
}

func (s *CachedTestSuite) Test_Success_Generate_PDF_With_Cache_Different_Headers_Returns_200() {
	pdfServiceMock := new(mocks.PDFserviceMock)
	pdfServiceMock.On("Generate", mock.Anything).Return()

	archivegen.RegisterOperation(s.api, "/pdf/generate/url", pdfServiceMock, s.cacheInstance)

	archive, archive_size := test.File("archive/valid/simple.tar.gz")
	file := service.NewFile("simple.tar.gz", archive, archive_size)
	multipart, _ := pdfgenerator.MultipartForm([]*service.File{&file}, nil)

	resp1 := s.api.Post(
		"/pdf/generate/url",
		"Content-type: "+multipart.FormDataContentType(),
		multipart.Body(),
	)

	s.assert.Equal(200, resp1.Code)

	archive, archive_size = test.File("archive/valid/with_json_schema.tar.gz")
	file = service.NewFile("with_json_schema.tar.gz", archive, archive_size)
	multipart, _ = pdfgenerator.MultipartForm([]*service.File{&file}, nil)

	resp2 := s.api.Post(
		"/pdf/generate/url",
		"Content-type: "+multipart.FormDataContentType(),
		multipart.Body(),
	)

	s.assert.Equal(200, resp2.Code)

	// TODO: Figure out because https://github.com/stretchr/testify/issues/944
	// archiveProjectServiceMock.AssertExpectations(s.T())
	// pdfServiceMock.AssertExpectations(s.T())

	cacheKey1, err := cache.Key([][]byte{[]byte("existing_template_non_existing_schema")})
	if err != nil {
		s.T().Fatal(err)
	}

	cacheKey2, err := cache.Key([][]byte{[]byte("existing_template_non_existing_schema")})
	if err != nil {
		s.T().Fatal(err)
	}

	s.cacheInstance.DeletePDF(s.ctx, cacheKey1)
	s.cacheInstance.DeletePDF(s.ctx, cacheKey2)
}

func (s *CachedTestSuite) Test_Success_Generate_PDF_With_Cache_With_Validate_Arguments_Returns_200() {
	pdfServiceMock := new(mocks.PDFserviceMock)
	pdfServiceMock.On("Generate", mock.Anything).Return()

	archivegen.RegisterOperation(s.api, "/pdf/generate/url", pdfServiceMock, s.cacheInstance)

	archive, archive_size := test.File("archive/valid/simple.tar.gz")
	file := service.NewFile("simple.tar.gz", archive, archive_size)
	multipart, _ := pdfgenerator.MultipartForm([]*service.File{&file}, &map[string]string{"variables": "{\"title\": \"this is a title\"}"})

	resp1 := s.api.Post(
		"/pdf/generate/url",
		"Content-type: "+multipart.FormDataContentType(),
		multipart.Body(),
	)

	s.assert.Equal(200, resp1.Code)

	archive, archive_size = test.File("archive/valid/simple.tar.gz")
	file = service.NewFile("simple.tar.gz", archive, archive_size)
	multipart, _ = pdfgenerator.MultipartForm([]*service.File{&file}, &map[string]string{"variables": "{\"title\": \"this is a title\"}"})

	resp2 := s.api.Post(
		"/pdf/generate/url",
		"Content-type: "+multipart.FormDataContentType(),
		multipart.Body(),
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

func (s *CachedTestSuite) Test_Fail_Generate_PDF_With_Cache_With_Validate_Arguments_Empty_Schema_Returns_200() {
	pdfServiceMock := new(mocks.PDFserviceMock)
	pdfServiceMock.On("Generate", mock.Anything).Return()

	archivegen.RegisterOperation(s.api, "/pdf/generate/url", pdfServiceMock, s.cacheInstance)

	archive, archive_size := test.File("archive/valid/with_empty_json_schema.tar.gz")
	file := service.NewFile("with_empty_json_schema.tar.gz", archive, archive_size)
	multipart, _ := pdfgenerator.MultipartForm([]*service.File{&file}, &map[string]string{"variables": "{\"title\": \"this is a title\"}", "validate": "true"})

	resp := s.api.Post(
		"/pdf/generate/url",
		"Content-type: "+multipart.FormDataContentType(),
		multipart.Body(),
	)

	s.assert.Equal(400, resp.Code)
	// TODO: Figure out because https://github.com/stretchr/testify/issues/944
	// archiveProjectServiceMock.AssertExpectations(s.T())
	// pdfServiceMock.AssertExpectations(s.T())
}

func TestCached(t *testing.T) {
	t.Parallel()

	suite.Run(t, new(CachedTestSuite))
}
