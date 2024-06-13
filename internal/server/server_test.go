package server_test

import (
	"context"
	"testing"

	"github.com/atoncetti/combiner/config"
	"github.com/atoncetti/combiner/internal/cacher"
	"github.com/atoncetti/combiner/internal/logger"
	"github.com/atoncetti/combiner/internal/server"
	"github.com/atoncetti/combiner/internal/service/pdfprovider"
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
	cacher *cacher.Cacher
	logger logger.Logger
}

func (s *TestSuite) SetupSuite() {
	config.Load()
	logger := logger.New()
	cacher, err := cacher.New(config.Config.CacheConfig.CacheUrl, nil)
	if err != nil {
		s.T().Fatal("failed to create cache")
	}

	s.cacher = cacher
	s.logger = logger
	s.assert = assert.New(s.T())
}

func setupApiWithCache(s *TestSuite) {
	_, api := humatest.New(s.T())

	httpPDFprovider := pdfprovider.NewHTTPpdfProvider(config.Config.PDFgeneratorEndpoint)

	addRoutes(api, httpPDFprovider, s.cacher, &s.logger)

	s.api = api
}

func setupApiWithoutCache(s *TestSuite) {
	_, api := humatest.New(s.T())

	httpPDFprovider := pdfprovider.NewHTTPpdfProvider(config.Config.PDFgeneratorEndpoint)

	addRoutes(api, httpPDFprovider, nil, &s.logger)

	s.api = api
}

func addRoutes(api humatest.TestAPI, pdfProvider *pdfprovider.HTTPpdfProvider, cacher *cacher.Cacher, logger *logger.Logger) {
	server.AddRoutes(api, pdfProvider, cacher, logger)
}

func (s *TestSuite) Test_200_Explicit_Generate() {
	mocks.NoMockPDFgenerator()

	setupApiWithoutCache(s)

	project := mocks.ProjectMock("test")

	multipart, err := pdfprovider.MultipartForm(project.GetFiles(), nil)
	if err != nil {
		s.T().Fatal(err)
	}

	resp := s.api.Post("/pdf/generate/html", "Content-type: "+multipart.FormDataContentType(), multipart.Body())

	s.assert.Equal(200, resp.Code)
}

func (s *TestSuite) Test_200_No_Cache_Archive_Generate() {
	mocks.NoMockPDFgenerator()
	mocks.MockHTTPfileRequest(mocks.SimpleFileMockEndpoint)

	setupApiWithoutCache(s)

	body := map[string]string{"url": mocks.SimpleFileMockEndpoint.Url}
	resp := s.api.Post("/pdf/generate/archive", "application/json", body)

	s.assert.Equal(200, resp.Code)
}

func (s *TestSuite) Test_200_With_Cache_Archive_Generate() {
	mocks.NoMockPDFgenerator()
	mocks.MockHTTPfileRequest(mocks.SimpleFileMockEndpoint)

	ctx := context.Background()

	setupApiWithCache(s)

	body := map[string]string{"url": mocks.SimpleFileMockEndpoint.Url}
	resp1 := s.api.Post("/pdf/generate/archive", "application/json", body)

	mocks.RemoveMockedRequests()
	resp2 := s.api.Post("/pdf/generate/archive", "application/json", body)

	s.assert.Equal(200, resp1.Code)
	s.assert.Equal(200, resp2.Code)

	archiveCacheKey, err := cacher.Key([][]byte{[]byte(mocks.SimpleFileMockEndpoint.Url)})
	if err != nil {
		s.T().Fatal(err)
	}

	pdfCacheKey, err := cacher.Key([][]byte{[]byte(mocks.SimpleFileMockEndpoint.Url)})
	if err != nil {
		s.T().Fatal(err)
	}

	s.cacher.DeleteArchive(ctx, archiveCacheKey)
	s.cacher.DeletePDF(ctx, pdfCacheKey)
}

func (s *TestSuite) Test_Fail_400_Archive_Generate_No_URL_Provided() {
	setupApiWithoutCache(s)

	body := map[string]string{"name": "John Doe", "occupation": "gardener"}
	resp := s.api.Post("/pdf/generate/archive", "application/json", body)

	s.assert.Equal(400, resp.Code)
}

func (s *TestSuite) Test_Fail_400_Archive_Generate_No_HTML_In_Archive() {
	mocks.NoMockPDFgenerator()
	mocks.MockHTTPfileRequest(mocks.NoHtmlFileMockEndpoint)

	setupApiWithoutCache(s)

	body := map[string]string{"url": mocks.NoHtmlFileMockEndpoint.Url}
	resp := s.api.Post("/pdf/generate/archive", "application/json", body)

	s.assert.Equal(400, resp.Code)
}

func TestPDFGeneration(t *testing.T) {
	t.Parallel()

	suite.Run(t, new(TestSuite))
}
