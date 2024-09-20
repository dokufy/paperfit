package server_test

import (
	"context"
	"net/http"
	"os"
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

type ArchiveNonCachedTestSuite struct {
	suite.Suite
	ctx    context.Context
	server *server.Server
	assert *assert.Assertions
}

func (s *ArchiveNonCachedTestSuite) SetupSuite() {
	os.Setenv("CACHE_URL", "")

	config.Load()
	mocks.PassPDFgenerator()
	mocks.PassPath("/pdf/from/archive/file")

	server, err := server.New()
	if err != nil {
		s.T().Fatal(err)
	}

	go server.Start(config.Config.ServerPort)

	s.ctx = context.Background()
	s.server = server
	s.assert = assert.New(s.T())
}

func (s *ArchiveNonCachedTestSuite) TearDownSuite() {
	s.server.Stop()
}

func (s *ArchiveNonCachedTestSuite) Test_Success_Generate_200() {
	file := test.File("archive/valid/simple.tar.gz")

	multipart, _ := pdfgenerator.MultipartForm("file", []*service.File{&file}, nil)

	resp := test.Post(
		test.FullURL("/pdf/from/archive/file"),
		map[string]string{
			"Content-Type": multipart.FormDataContentType(),
		},
		multipart.Body(),
	)

	s.assert.Equal(http.StatusOK, resp.StatusCode)
}

func (s *ArchiveNonCachedTestSuite) Test_Success_Generate_With_Variables_200() {
	file := test.File("archive/valid/simple.tar.gz")

	multipart, _ := pdfgenerator.MultipartForm(
		"file",
		[]*service.File{&file},
		&map[string]string{
			"input": `{"variables": {"title": "this is a title"}}`,
		},
	)

	resp := test.Post(
		test.FullURL("/pdf/from/archive/file"),
		map[string]string{
			"Content-Type": multipart.FormDataContentType(),
		},
		multipart.Body(),
	)

	s.assert.Equal(http.StatusOK, resp.StatusCode)
}

func (s *ArchiveNonCachedTestSuite) Test_Success_Generate_With_Validation_And_Variables_200() {
	file := test.File("archive/valid/with_json_schema.tar.gz")

	multipart, _ := pdfgenerator.MultipartForm(
		"file",
		[]*service.File{&file},
		&map[string]string{
			"input": `{"variables": {"title": "this is a title"}, "validate": true}`,
		},
	)

	resp := test.Post(
		test.FullURL("/pdf/from/archive/file"),
		map[string]string{
			"Content-Type": multipart.FormDataContentType(),
		},
		multipart.Body(),
	)

	s.assert.Equal(http.StatusOK, resp.StatusCode)
}

func (s *ArchiveNonCachedTestSuite) Test_Success_Generate_With_Variables_Empty_Schema_200() {
	file := test.File("archive/valid/with_empty_json_schema.tar.gz")

	multipart, _ := pdfgenerator.MultipartForm(
		"file",
		[]*service.File{&file},
		&map[string]string{
			"input": `{"variables": {"title": "this is a title"}}`,
		},
	)

	resp := test.Post(
		test.FullURL("/pdf/from/archive/file"),
		map[string]string{
			"Content-Type": multipart.FormDataContentType(),
		},
		multipart.Body(),
	)

	s.assert.Equal(http.StatusOK, resp.StatusCode)
}

func (s *ArchiveNonCachedTestSuite) Test_Fail_Generate_No_Archive_400() {
	multipart, _ := pdfgenerator.MultipartForm("file", []*service.File{}, nil)

	resp := test.Post(
		test.FullURL("/pdf/from/archive/file"),
		map[string]string{
			"Content-Type": multipart.FormDataContentType(),
		},
		multipart.Body(),
	)

	s.assert.Equal(http.StatusBadRequest, resp.StatusCode)
	s.assert.Contains(test.StringBody(s.T(), resp.Body), server.ErrMissingArchive.Error())
}

func (s *ArchiveNonCachedTestSuite) Test_Fail_Generate_Validation_And_Variables_Empty_Schema_400() {
	file := test.File("archive/valid/with_empty_json_schema.tar.gz")
	multipart, _ := pdfgenerator.MultipartForm(
		"file",
		[]*service.File{&file},
		&map[string]string{
			"input": `{"validate": true}`,
		},
	)

	resp := test.Post(
		test.FullURL("/pdf/from/archive/file"),
		map[string]string{
			"Content-Type": multipart.FormDataContentType(),
		},
		multipart.Body(),
	)

	s.assert.Equal(http.StatusBadRequest, resp.StatusCode)
	s.assert.Contains(test.StringBody(s.T(), resp.Body), server.ErrNoSchemaFile.Error())
}

func (s *ArchiveNonCachedTestSuite) Test_Fail_Generate_Invalid_Variables_Format_400() {
	file := test.File("archive/valid/with_json_schema.tar.gz")

	multipart, _ := pdfgenerator.MultipartForm(
		"file",
		[]*service.File{&file},
		&map[string]string{
			"input": `{"variables": "bad variables", "validate": true}`,
		},
	)

	resp := test.Post(
		test.FullURL("/pdf/from/archive/file"),
		map[string]string{
			"Content-Type": multipart.FormDataContentType(),
		},
		multipart.Body(),
	)

	s.assert.Equal(http.StatusBadRequest, resp.StatusCode)
	s.assert.Contains(test.StringBody(s.T(), resp.Body), server.ErrInvalidInputFormat.Error())
}

func (s *ArchiveNonCachedTestSuite) Test_Fail_Generate_Invalid_Validate_Format_400() {
	file := test.File("archive/valid/with_json_schema.tar.gz")

	multipart, _ := pdfgenerator.MultipartForm(
		"file",
		[]*service.File{&file},
		&map[string]string{
			"input": `{"variables": {}, "validate": "true"}`,
		},
	)

	resp := test.Post(
		test.FullURL("/pdf/from/archive/file"),
		map[string]string{
			"Content-Type": multipart.FormDataContentType(),
		},
		multipart.Body(),
	)

	s.assert.Equal(http.StatusBadRequest, resp.StatusCode)
	s.assert.Contains(test.StringBody(s.T(), resp.Body), server.ErrInvalidInputFormat.Error())
}

func (s *ArchiveNonCachedTestSuite) Test_Fail_Generate_Variables_Validation_Fails_400() {
	file := test.File("archive/valid/with_json_schema.tar.gz")

	multipart, _ := pdfgenerator.MultipartForm(
		"file",
		[]*service.File{&file},
		&map[string]string{
			"input": `{"variables": {"wrong": "missing!!"}, "validate": true}`,
		},
	)

	resp := test.Post(
		test.FullURL("/pdf/from/archive/file"),
		map[string]string{
			"Content-Type": multipart.FormDataContentType(),
		},
		multipart.Body(),
	)

	s.assert.Equal(http.StatusBadRequest, resp.StatusCode)
	s.assert.Contains(test.StringBody(s.T(), resp.Body), server.ErrBadVariables.Error())
}

func TestArchiveNonCached(t *testing.T) {
	t.Parallel()

	suite.Run(t, new(ArchiveNonCachedTestSuite))
}
