package archivegen_test

import (
	"context"
	"testing"

	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/dokufy/paperfit/config"
	"github.com/dokufy/paperfit/internal/server/operation/archivegen"
	"github.com/dokufy/paperfit/internal/service"
	"github.com/dokufy/paperfit/internal/service/pdfgenerator"
	"github.com/dokufy/paperfit/test"
	"github.com/dokufy/paperfit/test/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type TestSuite struct {
	suite.Suite
	ctx    context.Context
	api    humatest.TestAPI
	assert *assert.Assertions
}

func (s *TestSuite) SetupSuite() {
	config.Load()
	mocks.PassPDFgenerator()

	_, api := humatest.New(s.T())

	pdfProvider := pdfgenerator.NewHTTPpdfGenerator(config.Config.PDFgeneratorEndpoint)
	archivegen.RegisterOperation(api, "/pdf/generate/archive", pdfProvider, nil)

	s.ctx = context.Background()
	s.api = api
	s.assert = assert.New(s.T())
}

func (s *TestSuite) Test_Success_Generate_200() {
	file := test.File("archive/valid/simple.tar.gz")

	multipart, _ := pdfgenerator.MultipartForm([]*service.File{&file}, nil)

	resp := s.api.Post(
		"/pdf/generate/archive",
		"Content-type: "+multipart.FormDataContentType(),
		multipart.Body(),
	)

	s.assert.Equal(200, resp.Code)
}

func (s *TestSuite) Test_Success_Generate_With_Variables_200() {
	file := test.File("archive/valid/simple.tar.gz")

	multipart, _ := pdfgenerator.MultipartForm(
		[]*service.File{&file},
		&map[string]string{"variables": `{"title": "this is a title"}`},
	)

	resp := s.api.Post(
		"/pdf/generate/archive",
		"Content-type: "+multipart.FormDataContentType(),
		multipart.Body(),
	)

	s.assert.Equal(200, resp.Code)
}

func (s *TestSuite) Test_Success_Generate_With_Validation_And_Variables_200() {
	file := test.File("archive/valid/with_json_schema.tar.gz")
	multipart, _ := pdfgenerator.MultipartForm(
		[]*service.File{&file},
		&map[string]string{"variables": `{"title": "this is a title"}`, "validate": "true"},
	)

	resp := s.api.Post(
		"/pdf/generate/archive",
		"Content-type: "+multipart.FormDataContentType(),
		multipart.Body(),
	)

	s.assert.Equal(200, resp.Code)
}

func (s *TestSuite) Test_Success_Generate_With_Variables_Empty_Schema_200() {
	file := test.File("archive/valid/with_empty_json_schema.tar.gz")

	multipart, _ := pdfgenerator.MultipartForm(
		[]*service.File{&file},
		&map[string]string{"variables": `{"title": "this is a title"}`},
	)

	resp := s.api.Post(
		"/pdf/generate/archive",
		"Content-type: "+multipart.FormDataContentType(),
		multipart.Body(),
	)

	s.assert.Equal(200, resp.Code)
}

func (s *TestSuite) Test_Fail_Generate_No_Archive_400() {
	multipart, _ := pdfgenerator.MultipartForm([]*service.File{}, nil)

	resp := s.api.Post(
		"/pdf/generate/archive",
		"Content-type: "+multipart.FormDataContentType(),
		multipart.Body(),
	)

	s.assert.Equal(400, resp.Code)
	s.assert.Contains(resp.Body.String(), "an archive must be provided")
}

func (s *TestSuite) Test_Fail_Generate_Validation_And_Variables_Empty_Schema_400() {
	file := test.File("archive/valid/with_empty_json_schema.tar.gz")
	multipart, _ := pdfgenerator.MultipartForm([]*service.File{&file}, &map[string]string{"validate": "true"})

	resp := s.api.Post(
		"/pdf/generate/archive",
		"Content-type: "+multipart.FormDataContentType(),
		multipart.Body(),
	)

	s.assert.Equal(400, resp.Code)
	s.assert.Contains(resp.Body.String(), "No json schema found in the archive, please make sure that a json schema exists and it is not empty or disable variables validation")
}

func (s *TestSuite) Test_Fail_Generate_Bad_Variables_400() {
	file := test.File("archive/valid/with_json_schema.tar.gz")

	multipart, _ := pdfgenerator.MultipartForm(
		[]*service.File{&file},
		&map[string]string{"variables": "bad variables", "validate": "true"},
	)

	resp := s.api.Post(
		"/pdf/generate/archive",
		"Content-type: "+multipart.FormDataContentType(),
		multipart.Body(),
	)

	s.assert.Equal(400, resp.Code)
	s.assert.Contains(resp.Body.String(), "Failed to parse variables")
}

func (s *TestSuite) Test_Fail_Generate_Variables_Validation_Fails_400() {
	file := test.File("archive/valid/with_json_schema.tar.gz")

	multipart, _ := pdfgenerator.MultipartForm(
		[]*service.File{&file},
		&map[string]string{"variables": `{"wrong": "missing!!"}`, "validate": "true"},
	)

	resp := s.api.Post(
		"/pdf/generate/archive",
		"Content-type: "+multipart.FormDataContentType(),
		multipart.Body(),
	)

	s.assert.Equal(400, resp.Code)
	s.assert.Contains(resp.Body.String(), "Provided variables do not conform to the json schema")
}

func TestSuccessfulGeneration(t *testing.T) {
	t.Parallel()

	suite.Run(t, new(TestSuite))
}
