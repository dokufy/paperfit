package server_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/dokufy/paperfit/config"
	"github.com/dokufy/paperfit/internal/cache"
	"github.com/dokufy/paperfit/internal/httptool"
	"github.com/dokufy/paperfit/internal/server"
	"github.com/dokufy/paperfit/test"
	"github.com/dokufy/paperfit/test/mocks"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type ArchiveUrlCachedTestSuite struct {
	suite.Suite
	ctx           context.Context
	server        *server.Server
	cacheInstance *cache.Cache
	assert        *assert.Assertions
}

var archiveUrlCachedTestHeader = map[string]string{
	"Content-Type": echo.MIMEApplicationJSON,
}

func (s *ArchiveUrlCachedTestSuite) SetupSuite() {
	config.Load()

	mocks.PassPDFgenerator()
	mocks.PassPath("/pdf/from/archive/url")

	server, err := server.New()
	if err != nil {
		s.T().Fatal(err)
	}

	go server.Start(config.AppConfig.ServerPort)

	cacheInstance, err := cache.New(config.AppConfig.CacheConfig.CacheUrl)
	if err != nil {
		s.T().Fatal("failed to create cache")
	}

	s.ctx = context.Background()
	s.server = server
	s.cacheInstance = cacheInstance
	s.assert = assert.New(s.T())
}

func (s *ArchiveUrlCachedTestSuite) Test_Success_Generate_200() {
	mocks.MockFileDownload(mocks.ArchiveMockEndpoint)

	resp := test.Post(
		test.FullURL("/pdf/from/archive/url"),
		archiveUrlCachedTestHeader,
		test.PrepareBody(
			s.T(),
			map[string]any{
				"url": mocks.ArchiveMockEndpoint.Url,
			},
		),
	)

	s.assert.Equal(http.StatusOK, resp.StatusCode)

	cacheKey, _ := cache.Key(mocks.ArchiveMockEndpoint.Url)

	reader, err := s.cacheInstance.Get(s.ctx, cacheKey, "pdf")
	if err != nil {
		s.T().Fatal(err)
	}

	test.AssertReaderNotEmpty(s.Assertions, reader)

	s.cacheInstance.Delete(s.ctx, cacheKey, "pdf")
}

func (s *ArchiveUrlCachedTestSuite) Test_Success_Generate_With_Variables_200() {
	mocks.MockFileDownload(mocks.ArchiveWithJsonSchemaMockEndpoint)

	variables := map[string]string{"title": "this is a title"}
	json, _ := json.Marshal(variables)

	resp := test.Post(
		test.FullURL("/pdf/from/archive/url"),
		archiveUrlCachedTestHeader,
		test.PrepareBody(
			s.T(),
			map[string]any{
				"url": mocks.ArchiveWithJsonSchemaMockEndpoint.Url,
				"input": map[string]any{
					"variables": variables,
				},
			},
		),
	)

	s.assert.Equal(http.StatusOK, resp.StatusCode)

	cacheKey, _ := cache.Key(mocks.ArchiveWithJsonSchemaMockEndpoint.Url, string(json))

	reader, err := s.cacheInstance.Get(s.ctx, cacheKey, "pdf")
	if err != nil {
		s.T().Fatal(err)
	}

	test.AssertReaderNotEmpty(s.Assertions, reader)

	s.cacheInstance.Delete(s.ctx, cacheKey, "pdf")
}

func (s *ArchiveUrlCachedTestSuite) Test_Success_Generate_With_Validation_And_Variables_200() {
	mocks.MockFileDownload(mocks.ArchiveWithJsonSchemaMockEndpoint)

	variables := map[string]string{"title": "this is a title"}
	json, _ := json.Marshal(variables)

	resp := test.Post(
		test.FullURL("/pdf/from/archive/url"),
		archiveUrlCachedTestHeader,
		test.PrepareBody(
			s.T(),
			map[string]any{
				"url": mocks.ArchiveWithJsonSchemaMockEndpoint.Url,
				"input": map[string]any{
					"variables": variables,
					"validate":  true,
				},
			},
		),
	)

	s.assert.Equal(http.StatusOK, resp.StatusCode)

	cacheKey, _ := cache.Key(mocks.ArchiveWithJsonSchemaMockEndpoint.Url, string(json))

	reader, err := s.cacheInstance.Get(s.ctx, cacheKey, "pdf")
	if err != nil {
		s.T().Fatal(err)
	}

	test.AssertReaderNotEmpty(s.Assertions, reader)

	s.cacheInstance.Delete(s.ctx, cacheKey, "pdf")
}

func (s *ArchiveUrlCachedTestSuite) Test_Success_Generate_Use_Cache_200() {
	mocks.MockFileDownload(mocks.ArchiveWithJsonSchemaMockEndpoint)

	additionalHeaders := map[string]string{
		"Authorization": "Bearer test",
		"Content-Type":  echo.MIMEApplicationJSON,
	}
	variables := map[string]string{"title": "this is a title"}
	json, _ := json.Marshal(variables)

	resp1 := test.Post(
		test.FullURL("/pdf/from/archive/url"),
		archiveUrlCachedTestHeader,
		test.PrepareBody(
			s.T(),
			map[string]any{
				"url":               mocks.ArchiveWithJsonSchemaMockEndpoint.Url,
				"additionalHeaders": additionalHeaders,
				"input": map[string]any{
					"variables": variables,
					"validate":  true,
				},
			},
		),
	)

	s.assert.Equal(http.StatusOK, resp1.StatusCode)

	cacheKey, _ := cache.Key(mocks.ArchiveWithJsonSchemaMockEndpoint.Url, httptool.FlattenHeaders(&additionalHeaders), string(json))

	reader, err := s.cacheInstance.Get(s.ctx, cacheKey, "pdf")
	if err != nil {
		s.T().Fatal(err)
	}

	test.AssertReaderNotEmpty(s.Assertions, reader)

	resp2 := test.Post(
		test.FullURL("/pdf/from/archive/url"),
		archiveUrlCachedTestHeader,
		test.PrepareBody(
			s.T(),
			map[string]any{
				"url":               mocks.ArchiveWithJsonSchemaMockEndpoint.Url,
				"additionalHeaders": additionalHeaders,
				"input": map[string]any{
					"variables": variables,
					"validate":  true,
				},
			},
		),
	)

	s.assert.Equal(http.StatusOK, resp2.StatusCode)

	reader, err = s.cacheInstance.Get(s.ctx, cacheKey, "pdf")
	if err != nil {
		s.T().Fatal(err)
	}

	test.AssertReaderNotEmpty(s.Assertions, reader)

	s.cacheInstance.Delete(s.ctx, cacheKey, "pdf")
	s.cacheInstance.Delete(s.ctx, cacheKey, "pdf")
}

func (s *ArchiveUrlCachedTestSuite) Test_Success_Generate_Different_Headers_200() {
	mocks.MockFileDownload(mocks.ArchiveMockEndpoint)

	additionalHeaders := map[string]string{
		"Authorization": "Bearer test",
		"Content-Type":  echo.MIMEApplicationJSON,
	}

	resp := test.Post(
		test.FullURL("/pdf/from/archive/url"),
		archiveUrlCachedTestHeader,
		test.PrepareBody(
			s.T(),
			map[string]any{
				"url":               mocks.ArchiveMockEndpoint.Url,
				"additionalHeaders": additionalHeaders,
			},
		),
	)

	s.assert.Equal(http.StatusOK, resp.StatusCode)

	cacheKey1, err := cache.Key(mocks.ArchiveMockEndpoint.Url, httptool.FlattenHeaders(&additionalHeaders))

	reader, err := s.cacheInstance.Get(s.ctx, cacheKey1, "pdf")
	if err != nil {
		s.T().Fatal(err)
	}

	test.AssertReaderNotEmpty(s.Assertions, reader)

	additionalHeaders = map[string]string{
		"Authorization": "Bearer newtest",
		"Content-Type":  echo.MIMEApplicationJSON,
	}

	resp = test.Post(
		test.FullURL("/pdf/from/archive/url"),
		archiveUrlCachedTestHeader,
		test.PrepareBody(
			s.T(),
			map[string]any{
				"url":               mocks.ArchiveMockEndpoint.Url,
				"additionalHeaders": additionalHeaders,
			},
		),
	)

	s.assert.Equal(http.StatusOK, resp.StatusCode)

	cacheKey2, _ := cache.Key(mocks.ArchiveMockEndpoint.Url, httptool.FlattenHeaders(&additionalHeaders))

	reader, err = s.cacheInstance.Get(s.ctx, cacheKey2, "pdf")
	if err != nil {
		s.T().Fatal(err)
	}

	test.AssertReaderNotEmpty(s.Assertions, reader)

	s.cacheInstance.Delete(s.ctx, cacheKey1, "pdf")
	s.cacheInstance.Delete(s.ctx, cacheKey2, "pdf")
}

func (s *ArchiveUrlCachedTestSuite) Test_Success_Generate_Different_Variables_200() {
	mocks.MockFileDownload(mocks.ArchiveMockEndpoint)

	variables := map[string]string{"title": "this is a title"}
	jsonVars, _ := json.Marshal(variables)

	resp := test.Post(
		test.FullURL("/pdf/from/archive/url"),
		archiveUrlCachedTestHeader,
		test.PrepareBody(
			s.T(),
			map[string]any{
				"url": mocks.ArchiveMockEndpoint.Url,
				"input": map[string]any{
					"variables": variables,
				},
			},
		),
	)

	s.assert.Equal(http.StatusOK, resp.StatusCode)

	cacheKey1, _ := cache.Key(mocks.ArchiveMockEndpoint.Url, string(jsonVars))

	reader, err := s.cacheInstance.Get(s.ctx, cacheKey1, "pdf")
	if err != nil {
		s.T().Fatal(err)
	}

	test.AssertReaderNotEmpty(s.Assertions, reader)

	variables = map[string]string{"title": "new title"}
	jsonVars, _ = json.Marshal(variables)

	resp = test.Post(
		test.FullURL("/pdf/from/archive/url"),
		archiveUrlCachedTestHeader,
		test.PrepareBody(
			s.T(),
			map[string]any{
				"url": mocks.ArchiveMockEndpoint.Url,
				"input": map[string]any{
					"variables": variables,
				},
			},
		),
	)

	s.assert.Equal(http.StatusOK, resp.StatusCode)

	cacheKey2, _ := cache.Key(mocks.ArchiveMockEndpoint.Url, string(jsonVars))

	reader, err = s.cacheInstance.Get(s.ctx, cacheKey2, "pdf")
	if err != nil {
		s.T().Fatal(err)
	}

	test.AssertReaderNotEmpty(s.Assertions, reader)

	s.cacheInstance.Delete(s.ctx, cacheKey1, "pdf")
	s.cacheInstance.Delete(s.ctx, cacheKey2, "pdf")
}

func (s *ArchiveUrlCachedTestSuite) Test_Fail_Generate_No_Url_400() {
	variables := map[string]string{"title": "this is a title"}

	resp := test.Post(
		test.FullURL("/pdf/from/archive/url"),
		archiveUrlCachedTestHeader,
		test.PrepareBody(
			s.T(),
			map[string]any{
				"input": map[string]any{
					"variables": variables,
				},
			},
		),
	)

	s.assert.Equal(400, resp.StatusCode)
	s.assert.Contains(test.StringBody(s.T(), resp.Body), server.ErrNoUrlProvided.Error())
}

func (s *ArchiveUrlCachedTestSuite) Test_Fail_Generate_Bad_Url_404() {
	mocks.MockFileDownload(mocks.ArchiveNotExistingMockEndpoint)

	variables := map[string]string{"title": "this is a title"}

	resp := test.Post(
		test.FullURL("/pdf/from/archive/url"),
		archiveUrlCachedTestHeader,
		test.PrepareBody(
			s.T(),
			map[string]any{
				"url": mocks.ArchiveNotExistingMockEndpoint.Url,
				"input": map[string]any{
					"variables": variables,
				},
			},
		),
	)

	s.assert.Equal(404, resp.StatusCode)
	s.assert.Contains(test.StringBody(s.T(), resp.Body), server.ErrArchiveNotFound.Error())
}

func (s *ArchiveUrlCachedTestSuite) Test_Fail_Generate_With_Validation_No_Variables_400() {
	mocks.MockFileDownload(mocks.ArchiveWithJsonSchemaMockEndpoint)

	resp := test.Post(
		test.FullURL("/pdf/from/archive/url"),
		archiveUrlCachedTestHeader,
		test.PrepareBody(
			s.T(),
			map[string]any{
				"url": mocks.ArchiveWithJsonSchemaMockEndpoint.Url,
				"input": map[string]any{
					"validate": true,
				},
			},
		),
	)

	s.assert.Equal(400, resp.StatusCode)
	s.assert.Contains(test.StringBody(s.T(), resp.Body), server.ErrBadVariables.Error())
}

func (s *ArchiveUrlCachedTestSuite) Test_Fail_Generate_With_Validation_And_Variables_Empty_Schema_400() {
	mocks.MockFileDownload(mocks.ArchiveWithEmptyJsonSchemaMockEndpoint)

	variables := map[string]string{"title": "this is a title"}

	resp := test.Post(
		test.FullURL("/pdf/from/archive/url"),
		archiveUrlCachedTestHeader,
		test.PrepareBody(
			s.T(),
			map[string]any{
				"url": mocks.ArchiveWithEmptyJsonSchemaMockEndpoint.Url,
				"input": map[string]any{
					"variables": variables,
					"validate":  true,
				},
			},
		),
	)

	s.assert.Equal(400, resp.StatusCode)
	s.assert.Contains(test.StringBody(s.T(), resp.Body), server.ErrNoSchemaFile.Error())

	cacheKey, _ := cache.Key(mocks.ArchiveWithEmptyJsonSchemaMockEndpoint.Url)

	reader, err := s.cacheInstance.Get(s.ctx, cacheKey, "pdf")
	if err != nil {
		s.T().Fatal(err)
	}

	s.assert.Empty(reader)
}

func (s *ArchiveUrlCachedTestSuite) Test_Fail_Generate_Bad_Variables_400() {
	mocks.MockFileDownload(mocks.ArchiveWithJsonSchemaMockEndpoint)

	resp := test.Post(
		test.FullURL("/pdf/from/archive/url"),
		archiveUrlCachedTestHeader,
		test.PrepareBody(
			s.T(),
			map[string]any{
				"url": mocks.ArchiveWithJsonSchemaMockEndpoint.Url,
				"input": map[string]any{
					"variables": "bad variables",
					"validate":  true,
				},
			},
		),
	)

	s.assert.Equal(400, resp.StatusCode)
	s.assert.Contains(test.StringBody(s.T(), resp.Body), server.ErrBadRequest.Error())
}

func (s *ArchiveUrlCachedTestSuite) Test_Fail_Generate_Variables_Validation_Fails_400() {
	mocks.MockFileDownload(mocks.ArchiveWithJsonSchemaMockEndpoint)

	variables := map[string]string{"wrong": "wrong variable"}

	resp := test.Post(
		test.FullURL("/pdf/from/archive/url"),
		archiveUrlCachedTestHeader,
		test.PrepareBody(
			s.T(),
			map[string]any{
				"url": mocks.ArchiveWithJsonSchemaMockEndpoint.Url,
				"input": map[string]any{
					"variables": variables,
					"validate":  true,
				},
			},
		),
	)

	s.assert.Equal(400, resp.StatusCode)
	s.assert.Contains(test.StringBody(s.T(), resp.Body), server.ErrBadVariables.Error())

	cacheKey, _ := cache.Key(mocks.ArchiveWithJsonSchemaMockEndpoint.Url)

	reader, err := s.cacheInstance.Get(s.ctx, cacheKey, "pdf")
	if err != nil {
		s.T().Fatal(err)
	}

	s.assert.Empty(reader)
}

func TestArchiveUrlCached(t *testing.T) {
	t.Parallel()

	suite.Run(t, new(ArchiveUrlCachedTestSuite))
}
