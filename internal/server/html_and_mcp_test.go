package server_test

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"testing"

	"github.com/dokufy/paperfit/config"
	"github.com/dokufy/paperfit/internal/server"
	"github.com/dokufy/paperfit/test"
	"github.com/dokufy/paperfit/test/mocks"
	"github.com/h2non/gock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type HtmlAndMcpTestSuite struct {
	suite.Suite
	ctx    context.Context
	server *server.Server
	assert *assert.Assertions
}

func (s *HtmlAndMcpTestSuite) SetupSuite() {
	_ = os.Setenv("CACHE_URL", "")
	config.Load()
	mocks.PassPDFgenerator()
	mocks.PassPath("/pdf/from/html")
	gock.Disable()

	server, err := server.New()
	if err != nil {
		s.T().Fatal(err)
	}

	go func() { _ = server.Start(config.AppConfig.ServerPort) }()
	test.WaitForServer(s.T())

	s.ctx = context.Background()
	s.server = server
	s.assert = assert.New(s.T())
}

func (s *HtmlAndMcpTestSuite) TearDownSuite() {
	s.server.Stop()
}

func (s *HtmlAndMcpTestSuite) Test_Healthy_200() {
	resp, err := http.Get(test.FullURL("/application/healthy"))
	s.assert.NoError(err)
	s.assert.Equal(http.StatusOK, resp.StatusCode)
}

func (s *HtmlAndMcpTestSuite) Test_LlmsTxt_200() {
	resp, err := http.Get(test.FullURL("/llms.txt"))
	s.assert.NoError(err)
	s.assert.Equal(http.StatusOK, resp.StatusCode)
	body := test.StringBody(s.T(), resp.Body)
	s.assert.Contains(body, "POST /pdf/from/html")
	s.assert.Contains(body, "window.__PAPERFIT_INPUT__")
}

func (s *HtmlAndMcpTestSuite) Test_Html_Valid_200() {
	body := map[string]any{"html": "<html><body>hi</body></html>"}
	resp := test.Post(test.FullURL("/pdf/from/html"), map[string]string{"Content-Type": "application/json"}, test.PrepareBody(s.T(), body))
	s.assert.Equal(http.StatusOK, resp.StatusCode)
}

func (s *HtmlAndMcpTestSuite) Test_Html_Variables_Schema_Pass_200() {
	body := map[string]any{
		"html":       "<html><body>{{title}}</body></html>",
		"jsonSchema": map[string]any{"type": "object", "properties": map[string]any{"title": map[string]any{"type": "string"}}, "required": []string{"title"}},
		"input":      map[string]any{"validate": true, "variables": map[string]any{"title": "hello"}},
	}
	resp := test.Post(test.FullURL("/pdf/from/html"), map[string]string{"Content-Type": "application/json"}, test.PrepareBody(s.T(), body))
	s.assert.Equal(http.StatusOK, resp.StatusCode)
}

func (s *HtmlAndMcpTestSuite) Test_Html_Variables_Schema_Fail_400() {
	body := map[string]any{
		"html":       "<html><body>{{title}}</body></html>",
		"jsonSchema": map[string]any{"type": "object", "properties": map[string]any{"title": map[string]any{"type": "string"}}, "required": []string{"title"}},
		"input":      map[string]any{"validate": true, "variables": map[string]any{}},
	}
	resp := test.Post(test.FullURL("/pdf/from/html"), map[string]string{"Content-Type": "application/json"}, test.PrepareBody(s.T(), body))
	s.assert.Equal(http.StatusBadRequest, resp.StatusCode)
}

func (s *HtmlAndMcpTestSuite) Test_Html_Assets_Options_200() {
	body := map[string]any{
		"html":    "<html><head><link rel=\"stylesheet\" href=\"style.css\"></head><body>hi</body></html>",
		"assets":  map[string]any{"style.css": "body{color:red;}"},
		"options": map[string]any{"landscape": true, "waitForExpression": "window.__PAPERFIT_READY__ !== false"},
	}
	resp := test.Post(test.FullURL("/pdf/from/html"), map[string]string{"Content-Type": "application/json"}, test.PrepareBody(s.T(), body))
	s.assert.Equal(http.StatusOK, resp.StatusCode)
}

func (s *HtmlAndMcpTestSuite) Test_Html_Missing_400() {
	body := map[string]any{}
	resp := test.Post(test.FullURL("/pdf/from/html"), map[string]string{"Content-Type": "application/json"}, test.PrepareBody(s.T(), body))
	s.assert.Equal(http.StatusBadRequest, resp.StatusCode)
}

func (s *HtmlAndMcpTestSuite) Test_MCP_Over_HTTP() {
	// initialize
	reqBody := map[string]any{"jsonrpc": "2.0", "id": 1, "method": "initialize", "params": map[string]any{"protocolVersion": "2024-11-05"}}
	resp := test.Post(test.FullURL("/mcp"), map[string]string{"Content-Type": "application/json"}, test.PrepareBody(s.T(), reqBody))
	s.assert.Equal(http.StatusOK, resp.StatusCode)
	var r map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&r)
	s.assert.Nil(r["error"])

	// tools/list
	reqBody = map[string]any{"jsonrpc": "2.0", "id": 2, "method": "tools/list"}
	resp = test.Post(test.FullURL("/mcp"), map[string]string{"Content-Type": "application/json"}, test.PrepareBody(s.T(), reqBody))
	s.assert.Equal(http.StatusOK, resp.StatusCode)

	// tools/call render_pdf_from_html
	reqBody = map[string]any{"jsonrpc": "2.0", "id": 3, "method": "tools/call", "params": map[string]any{"name": "render_pdf_from_html", "arguments": map[string]any{"html": "<html><body>hi</body></html>"}}}
	resp = test.Post(test.FullURL("/mcp"), map[string]string{"Content-Type": "application/json"}, test.PrepareBody(s.T(), reqBody))
	s.assert.Equal(http.StatusOK, resp.StatusCode)
	var r3 map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&r3)
	res := r3["result"].(map[string]any)
	s.assert.NotNil(res["content"])
	s.assert.NotNil(res["structuredContent"])
}

func (s *HtmlAndMcpTestSuite) Test_Html_RootPath_Traversal_400() {
	body := map[string]any{"html": "<html><body>hi</body></html>", "rootPath": "../../../../tmp/evil"}
	resp := test.Post(test.FullURL("/pdf/from/html"), map[string]string{"Content-Type": "application/json"}, test.PrepareBody(s.T(), body))
	s.assert.Equal(http.StatusBadRequest, resp.StatusCode)
}

func (s *HtmlAndMcpTestSuite) Test_Html_Assets_Invalid_400() {
	body := map[string]any{
		"html":   "<html><body>hi</body></html>",
		"assets": map[string]any{"index.html": "x", "../evil.js": "y"},
	}
	resp := test.Post(test.FullURL("/pdf/from/html"), map[string]string{"Content-Type": "application/json"}, test.PrepareBody(s.T(), body))
	s.assert.Equal(http.StatusBadRequest, resp.StatusCode)
}

func TestHtmlAndMcp(t *testing.T) {
	suite.Run(t, new(HtmlAndMcpTestSuite))
}
