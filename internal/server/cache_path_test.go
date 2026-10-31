package server_test

import (
	"context"
	"net/http"
	"os"
	"testing"

	"github.com/dokufy/paperfit/config"
	"github.com/dokufy/paperfit/internal/cache"
	"github.com/dokufy/paperfit/internal/server"
	"github.com/dokufy/paperfit/test"
	"github.com/dokufy/paperfit/test/mocks"
	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type CachePathTestSuite struct {
	suite.Suite
	ctx                context.Context
	server             *server.Server
	cacheInstance      *cache.Cache
	assert             *assert.Assertions
	defaultPDFpath     string
	defaultArchivePath string
}

var cachePathTestHeader = map[string]string{
	"Content-Type": echo.MIMEApplicationJSON,
}

func (s *CachePathTestSuite) SetupSuite() {
	test.UseCacheURL()
	config.Load()

	s.defaultPDFpath = "/my-customers/pdf/"
	s.defaultArchivePath = "/my-customers/archive/"

	mocks.PassPDFgenerator()
	mocks.PassPath("/pdf/from/archive/url")

	_ = os.Setenv("CACHE_DEFAULT_PDF_PATH", s.defaultPDFpath)
	_ = os.Setenv("CACHE_DEFAULT_ARCHIVE_PATH", s.defaultArchivePath)

	server, err := server.New()
	if err != nil {
		s.T().Fatal(err)
	}

	go func() { _ = server.Start(config.AppConfig.ServerPort) }()
	test.WaitForServer(s.T())

	cacheInstance, err := cache.New(config.AppConfig.CacheConfig.CacheUrl)
	if err != nil {
		s.T().Fatal("failed to create cache")
	}

	s.ctx = context.Background()
	s.server = server
	s.cacheInstance = cacheInstance
	s.assert = assert.New(s.T())
}

func (s *CachePathTestSuite) TearDownSuite() {
	s.server.Stop()

	_ = os.Unsetenv("CACHE_DEFAULT_PDF_PATH")
	_ = os.Unsetenv("CACHE_DEFAULT_ARCHIVE_PATH")
}

func (s *CachePathTestSuite) Test_Default_Cache_Paths() {
	mocks.MockFileDownload(mocks.ArchiveMockEndpoint)

	resp := test.Post(
		test.FullURL("/pdf/from/archive/url"),
		cachePathTestHeader,
		test.PrepareBody(
			s.T(),
			map[string]any{
				"url": mocks.ArchiveMockEndpoint.Url,
			},
		),
	)

	s.assert.Equal(http.StatusOK, resp.StatusCode)

	cacheKey, _ := cache.Key(mocks.ArchiveMockEndpoint.Url)

	reader, err := s.cacheInstance.Get(s.ctx, cacheKey, s.defaultPDFpath)
	if err != nil {
		s.T().Fatal(err)
	}

	test.AssertReaderNotEmpty(s.Assertions, reader)

	reader, err = s.cacheInstance.Get(s.ctx, cacheKey, s.defaultArchivePath)
	if err != nil {
		s.T().Fatal(err)
	}

	test.AssertReaderNotEmpty(s.Assertions, reader)

	_ = s.cacheInstance.Delete(s.ctx, cacheKey, s.defaultPDFpath)
	_ = s.cacheInstance.Delete(s.ctx, cacheKey, s.defaultArchivePath)
}

func (s *CachePathTestSuite) Test_Specified_Cache_Path() {
	mocks.MockFileDownload(mocks.ArchiveMockEndpoint)

	rootPath := "/user/1234/documents"

	resp := test.Post(
		test.FullURL("/pdf/from/archive/url"),
		cachePathTestHeader,
		test.PrepareBody(
			s.T(),
			map[string]any{
				"url":      mocks.ArchiveMockEndpoint.Url,
				"rootPath": rootPath,
			},
		),
	)

	s.assert.Equal(http.StatusOK, resp.StatusCode)

	cacheKey, _ := cache.Key(mocks.ArchiveMockEndpoint.Url)

	reader, err := s.cacheInstance.Get(s.ctx, cacheKey, rootPath)
	if err != nil {
		s.T().Fatal(err)
	}

	test.AssertReaderNotEmpty(s.Assertions, reader)

	reader, err = s.cacheInstance.Get(s.ctx, cacheKey, s.defaultArchivePath)
	if err != nil {
		s.T().Fatal(err)
	}

	test.AssertReaderNotEmpty(s.Assertions, reader)

	_ = s.cacheInstance.Delete(s.ctx, cacheKey, rootPath)
	_ = s.cacheInstance.Delete(s.ctx, cacheKey, s.defaultArchivePath)
}

func TestCachePath(t *testing.T) {
	suite.Run(t, new(CachePathTestSuite))
}
