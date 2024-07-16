package archivegen_test

import (
	"context"
	"testing"

	"github.com/atoncetti/combiner/internal/cache"
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
	pdfServiceMock := new(mocks.PDFserviceMock)
	pdfServiceMock.On("Generate", mock.Anything).Return()

	archivegen.RegisterOperation(s.api, "/pdf/generate/url", pdfServiceMock, nil)

	archive, archive_size := test.File("archive/valid/simple.tar.gz")
	file := service.NewFile("simple.tar.gz", archive, archive_size)

	multipart, _ := pdfgenerator.MultipartForm([]*service.File{&file}, nil)

	resp := s.api.Post(
		"/pdf/generate/url",
		"Content-type: "+multipart.FormDataContentType(),
		multipart.Body(),
	)

	s.assert.Equal(200, resp.Code)
}

func (s *TestSuite) Test_Generate_PDF_No_Cache_With_Validate_Variables_Returns_200() {
	pdfServiceMock := new(mocks.PDFserviceMock)
	pdfServiceMock.On("Generate", mock.Anything).Return()

	archivegen.RegisterOperation(s.api, "/pdf/generate/url", pdfServiceMock, nil)

	archive, archive_size := test.File("archive/valid/simple.tar.gz")
	file := service.NewFile("simple.tar.gz", archive, archive_size)

	multipart, _ := pdfgenerator.MultipartForm(
		[]*service.File{&file},
		&map[string]string{"variables": "{\"title\": \"this is a title\"}"},
	)

	resp := s.api.Post(
		"/pdf/generate/url",
		"Content-type: "+multipart.FormDataContentType(),
		multipart.Body(),
	)

	s.assert.Equal(200, resp.Code)

	// TODO: Figure out because https://github.com/stretchr/testify/issues/944
	// archiveProjectServiceMock.AssertExpectations(s.T())
	// pdfServiceMock.AssertExpectations(s.T())
}

func (s *TestSuite) Test_Success_Generate_PDF_No_Cache_With_Validate_Variables_Empty_Schema_Returns_200() {
	pdfServiceMock := new(mocks.PDFserviceMock)
	pdfServiceMock.On("Generate", mock.Anything).Return()

	archivegen.RegisterOperation(s.api, "/pdf/generate/url", pdfServiceMock, nil)

	archive, archive_size := test.File("archive/valid/simple.tar.gz")
	file := service.NewFile("simple.tar.gz", archive, archive_size)

	multipart, _ := pdfgenerator.MultipartForm(
		[]*service.File{&file},
		&map[string]string{"variables": "{\"title\": \"this is a title\"}"},
	)

	resp := s.api.Post(
		"/pdf/generate/url",
		"Content-type: "+multipart.FormDataContentType(),
		multipart.Body(),
	)

	s.assert.Equal(200, resp.Code)
}

func (s *TestSuite) Test_Fail_Generate_PDF_No_Cache_Archive_Not_Provided_Returns_400() {
	pdfServiceMock := new(mocks.PDFserviceMock)
	pdfServiceMock.On("Generate", mock.Anything).Return()

	archivegen.RegisterOperation(s.api, "/pdf/generate/url", pdfServiceMock, nil)

	multipart, _ := pdfgenerator.MultipartForm([]*service.File{}, nil)

	resp := s.api.Post(
		"/pdf/generate/url",
		"Content-type: "+multipart.FormDataContentType(),
		multipart.Body(),
	)

	pdfServiceMock.AssertNotCalled(s.T(), "Generate", mock.Anything)

	s.assert.Equal(400, resp.Code)
}

func (s *TestSuite) Test_Fail_Generate_PDF_No_Cache_JSONschema_Required_And_NotFound_Returns_400() {
	pdfServiceMock := new(mocks.PDFserviceMock)

	archivegen.RegisterOperation(s.api, "/pdf/generate/url", pdfServiceMock, nil)

	archive, archive_size := test.File("archive/valid/simple.tar.gz")
	file := service.NewFile("simple.tar.gz", archive, archive_size)

	multipart, _ := pdfgenerator.MultipartForm([]*service.File{&file}, &map[string]string{"validate": "true"})

	resp := s.api.Post(
		"/pdf/generate/url",
		"Content-type: "+multipart.FormDataContentType(),
		multipart.Body(),
	)

	// archiveProjectServiceMock.AssertExpectations(s.T())
	// pdfServiceMock.AssertNotCalled(s.T(), "Generate", mock.Anything)

	s.assert.Equal(400, resp.Code)
}

func (s *TestSuite) Test_Fail_Generate_PDF_No_Cache_Variables_Validation_Fails_Returns_400() {
	pdfServiceMock := new(mocks.PDFserviceMock)
	pdfServiceMock.On("Generate", mock.Anything).Return()

	archivegen.RegisterOperation(s.api, "/pdf/generate/url", pdfServiceMock, nil)

	archive, archive_size := test.File("archive/valid/with_json_schema.tar.gz")
	file := service.NewFile("simple.tar.gz", archive, archive_size)

	multipart, _ := pdfgenerator.MultipartForm(
		[]*service.File{&file},
		&map[string]string{"variables": "{\"wrong\": \"missing!!\"}", "validate": "true"},
	)

	resp := s.api.Post(
		"/pdf/generate/url",
		"Content-type: "+multipart.FormDataContentType(),
		multipart.Body(),
	)

	// archiveProjectServiceMock.AssertExpectations(s.T())
	pdfServiceMock.AssertNotCalled(s.T(), "Generate", mock.Anything)

	s.assert.Equal(400, resp.Code)
}

func TestSuccessfulGeneration(t *testing.T) {
	t.Parallel()

	suite.Run(t, new(TestSuite))
}
