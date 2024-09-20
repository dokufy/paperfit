package server_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/dokufy/paperfit/config"
	"github.com/dokufy/paperfit/internal/server"
	"github.com/dokufy/paperfit/internal/service"
	"github.com/dokufy/paperfit/internal/service/pdfgenerator"
	"github.com/dokufy/paperfit/test"
	"github.com/dokufy/paperfit/test/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type FileTestSuite struct {
	suite.Suite
	ctx    context.Context
	server *server.Server
	assert *assert.Assertions
}

func (s *FileTestSuite) SetupSuite() {
	config.Load()
	mocks.PassPDFgenerator()
	mocks.PassPath("/pdf/from/html/file")

	server, err := server.New()
	if err != nil {
		s.T().Fatal(err)
	}

	go server.Start(config.Config.ServerPort)

	s.server = server
	s.assert = assert.New(s.T())
}

func (s *FileTestSuite) TearDownSuite() {
	s.server.Stop()
}

func (s *FileTestSuite) Test_Success_Generate_200() {
	index := test.File("index.html")

	multipart, _ := pdfgenerator.MultipartForm("files", []*service.File{&index}, nil)

	resp := test.Post(
		test.FullURL("/pdf/from/html/file"),
		map[string]string{
			"Content-Type": multipart.FormDataContentType(),
		},
		multipart.Body(),
	)

	s.assert.Equal(200, resp.StatusCode)
}

func (s *FileTestSuite) Test_Success_Generate_With_Variables_200() {
	index := test.File("index.html")
	schema := test.File("valid_json_schema/json_schema.json")

	multipart, _ := pdfgenerator.MultipartForm(
		"files",
		[]*service.File{&index, &schema},
		&map[string]string{
			"input": `{"variables": {"title": "this is a title"}}`,
		},
	)

	resp := test.Post(
		test.FullURL("/pdf/from/html/file"),
		map[string]string{
			"Content-Type": multipart.FormDataContentType(),
		},
		multipart.Body(),
	)

	s.assert.Equal(200, resp.StatusCode)
}

func (s *FileTestSuite) Test_Success_Generate_With_Validation_And_Variables_200() {
	index := test.File("index.html")
	schema := test.File("valid_json_schema/json_schema.json")

	multipart, _ := pdfgenerator.MultipartForm(
		"files",
		[]*service.File{&index, &schema},
		&map[string]string{
			"input": `{"variables": {"title": "this is a title"}, "validate": true}`,
		},
	)

	resp := test.Post(
		test.FullURL("/pdf/from/html/file"),
		map[string]string{
			"Content-Type": multipart.FormDataContentType(),
		},
		multipart.Body(),
	)

	s.assert.Equal(200, resp.StatusCode)
}

func (s *FileTestSuite) Test_Fail_Generate_No_File_400() {
	multipart, _ := pdfgenerator.MultipartForm("files", []*service.File{}, nil)

	resp := test.Post(
		test.FullURL("/pdf/from/html/file"),
		map[string]string{
			"Content-Type": multipart.FormDataContentType(),
		},
		multipart.Body(),
	)

	s.assert.Equal(http.StatusBadRequest, resp.StatusCode)
	s.assert.Contains(test.StringBody(s.T(), resp.Body), server.ErrNoFileProvided.Error())
}

func (s *FileTestSuite) Test_Fail_Generate_Validation_And_Variables_Empty_Schema_400() {
	index := test.File("index.html")
	schema := test.File("empty_json_schema/json_schema.json")

	multipart, _ := pdfgenerator.MultipartForm(
		"files",
		[]*service.File{&index, &schema},
		&map[string]string{
			"input": `{"variables": {"title": "this is a title"}, "validate": true}`,
		},
	)

	resp := test.Post(
		test.FullURL("/pdf/from/html/file"),
		map[string]string{
			"Content-Type": multipart.FormDataContentType(),
		},
		multipart.Body(),
	)

	s.assert.Equal(http.StatusBadRequest, resp.StatusCode)
	s.assert.Contains(test.StringBody(s.T(), resp.Body), server.ErrNoSchemaFile.Error())
}

func (s *FileTestSuite) Test_Fail_Generate_Bad_Variables_400() {
	index := test.File("index.html")
	schema := test.File("valid_json_schema/json_schema.json")

	multipart, _ := pdfgenerator.MultipartForm(
		"files",
		[]*service.File{&index, &schema},
		&map[string]string{
			"input": `{"variables": "bad variables", "validate": true}`,
		},
	)

	resp := test.Post(
		test.FullURL("/pdf/from/html/file"),
		map[string]string{
			"Content-Type": multipart.FormDataContentType(),
		},
		multipart.Body(),
	)

	s.assert.Equal(http.StatusBadRequest, resp.StatusCode)
	s.assert.Contains(test.StringBody(s.T(), resp.Body), server.ErrInvalidInputFormat.Error())
}

func (s *FileTestSuite) Test_Fail_Generate_Variables_Validation_Fails_400() {
	index := test.File("index.html")
	schema := test.File("valid_json_schema/json_schema.json")

	multipart, _ := pdfgenerator.MultipartForm(
		"files",
		[]*service.File{&index, &schema},
		&map[string]string{
			"input": `{"variables": {"wrong": "missing!!"}, "validate": true}`,
		},
	)

	resp := test.Post(
		test.FullURL("/pdf/from/html/file"),
		map[string]string{
			"Content-Type": multipart.FormDataContentType(),
		},
		multipart.Body(),
	)

	s.assert.Equal(http.StatusBadRequest, resp.StatusCode)
	s.assert.Contains(test.StringBody(s.T(), resp.Body), server.ErrBadVariables.Error())
}

func TestSuccessfulGeneration(t *testing.T) {
	t.Parallel()

	suite.Run(t, new(FileTestSuite))
}
