package htmlgen_test

import (
	"context"
	"testing"

	"github.com/atoncetti/combiner/internal/server/operation/htmlgen"
	"github.com/atoncetti/combiner/internal/service/pdfprovider"
	"github.com/atoncetti/combiner/test/mocks"
	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type TestSuite struct {
	suite.Suite
	ctx    context.Context
	api    humatest.TestAPI
	assert *assert.Assertions
}

func (s *TestSuite) SetupSuite() {
	_, api := humatest.New(s.T())

	s.api = api
	s.assert = assert.New(s.T())
}

func (s *TestSuite) TestGeneratePDF_200() {
	pdfServiceMock := new(mocks.PDFserviceMock)
	project := mocks.ProjectMock("test")

	multipart, _ := pdfprovider.MultipartForm(project.GetFiles(), nil)

	pdfServiceMock.On("Generate", mock.Anything).Return()

	htmlgen.RegisterOperation(s.api, "/pdf/generate/html", pdfServiceMock)

	resp := s.api.Post(
		"/pdf/generate/html",
		"Content-type: "+multipart.FormDataContentType(),
		multipart.Body(),
	)

	// pdfServiceMock.AssertExpectations(s.T())

	s.assert.Equal(200, resp.Code)
}

func (s *TestSuite) TestGeneratePDF_WithValidateArguments200() {
	pdfServiceMock := new(mocks.PDFserviceMock)
	project := mocks.ProjectMock("test")

	multipart, _ := pdfprovider.MultipartForm(project.GetFiles(), &map[string]string{"variables": "{\"title\": \"this is a title\"}"})

	pdfServiceMock.On("Generate", mock.Anything).Return()

	htmlgen.RegisterOperation(s.api, "/pdf/generate/html", pdfServiceMock)

	resp := s.api.Post(
		"/pdf/generate/html",
		"Content-type: "+multipart.FormDataContentType(),
		multipart.Body(),
	)

	// pdfServiceMock.AssertExpectations(s.T())

	s.assert.Equal(200, resp.Code)
}

func (s *TestSuite) TestGeneratePDF_Fails400_InvalidArguments() {
	pdfServiceMock := new(mocks.PDFserviceMock)
	project := mocks.ProjectMock("test")

	multipart, _ := pdfprovider.MultipartForm(project.GetFiles(), &map[string]string{"variables": "bad variables"})

	pdfServiceMock.On("Generate", mock.Anything).Return()

	htmlgen.RegisterOperation(s.api, "/pdf/generate/html", pdfServiceMock)

	resp := s.api.Post(
		"/pdf/generate/html",
		"Content-type: "+multipart.FormDataContentType(),
		multipart.Body(),
	)

	// pdfServiceMock.AssertNotCalled(s.T(), "Generate", mock.Anything)

	s.assert.Equal(400, resp.Code)
}

func TestSuccessfulGeneration(t *testing.T) {
	t.Parallel()

	suite.Run(t, new(TestSuite))
}
