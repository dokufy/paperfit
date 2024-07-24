package server_test

import (
	"context"
	"testing"

	"github.com/atoncetti/combiner/config"
	"github.com/atoncetti/combiner/internal/cache"
	"github.com/atoncetti/combiner/internal/logger"
	"github.com/atoncetti/combiner/internal/server"
	"github.com/atoncetti/combiner/internal/service"
	"github.com/atoncetti/combiner/internal/service/pdfgenerator"
	"github.com/atoncetti/combiner/test"
	"github.com/atoncetti/combiner/test/mocks"
	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type TestSuite struct {
	suite.Suite
	ctx    context.Context
	assert *assert.Assertions
	api    humatest.TestAPI
	cache  *cache.Cache
	logger logger.Logger
}

func (s *TestSuite) SetupSuite() {
	config.Load()
	logger := logger.New()
	cache, err := cache.New(config.Config.CacheConfig.CacheUrl, nil)
	if err != nil {
		s.T().Fatal("failed to create cache")
	}

	s.cache = cache
	s.logger = logger
	s.assert = assert.New(s.T())
}

func setupApiWithCache(s *TestSuite) {
	_, api := humatest.New(s.T())

	httpPDFprovider := pdfgenerator.NewHTTPpdfGenerator(config.Config.PDFgeneratorEndpoint)

	addRoutes(api, httpPDFprovider, s.cache, &s.logger)

	s.api = api
}

func setupApiWithoutCache(s *TestSuite) {
	_, api := humatest.New(s.T())

	httpPDFprovider := pdfgenerator.NewHTTPpdfGenerator(config.Config.PDFgeneratorEndpoint)

	addRoutes(api, httpPDFprovider, nil, &s.logger)

	s.api = api
}

func addRoutes(api humatest.TestAPI, pdfProvider *pdfgenerator.HTTPpdfGenerator, cache *cache.Cache, logger *logger.Logger) {
	server.AddRoutes(api, pdfProvider, cache, logger)
}

// Testing html operation

func (s *TestSuite) Test_200_Html_Generate() {
	mocks.PassPDFgenerator()

	setupApiWithoutCache(s)

	project := mocks.ProjectMock("test")

	multipart, err := pdfgenerator.MultipartForm(project.GetFiles(), nil)
	if err != nil {
		s.T().Fatal(err)
	}

	resp := s.api.Post("/pdf/generate/html", "Content-type: "+multipart.FormDataContentType(), multipart.Body())

	s.assert.Equal(200, resp.Code)
}

// // Testing url operation

func (s *TestSuite) Test_200_No_Cache_Url_Generate() {
	mocks.PassPDFgenerator()
	mocks.MockHTTPfileRequest(mocks.SimpleFileMockEndpoint)

	setupApiWithoutCache(s)

	body := map[string]string{"url": mocks.SimpleFileMockEndpoint.Url}
	resp := s.api.Post("/pdf/generate/url", "application/json", body)

	s.assert.Equal(200, resp.Code)
}

func (s *TestSuite) Test_200_With_Cache_Url_Generate() {
	mocks.PassPDFgenerator()
	mocks.MockHTTPfileRequest(mocks.SimpleFileMockEndpoint)

	ctx := context.Background()

	setupApiWithCache(s)

	body := map[string]string{"url": mocks.SimpleFileMockEndpoint.Url}
	resp1 := s.api.Post("/pdf/generate/url", "application/json", body)

	mocks.RemoveMockedRequests()
	resp2 := s.api.Post("/pdf/generate/url", "application/json", body)

	s.assert.Equal(200, resp1.Code)
	s.assert.Equal(200, resp2.Code)

	archiveCacheKey, err := cache.Key([][]byte{[]byte(mocks.SimpleFileMockEndpoint.Url)})
	if err != nil {
		s.T().Fatal(err)
	}

	pdfCacheKey, err := cache.Key([][]byte{[]byte(mocks.SimpleFileMockEndpoint.Url)})
	if err != nil {
		s.T().Fatal(err)
	}

	s.cache.DeleteArchive(ctx, archiveCacheKey)
	s.cache.DeletePDF(ctx, pdfCacheKey)
}

func (s *TestSuite) Test_Fail_400_Url_Generate_No_URL_Provided() {
	setupApiWithoutCache(s)

	body := map[string]string{"name": "John Doe", "occupation": "gardener"}
	resp := s.api.Post("/pdf/generate/url", "application/json", body)

	s.assert.Equal(400, resp.Code)
}

func (s *TestSuite) Test_Fail_400_Url_Generate_No_HTML_In_Archive() {
	mocks.PassPDFgenerator()
	mocks.MockHTTPfileRequest(mocks.NoHtmlFileMockEndpoint)

	setupApiWithoutCache(s)

	body := map[string]string{"url": mocks.NoHtmlFileMockEndpoint.Url}
	resp := s.api.Post("/pdf/generate/url", "application/json", body)

	s.assert.Equal(400, resp.Code)
}

func (s *TestSuite) Test_200_No_Cache_Archive_Generate() {
	mocks.PassPDFgenerator()
	mocks.MockHTTPfileRequest(mocks.SimpleFileMockEndpoint)

	setupApiWithoutCache(s)

	body := map[string]string{"url": mocks.SimpleFileMockEndpoint.Url}
	resp := s.api.Post("/pdf/generate/url", "Content-type: application/json", body)

	s.assert.Equal(200, resp.Code)
}

// Testing archive operation

func (s *TestSuite) Test_200_With_Cache_Archive_Generate() {
	mocks.PassPDFgenerator()

	ctx := context.Background()

	setupApiWithCache(s)

	file := test.File("archive/valid/with_json_schema.tar.gz")

	multipart, _ := pdfgenerator.MultipartForm(
		[]*service.File{&file},
		&map[string]string{"variables": `{"title": "this is a title"}`},
	)

	resp1 := s.api.Post(
		"/pdf/generate/archive",
		"Content-type: "+multipart.FormDataContentType(),
		multipart.Body(),
	)

	mocks.RemoveMockedRequests()

	file = test.File("archive/valid/with_json_schema.tar.gz")

	multipart, _ = pdfgenerator.MultipartForm(
		[]*service.File{&file},
		&map[string]string{"variables": `{"title": "this is a title"}`},
	)

	resp2 := s.api.Post(
		"/pdf/generate/archive",
		"Content-type: "+multipart.FormDataContentType(),
		multipart.Body(),
	)

	s.assert.Equal(200, resp1.Code)
	s.assert.Equal(200, resp2.Code)

	archiveCacheKey, err := cache.Key([][]byte{[]byte(mocks.SimpleFileMockEndpoint.Url)})
	if err != nil {
		s.T().Fatal(err)
	}

	pdfCacheKey, err := cache.Key([][]byte{[]byte(mocks.SimpleFileMockEndpoint.Url)})
	if err != nil {
		s.T().Fatal(err)
	}

	s.cache.DeleteArchive(ctx, archiveCacheKey)
	s.cache.DeletePDF(ctx, pdfCacheKey)
}

func (s *TestSuite) Test_Fail_400_Archive_Generate_No_Archive_Provided() {
	setupApiWithoutCache(s)

	multipart, _ := pdfgenerator.MultipartForm([]*service.File{}, &map[string]string{"validate": "true"})

	resp := s.api.Post(
		"/pdf/generate/archive",
		"Content-type: "+multipart.FormDataContentType(),
		multipart.Body(),
	)

	s.assert.Equal(400, resp.Code)
}

func TestPDFGeneration(t *testing.T) {
	t.Parallel()

	suite.Run(t, new(TestSuite))
}
