package server_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/dokufy/paperfit/config"
	"github.com/dokufy/paperfit/internal/cache"
	"github.com/dokufy/paperfit/internal/logger"
	"github.com/dokufy/paperfit/internal/server"
	"github.com/dokufy/paperfit/internal/service"
	"github.com/dokufy/paperfit/internal/service/pdfgenerator"
	"github.com/dokufy/paperfit/test"
	"github.com/dokufy/paperfit/test/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type ArchiveCachedTestSuite struct {
	suite.Suite
	ctx           context.Context
	server        *server.Server
	cacheInstance *cache.Cache
	assert        *assert.Assertions
}

func (s *ArchiveCachedTestSuite) SetupSuite() {
	config.Load()

	logger.Initialize()
	mocks.PassPDFgenerator()
	mocks.PassPath("/pdf/from/archive/file")

	server, err := server.New()
	if err != nil {
		s.T().Fatal(err)
	}

	go server.Start(config.Config.ServerPort)

	cacheInstance, err := cache.New(config.Config.CacheConfig.CacheUrl)
	if err != nil {
		s.T().Fatal("failed to create cache")
	}

	s.ctx = context.Background()
	s.server = server
	s.cacheInstance = cacheInstance
	s.assert = assert.New(s.T())
}

func (s *ArchiveCachedTestSuite) TearDownSuite() {
	s.server.Stop()
}

func (s *ArchiveCachedTestSuite) Test_Success_Generate_200() {
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

	cacheKey, _ := cache.Key(file.Name())

	reader, err := s.cacheInstance.GetPDF(s.ctx, cacheKey)
	if err != nil {
		s.T().Fatal(err)
	}

	assert.NotEmpty(s.T(), reader)

	s.cacheInstance.DeletePDF(s.ctx, cacheKey)
}

func (s *ArchiveCachedTestSuite) Test_Success_Generate_With_Variables_200() {
	file := test.File("archive/valid/simple.tar.gz")
	variables := map[string]string{"title": "this is a title"}
	json, _ := json.Marshal(variables)

	multipart, _ := pdfgenerator.MultipartForm(
		"file",
		[]*service.File{&file},
		&map[string]string{"input": fmt.Sprintf(`{"variables": %s}`, string(json))},
	)

	resp := test.Post(
		test.FullURL("/pdf/from/archive/file"),
		map[string]string{
			"Content-Type": multipart.FormDataContentType(),
		},
		multipart.Body(),
	)

	s.assert.Equal(http.StatusOK, resp.StatusCode)

	cacheKey, _ := cache.Key(file.Name(), string(json))

	reader, err := s.cacheInstance.GetPDF(s.ctx, cacheKey)
	if err != nil {
		s.T().Fatal(err)
	}

	assert.NotEmpty(s.T(), reader)

	s.cacheInstance.DeletePDF(s.ctx, cacheKey)
}

func (s *ArchiveCachedTestSuite) Test_Success_Generate_With_Validation_And_Variables_200() {
	file := test.File("archive/valid/with_json_schema.tar.gz")
	variables := map[string]string{"title": "this is a title"}
	json, _ := json.Marshal(variables)

	multipart, _ := pdfgenerator.MultipartForm(
		"file",
		[]*service.File{&file},
		&map[string]string{
			"input": fmt.Sprintf(`{"variables": %s, "validate": true}`, string(json)),
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

	cacheKey, _ := cache.Key(file.Name(), string(json))

	s.cacheInstance.DeletePDF(s.ctx, cacheKey)
}

func (s *ArchiveCachedTestSuite) Test_Success_Generate_With_Variables_Empty_Schema_200() {
	file := test.File("archive/valid/with_empty_json_schema.tar.gz")
	variables := map[string]string{"title": "this is a title"}
	json, _ := json.Marshal(variables)

	multipart, _ := pdfgenerator.MultipartForm(
		"file",
		[]*service.File{&file},
		&map[string]string{
			"input": fmt.Sprintf(`{"variables": %s}`, string(json)),
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

	cacheKey, _ := cache.Key(file.Name(), string(json))

	s.cacheInstance.DeletePDF(s.ctx, cacheKey)
}

func (s *ArchiveCachedTestSuite) Test_Success_Generate_Use_Cache_200() {
	file := test.File("archive/valid/with_json_schema.tar.gz")
	variables := map[string]string{"title": "this is a title"}
	json, _ := json.Marshal(variables)

	multipart, _ := pdfgenerator.MultipartForm(
		"file",
		[]*service.File{&file},
		&map[string]string{
			"input": fmt.Sprintf(`{"variables": %s, "validate": true}`, string(json)),
		},
	)

	resp1 := test.Post(
		test.FullURL("/pdf/from/archive/file"),
		map[string]string{
			"Content-Type": multipart.FormDataContentType(),
		},
		multipart.Body(),
	)

	s.assert.Equal(http.StatusOK, resp1.StatusCode)

	cacheKey, _ := cache.Key(file.Name(), string(json))

	reader, err := s.cacheInstance.GetPDF(s.ctx, cacheKey)
	if err != nil {
		s.T().Fatal(err)
	}

	assert.NotEmpty(s.T(), reader)

	file = test.File("archive/valid/with_json_schema.tar.gz")
	multipart, _ = pdfgenerator.MultipartForm(
		"file",
		[]*service.File{&file},
		&map[string]string{
			"input": fmt.Sprintf(`{"variables": %s, "validate": true}`, string(json)),
		},
	)

	resp2 := test.Post(
		test.FullURL("/pdf/from/archive/file"),
		map[string]string{
			"Content-Type": multipart.FormDataContentType(),
		},
		multipart.Body(),
	)

	s.assert.Equal(http.StatusOK, resp2.StatusCode)

	cacheKey, _ = cache.Key(file.Name(), string(json))

	reader, err = s.cacheInstance.GetPDF(s.ctx, cacheKey)
	if err != nil {
		s.T().Fatal(err)
	}

	assert.NotEmpty(s.T(), reader)

	s.cacheInstance.DeletePDF(s.ctx, cacheKey)
}

func (s *ArchiveCachedTestSuite) Test_Success_Generate_Different_Variables_200() {
	file := test.File("archive/valid/with_json_schema.tar.gz")
	variables := map[string]string{"title": "this is a title"}
	json, _ := json.Marshal(variables)

	multipart, _ := pdfgenerator.MultipartForm(
		"file",
		[]*service.File{&file},
		&map[string]string{
			"input": fmt.Sprintf(`{"variables": %s, "validate": true}`, string(json)),
		},
	)

	resp1 := test.Post(
		test.FullURL("/pdf/from/archive/file"),
		map[string]string{
			"Content-Type": multipart.FormDataContentType(),
		},
		multipart.Body(),
	)

	s.assert.Equal(http.StatusOK, resp1.StatusCode)

	cacheKey1, _ := cache.Key(file.Name(), string(json))

	reader, err := s.cacheInstance.GetPDF(s.ctx, cacheKey1)
	if err != nil {
		s.T().Fatal(err)
	}

	assert.NotEmpty(s.T(), reader)

	file = test.File("archive/valid/with_json_schema.tar.gz")

	multipart, _ = pdfgenerator.MultipartForm(
		"file",
		[]*service.File{&file},
		&map[string]string{
			"input": fmt.Sprintf(`{"variables": %s, "validate": true}`, string(json)),
		},
	)

	resp2 := test.Post(
		test.FullURL("/pdf/from/archive/file"),
		map[string]string{
			"Content-Type": multipart.FormDataContentType(),
		},
		multipart.Body(),
	)

	s.assert.Equal(http.StatusOK, resp2.StatusCode)

	cacheKey2, _ := cache.Key(file.Name(), string(json))

	reader, err = s.cacheInstance.GetPDF(s.ctx, cacheKey2)
	if err != nil {
		s.T().Fatal(err)
	}

	assert.NotEmpty(s.T(), reader)

	s.cacheInstance.DeletePDF(s.ctx, cacheKey1)
	s.cacheInstance.DeletePDF(s.ctx, cacheKey2)
}

func (s *ArchiveCachedTestSuite) Test_Fail_Generate_No_Archive_400() {
	multipart, _ := pdfgenerator.MultipartForm(
		"file",
		[]*service.File{},
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

	s.assert.Equal(http.StatusBadRequest, resp.StatusCode)
	s.assert.Contains(test.StringBody(s.T(), resp.Body), server.ErrMissingArchive.Error())
}

func (s *ArchiveCachedTestSuite) Test_Fail_Generate_With_Validation_No_Variables_400() {
	file := test.File("archive/valid/with_json_schema.tar.gz")
	multipart, _ := pdfgenerator.MultipartForm(
		"file",
		[]*service.File{&file},
		&map[string]string{"input": `{"validate": true}`},
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

	cacheKey, _ := cache.Key(file.Name())

	reader, err := s.cacheInstance.GetPDF(s.ctx, cacheKey)
	if err != nil {
		s.T().Fatal(err)
	}

	s.assert.Empty(reader)
}

func (s *ArchiveCachedTestSuite) Test_Fail_Generate_With_Validation_And_Variables_Empty_Schema_400() {
	file := test.File("archive/valid/with_empty_json_schema.tar.gz")
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

	s.assert.Equal(http.StatusBadRequest, resp.StatusCode)
	s.assert.Contains(test.StringBody(s.T(), resp.Body), server.ErrNoSchemaFile.Error())

	cacheKey, _ := cache.Key(file.Name())

	reader, err := s.cacheInstance.GetPDF(s.ctx, cacheKey)
	if err != nil {
		s.T().Fatal(err)
	}

	s.assert.Empty(reader)
}

func (s *ArchiveCachedTestSuite) Test_Fail_Generate_Bad_Variables_400() {
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

func (s *ArchiveCachedTestSuite) Test_Fail_Generate_Variables_Validation_Fails_400() {
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

	cacheKey, _ := cache.Key(file.Name())

	reader, err := s.cacheInstance.GetPDF(s.ctx, cacheKey)
	if err != nil {
		s.T().Fatal(err)
	}

	s.assert.Empty(reader)
}

func TestArchiveCached(t *testing.T) {
	t.Parallel()

	suite.Run(t, new(ArchiveCachedTestSuite))
}
