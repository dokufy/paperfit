package mocks

import (
	"github.com/dokufy/paperfit/config"
	"github.com/dokufy/paperfit/test"
	"github.com/h2non/gock"
)

type MockEnpointInfo struct {
	ID   string
	Url  string
	Path string
}

var ArchiveMockEndpoint = MockEnpointInfo{
	ID:   "Valid simple archive",
	Url:  "http://localhost/valid/simple",
	Path: "/valid/simple",
}

var ArchiveWithJsonSchemaMockEndpoint = MockEnpointInfo{
	ID:   "Valid archive with json schema",
	Url:  "http://localhost/valid/with-json-schema",
	Path: "/valid/with-json-schema",
}

var ArchiveWithEmptyJsonSchemaMockEndpoint = MockEnpointInfo{
	ID:   "Valid archive without json schema",
	Url:  "http://localhost/valid/with-empty-json-schema",
	Path: "/valid/with-empty-json-schema",
}

var ArchiveWithoutHtmlMockEndpoint = MockEnpointInfo{
	ID:   "Invalid no html file in archive",
	Url:  "http://localhost/invalid/no-html",
	Path: "/invalid/no-html",
}

var ArchiveNotExistingMockEndpoint = MockEnpointInfo{
	ID:   "Not existing archive",
	Url:  "http://localhost/not-existing",
	Path: "/not-existing",
}

func PassPDFgenerator() {
	config.Load()

	gock.New(config.Config.PDFgeneratorEndpoint).
		Post("/").
		Persist().
		EnableNetworking()
}

func PassPath(path string) {
	config.Load()

	gock.New("http://localhost:" + config.Config.ServerPort).
		Post(path).
		Persist().
		EnableNetworking()
}

func MockFileDownload(endpointInfo MockEnpointInfo) {
	switch endpointInfo.ID {
	case ArchiveMockEndpoint.ID:
		ExplicitMockFileDownload(ArchiveMockEndpoint.Path, "archive/valid/simple.tar.gz")
		break
	case ArchiveWithJsonSchemaMockEndpoint.ID:
		ExplicitMockFileDownload(ArchiveWithJsonSchemaMockEndpoint.Path, "archive/valid/with_json_schema.tar.gz")
		break
	case ArchiveWithEmptyJsonSchemaMockEndpoint.ID:
		ExplicitMockFileDownload(ArchiveWithEmptyJsonSchemaMockEndpoint.Path, "archive/valid/with_empty_json_schema.tar.gz")
		break
	case ArchiveWithoutHtmlMockEndpoint.ID:
		ExplicitMockFileDownload(ArchiveWithoutHtmlMockEndpoint.Path, "archive/invalid/no-html.tar.gz")
		break
	case ArchiveNotExistingMockEndpoint.ID:
		MockNotExistingFile(ArchiveNotExistingMockEndpoint.Path)
	default:
		panic("mocked path is not supported")
	}
}

func ExplicitMockFileDownload(path string, relFilePath string) {
	gock.New("http://localhost").
		Get(path).
		Persist().
		Reply(200).
		File(test.FilePath(relFilePath))
}

func MockNotExistingFile(path string) {
	gock.New("http://localhost").
		Get(path).
		Persist().
		Reply(404)
}
