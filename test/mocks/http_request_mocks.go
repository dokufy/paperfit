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

var SimpleFileMockEndpoint = MockEnpointInfo{
	ID:   "Valid simple archive",
	Url:  "http://localhost/valid/simple",
	Path: "/valid/simple",
}

var NoHtmlFileMockEndpoint = MockEnpointInfo{
	ID:   "Invalid no html file in archive",
	Url:  "http://localhost/invalid/no-html",
	Path: "/invalid/no-html",
}

func EnableNetworking() {
	gock.EnableNetworking()
}

func RemoveMockedRequests() {
	gock.Off()
}

func PassPDFgenerator() {
	config.Load()

	gock.New(config.Config.PDFgeneratorEndpoint).
		Post("/").
		Persist().
		EnableNetworking()
}

func MockHTTPfileRequest(endpointInfo MockEnpointInfo) {
	switch endpointInfo.ID {
	case SimpleFileMockEndpoint.ID:
		ExplicitMockHTTPfileRequest(SimpleFileMockEndpoint.Path, "archive/valid/simple.tar.gz")
		break
	case NoHtmlFileMockEndpoint.ID:
		ExplicitMockHTTPfileRequest(NoHtmlFileMockEndpoint.Path, "archive/invalid/no-html.tar.gz")
		break
	default:
		panic("mocked path is not supported")
	}
}

func ExplicitMockHTTPfileRequest(path string, relFilePath string) {
	gock.New("http://localhost").
		Get(path).
		Persist().
		Reply(200).
		File(test.FilePath(relFilePath))
}
