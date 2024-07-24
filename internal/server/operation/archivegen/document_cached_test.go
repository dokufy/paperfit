package archivegen_test

import (
	"context"
	"testing"

	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/dokufy/paperfit/config"
	"github.com/dokufy/paperfit/internal/cache"
	"github.com/dokufy/paperfit/internal/server/operation/archivegen"
	"github.com/dokufy/paperfit/internal/service"
	"github.com/dokufy/paperfit/internal/service/pdfgenerator"
	"github.com/dokufy/paperfit/test"
	"github.com/dokufy/paperfit/test/mocks"
	"github.com/stretchr/testify/assert"
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

	pdfProvider := pdfgenerator.NewHTTPpdfGenerator(config.Config.PDFgeneratorEndpoint)

	cacheInstance, err := cache.New(config.Config.CacheConfig.CacheUrl)
	if err != nil {
		s.T().Fatal("failed to create cache")
	}

	_, api := humatest.New(s.T())

	archivegen.RegisterOperation(api, "/pdf/generate/archive", pdfProvider, cacheInstance)

	s.ctx = context.Background()
	s.api = api
	s.cacheInstance = cacheInstance
	s.assert = assert.New(s.T())
}

func (s *CachedTestSuite) Test_Success_Generate_200() {
	file := test.File("archive/valid/simple.tar.gz")

	multipart, _ := pdfgenerator.MultipartForm([]*service.File{&file}, nil)
	resp := s.api.Post(
		"/pdf/generate/archive",
		"Content-type: "+multipart.FormDataContentType(),
		multipart.Body(),
	)

	s.assert.Equal(200, resp.Code)

	cacheKey, _ := cache.Key([][]byte{[]byte(file.Name())})

	reader, err := s.cacheInstance.GetPDF(s.ctx, cacheKey)
	if err != nil {
		s.T().Fatal(err)
	}

	assert.NotEmpty(s.T(), reader)

	s.cacheInstance.DeletePDF(s.ctx, cacheKey)
}

func (s *CachedTestSuite) Test_Success_Generate_With_Variables_200() {
	file := test.File("archive/valid/simple.tar.gz")
	variables := `{"title": "this is a title"}`
	multipart, _ := pdfgenerator.MultipartForm(
		[]*service.File{&file},
		&map[string]string{"variables": variables},
	)

	resp := s.api.Post(
		"/pdf/generate/archive",
		"Content-type: "+multipart.FormDataContentType(),
		multipart.Body(),
	)

	s.assert.Equal(200, resp.Code)

	cacheKey, _ := cache.Key([][]byte{[]byte(file.Name()), []byte(variables)})

	reader, err := s.cacheInstance.GetPDF(s.ctx, cacheKey)
	if err != nil {
		s.T().Fatal(err)
	}

	assert.NotEmpty(s.T(), reader)

	s.cacheInstance.DeletePDF(s.ctx, cacheKey)
}

func (s *CachedTestSuite) Test_Success_Generate_With_Validation_And_Variables_200() {
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

	cacheKey, _ := cache.Key([][]byte{[]byte(file.Name())})

	s.cacheInstance.DeletePDF(s.ctx, cacheKey)
}

func (s *CachedTestSuite) Test_Success_Generate_With_Variables_Empty_Schema_200() {
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

func (s *CachedTestSuite) Test_Success_Generate_Use_Cache_200() {
	file := test.File("archive/valid/with_json_schema.tar.gz")
	variables := `{"title": "this is a title"}`
	multipart, _ := pdfgenerator.MultipartForm(
		[]*service.File{&file},
		&map[string]string{"variables": variables, "validate": "true"},
	)

	resp1 := s.api.Post(
		"/pdf/generate/archive",
		"Content-type: "+multipart.FormDataContentType(),
		multipart.Body(),
	)

	s.assert.Equal(200, resp1.Code)

	cacheKey, _ := cache.Key([][]byte{[]byte(file.Name()), []byte(variables)})

	reader, err := s.cacheInstance.GetPDF(s.ctx, cacheKey)
	if err != nil {
		s.T().Fatal(err)
	}

	assert.NotEmpty(s.T(), reader)

	file = test.File("archive/valid/with_json_schema.tar.gz")
	multipart, _ = pdfgenerator.MultipartForm(
		[]*service.File{&file},
		&map[string]string{"variables": variables, "validate": "true"},
	)

	resp2 := s.api.Post(
		"/pdf/generate/archive",
		"Content-type: "+multipart.FormDataContentType(),
		multipart.Body(),
	)

	s.assert.Equal(200, resp2.Code)

	cacheKey, _ = cache.Key([][]byte{[]byte(file.Name()), []byte(variables)})

	reader, err = s.cacheInstance.GetPDF(s.ctx, cacheKey)
	if err != nil {
		s.T().Fatal(err)
	}

	assert.NotEmpty(s.T(), reader)

	s.cacheInstance.DeletePDF(s.ctx, cacheKey)
}

func (s *CachedTestSuite) Test_Success_Generate_Different_Variables_200() {
	file := test.File("archive/valid/with_json_schema.tar.gz")
	variables1 := `{"title": "this is a title"}`
	multipart, _ := pdfgenerator.MultipartForm(
		[]*service.File{&file},
		&map[string]string{"variables": variables1, "validate": "true"},
	)

	resp1 := s.api.Post(
		"/pdf/generate/archive",
		"Content-type: "+multipart.FormDataContentType(),
		multipart.Body(),
	)

	s.assert.Equal(200, resp1.Code)

	cacheKey1, _ := cache.Key([][]byte{[]byte(file.Name()), []byte(variables1)})

	reader, err := s.cacheInstance.GetPDF(s.ctx, cacheKey1)
	if err != nil {
		s.T().Fatal(err)
	}

	assert.NotEmpty(s.T(), reader)

	file = test.File("archive/valid/with_json_schema.tar.gz")
	variables2 := `{"title": "this is a title"}`
	multipart, _ = pdfgenerator.MultipartForm(
		[]*service.File{&file},
		&map[string]string{"variables": variables2, "validate": "true"},
	)

	resp2 := s.api.Post(
		"/pdf/generate/archive",
		"Content-type: "+multipart.FormDataContentType(),
		multipart.Body(),
	)

	s.assert.Equal(200, resp2.Code)

	cacheKey2, _ := cache.Key([][]byte{[]byte(file.Name()), []byte(variables2)})

	reader, err = s.cacheInstance.GetPDF(s.ctx, cacheKey2)
	if err != nil {
		s.T().Fatal(err)
	}

	assert.NotEmpty(s.T(), reader)

	s.cacheInstance.DeletePDF(s.ctx, cacheKey1)
	s.cacheInstance.DeletePDF(s.ctx, cacheKey2)
}

func (s *CachedTestSuite) Test_Fail_Generate_No_Archive_400() {
	multipart, _ := pdfgenerator.MultipartForm(
		[]*service.File{},
		&map[string]string{"variables": `{"title": "this is a title"}`, "validate": "true"},
	)

	resp := s.api.Post(
		"/pdf/generate/archive",
		"Content-type: "+multipart.FormDataContentType(),
		multipart.Body(),
	)

	s.assert.Equal(400, resp.Code)
	s.assert.Contains(resp.Body.String(), "an archive must be provided")
}

func (s *CachedTestSuite) Test_Fail_Generate_With_Validation_No_Variables_400() {
	file := test.File("archive/valid/with_json_schema.tar.gz")
	multipart, _ := pdfgenerator.MultipartForm(
		[]*service.File{&file},
		&map[string]string{"validate": "true"},
	)

	resp := s.api.Post(
		"/pdf/generate/archive",
		"Content-type: "+multipart.FormDataContentType(),
		multipart.Body(),
	)

	s.assert.Equal(400, resp.Code)
	s.assert.Contains(resp.Body.String(), "Provided variables do not conform to the json schema")

	cacheKey, _ := cache.Key([][]byte{[]byte(file.Name())})

	reader, err := s.cacheInstance.GetPDF(s.ctx, cacheKey)
	if err != nil {
		s.T().Fatal(err)
	}

	s.assert.Empty(reader)
}

func (s *CachedTestSuite) Test_Fail_Generate_With_Validation_And_Variables_Empty_Schema_400() {
	file := test.File("archive/valid/with_empty_json_schema.tar.gz")
	multipart, _ := pdfgenerator.MultipartForm(
		[]*service.File{&file},
		&map[string]string{"variables": `{"title": "this is a title"}`, "validate": "true"},
	)

	resp := s.api.Post(
		"/pdf/generate/archive",
		"Content-type: "+multipart.FormDataContentType(),
		multipart.Body(),
	)

	s.assert.Equal(400, resp.Code)
	s.assert.Contains(resp.Body.String(), "No json schema found in the archive, please make sure that a json schema exists and it is not empty or disable variables validation")

	cacheKey, _ := cache.Key([][]byte{[]byte(file.Name())})

	reader, err := s.cacheInstance.GetPDF(s.ctx, cacheKey)
	if err != nil {
		s.T().Fatal(err)
	}

	s.assert.Empty(reader)
}

func (s *CachedTestSuite) Test_Fail_Generate_Bad_Variables_400() {
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

func (s *CachedTestSuite) Test_Fail_Generate_Variables_Validation_Fails_400() {
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

	cacheKey, _ := cache.Key([][]byte{[]byte(file.Name())})

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
