package server_test

import (
	"context"
	"net/http"
	"os"
	"testing"

	"github.com/dokufy/paperfit/config"
	"github.com/dokufy/paperfit/internal/server"
	"github.com/dokufy/paperfit/test"
	"github.com/dokufy/paperfit/test/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type ArchiveUrlNonCachedTestSuite struct {
	suite.Suite
	ctx    context.Context
	server *server.Server
	assert *assert.Assertions
}

func (s *ArchiveUrlNonCachedTestSuite) SetupSuite() {
	os.Setenv("CACHE_URL", "")
	config.Load()

	mocks.PassPDFgenerator()
	mocks.PassPath("/pdf/from/archive/url")

	server, err := server.New()
	if err != nil {
		s.T().Fatal(err)
	}

	go server.Start(config.Config.ServerPort)

	s.ctx = context.Background()
	s.server = server
	s.assert = assert.New(s.T())
}

func (s *ArchiveUrlNonCachedTestSuite) TearDownSuite() {
	s.server.Stop()
}

func (s *ArchiveUrlNonCachedTestSuite) Test_Success_Generate_200() {
	mocks.MockFileDownload(mocks.ArchiveMockEndpoint)

	resp := test.Post(
		test.FullURL("/pdf/from/archive/url"),
		applicationJsonHeader,
		test.PrepareBody(
			s.T(),
			map[string]any{
				"url": mocks.ArchiveMockEndpoint.Url,
			},
		),
	)

	s.assert.Equal(http.StatusOK, resp.StatusCode)
}

func (s *ArchiveUrlNonCachedTestSuite) Test_Success_Generate_With_Variables_200() {
	mocks.MockFileDownload(mocks.ArchiveMockEndpoint)

	resp := test.Post(
		test.FullURL("/pdf/from/archive/url"),
		applicationJsonHeader,
		test.PrepareBody(
			s.T(),
			map[string]any{
				"url": mocks.ArchiveMockEndpoint.Url,
				"input": map[string]any{
					"variables": map[string]any{
						"title": "this is a title",
					},
				},
			},
		),
	)

	s.assert.Equal(http.StatusOK, resp.StatusCode)
}

func (s *ArchiveUrlNonCachedTestSuite) Test_Generate_With_Validation_And_Variables_200() {
	mocks.MockFileDownload(mocks.ArchiveWithJsonSchemaMockEndpoint)

	resp := test.Post(
		test.FullURL("/pdf/from/archive/url"),
		applicationJsonHeader,
		test.PrepareBody(
			s.T(),
			map[string]any{
				"url": mocks.ArchiveWithJsonSchemaMockEndpoint.Url,
				"input": map[string]any{
					"variables": map[string]any{
						"title": "this is a title",
					},
					"validate": true,
				},
			},
		),
	)

	s.assert.Equal(http.StatusOK, resp.StatusCode)
}

func (s *ArchiveUrlNonCachedTestSuite) Test_Success_Generate_With_Variables_Empty_Schema_200() {
	mocks.MockFileDownload(mocks.ArchiveWithEmptyJsonSchemaMockEndpoint)

	resp := test.Post(
		test.FullURL("/pdf/from/archive/url"),
		applicationJsonHeader,
		test.PrepareBody(
			s.T(),
			map[string]any{
				"url": mocks.ArchiveWithEmptyJsonSchemaMockEndpoint.Url,
				"input": map[string]any{
					"variables": map[string]any{
						"test": "some_value",
					},
				},
			},
		),
	)

	s.assert.Equal(http.StatusOK, resp.StatusCode)
}

func (s *ArchiveUrlNonCachedTestSuite) Test_Fail_Generate_No_Url_400() {
	resp := test.Post(
		test.FullURL("/pdf/from/archive/url"),
		applicationJsonHeader,
		test.PrepareBody(
			s.T(),
			map[string]any{},
		),
	)

	s.assert.Equal(http.StatusBadRequest, resp.StatusCode)
	s.assert.Contains(test.StringBody(s.T(), resp.Body), server.ErrNoUrlProvided.Error())
}

func (s *ArchiveUrlNonCachedTestSuite) Test_Fail_Generate_Bad_Url_404() {
	mocks.MockFileDownload(mocks.ArchiveNotExistingMockEndpoint)

	resp := test.Post(
		test.FullURL("/pdf/from/archive/url"),
		applicationJsonHeader,
		test.PrepareBody(
			s.T(),
			map[string]any{"url": mocks.ArchiveNotExistingMockEndpoint.Url},
		),
	)

	s.assert.Equal(http.StatusNotFound, resp.StatusCode)
	s.assert.Contains(test.StringBody(s.T(), resp.Body), server.ErrArchiveNotFound.Error())
}

func (s *ArchiveUrlNonCachedTestSuite) Test_Fail_Generate_Validation_And_Variables_Empty_Schema_400() {
	mocks.MockFileDownload(mocks.ArchiveMockEndpoint)

	resp := test.Post(
		test.FullURL("/pdf/from/archive/url"),
		applicationJsonHeader,
		test.PrepareBody(
			s.T(),
			map[string]any{
				"url": mocks.ArchiveMockEndpoint.Url,
				"input": map[string]any{
					"variables": map[string]any{
						"test": "some_value",
					},
					"validate": true,
				},
			},
		),
	)

	s.assert.Equal(http.StatusBadRequest, resp.StatusCode)
	s.assert.Contains(test.StringBody(s.T(), resp.Body), server.ErrNoSchemaFile.Error())
}

func (s *ArchiveUrlNonCachedTestSuite) Test_Fail_Generate_Variables_Validation_Fails_400() {
	mocks.MockFileDownload(mocks.ArchiveWithJsonSchemaMockEndpoint)

	resp := test.Post(
		test.FullURL("/pdf/from/archive/url"),
		applicationJsonHeader,
		test.PrepareBody(
			s.T(),
			map[string]any{
				"url": mocks.ArchiveWithJsonSchemaMockEndpoint.Url,
				"input": map[string]any{
					"variables": "bad variables",
				},
				"validate": true,
			},
		),
	)

	s.assert.Equal(http.StatusBadRequest, resp.StatusCode)
	s.assert.Contains(test.StringBody(s.T(), resp.Body), server.ErrBadRequest.Error())
}

func (s *ArchiveUrlNonCachedTestSuite) Test_Fail_Generate_Bad_Variables_400() {
	mocks.MockFileDownload(mocks.ArchiveWithJsonSchemaMockEndpoint)

	resp := test.Post(
		test.FullURL("/pdf/from/archive/url"),
		applicationJsonHeader,
		test.PrepareBody(
			s.T(),
			map[string]any{
				"url": mocks.ArchiveWithJsonSchemaMockEndpoint.Url,
				"input": map[string]any{
					"variables": map[string]any{
						"test": "bad variables",
					},
					"validate": true,
				},
			},
		),
	)

	s.assert.Equal(http.StatusBadRequest, resp.StatusCode)
	s.assert.Contains(test.StringBody(s.T(), resp.Body), server.ErrBadVariables.Error())
}

func TestArchiveUrlNonCached(t *testing.T) {
	t.Parallel()

	suite.Run(t, new(ArchiveUrlNonCachedTestSuite))
}
