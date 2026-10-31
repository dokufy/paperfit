package mcp

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dokufy/paperfit/config"
	"github.com/dokufy/paperfit/internal/logger"
	"github.com/dokufy/paperfit/internal/service/pdfgenerator"
)

func testServer(t *testing.T) *Server {
	t.Helper()
	logger.Initialize()
	config.Load()
	return New(pdfgenerator.New(config.AppConfig.PDFgeneratorEndpoint))
}

func simpleArchiveBase64(t *testing.T) string {
	t.Helper()
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)
	content := "<html><body>hello</body></html>"
	hdr := &tar.Header{Name: "index.html", Mode: 0o644, Size: int64(len(content))}
	if err := tw.WriteHeader(hdr); err != nil {
		t.Fatal(err)
	}
	_, _ = tw.Write([]byte(content))
	_ = tw.Close()
	_ = gw.Close()
	return base64.StdEncoding.EncodeToString(buf.Bytes())
}

func call(t *testing.T, s *Server, method string, params interface{}) map[string]interface{} {
	t.Helper()
	req := map[string]interface{}{"jsonrpc": "2.0", "id": 1, "method": method}
	if params != nil {
		b, _ := json.Marshal(params)
		req["params"] = json.RawMessage(b)
	}
	raw, _ := json.Marshal(req)
	respBytes, isNotif, err := s.HandleRequest(context.Background(), raw)
	if err != nil {
		t.Fatalf("HandleRequest error: %v", err)
	}
	if isNotif {
		return nil
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(respBytes, &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	return resp
}

func TestMCPInitialize(t *testing.T) {
	s := testServer(t)
	resp := call(t, s, "initialize", map[string]interface{}{"protocolVersion": "2024-11-05"})
	if resp["error"] != nil {
		t.Fatalf("unexpected error: %v", resp["error"])
	}
	res := resp["result"].(map[string]interface{})
	if res["protocolVersion"] != "2024-11-05" {
		t.Errorf("protocolVersion mismatch: %v", res["protocolVersion"])
	}
}

func TestMCPPing(t *testing.T) {
	s := testServer(t)
	resp := call(t, s, "ping", nil)
	if resp["error"] != nil {
		t.Fatalf("unexpected error: %v", resp["error"])
	}
}

func TestMCPNotificationsInitialized(t *testing.T) {
	s := testServer(t)
	req := map[string]interface{}{"jsonrpc": "2.0", "method": "notifications/initialized"}
	raw, _ := json.Marshal(req)
	respBytes, isNotif, err := s.HandleRequest(context.Background(), raw)
	if err != nil {
		t.Fatal(err)
	}
	if !isNotif {
		t.Fatalf("expected notification, got %s", string(respBytes))
	}
}

func TestMCPToolsList(t *testing.T) {
	s := testServer(t)
	resp := call(t, s, "tools/list", nil)
	res := resp["result"].(map[string]interface{})
	tools := res["tools"].([]interface{})
	if len(tools) != 6 {
		t.Errorf("expected 6 tools, got %d", len(tools))
	}
}

func TestMCPToolsCallAll(t *testing.T) {
	s := testServer(t)

	// render_pdf_from_html
	resp := call(t, s, "tools/call", map[string]interface{}{
		"name":      "render_pdf_from_html",
		"arguments": map[string]interface{}{"html": "<html><body>hi</body></html>"},
	})
	assertToolResult(t, resp)

	// render_pdf_from_archive
	resp = call(t, s, "tools/call", map[string]interface{}{
		"name":      "render_pdf_from_archive",
		"arguments": map[string]interface{}{"archive_base64": simpleArchiveBase64(t)},
	})
	assertToolResult(t, resp)

	// render_pdf_from_archive_url via httptest
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, _ := base64.StdEncoding.DecodeString(simpleArchiveBase64(t))
		_, _ = w.Write(data)
	}))
	defer ts.Close()
	resp = call(t, s, "tools/call", map[string]interface{}{
		"name":      "render_pdf_from_archive_url",
		"arguments": map[string]interface{}{"url": ts.URL},
	})
	assertToolResult(t, resp)

	// validate_template_variables
	resp = call(t, s, "tools/call", map[string]interface{}{
		"name": "validate_template_variables",
		"arguments": map[string]interface{}{
			"variables":   map[string]interface{}{"title": "hi"},
			"json_schema": map[string]interface{}{"type": "object", "properties": map[string]interface{}{"title": map[string]interface{}{"type": "string"}}, "required": []string{"title"}},
		},
	})
	assertToolResult(t, resp)

	// inspect_template_archive via url
	ts2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, _ := base64.StdEncoding.DecodeString(simpleArchiveBase64(t))
		_, _ = w.Write(data)
	}))
	defer ts2.Close()
	resp = call(t, s, "tools/call", map[string]interface{}{
		"name":      "inspect_template_archive",
		"arguments": map[string]interface{}{"url": ts2.URL},
	})
	assertToolResult(t, resp)

	// scaffold
	resp = call(t, s, "tools/call", map[string]interface{}{
		"name":      "scaffold_paperfit_template",
		"arguments": map[string]interface{}{"document_type": "invoice", "framework": "react"},
	})
	assertToolResult(t, resp)
}

func assertToolResult(t *testing.T, resp map[string]interface{}) {
	t.Helper()
	if resp["error"] != nil {
		t.Fatalf("unexpected error: %v", resp["error"])
	}
	res := resp["result"].(map[string]interface{})
	content, ok := res["content"].([]interface{})
	if !ok || len(content) == 0 {
		t.Fatalf("missing content: %v", res)
	}
	c0 := content[0].(map[string]interface{})
	if c0["type"] != "text" {
		t.Errorf("content type mismatch: %v", c0["type"])
	}
	if _, ok := res["structuredContent"]; !ok {
		t.Errorf("missing structuredContent: %v", res)
	}
}

func TestMCPResourcesListAndRead(t *testing.T) {
	s := testServer(t)
	resp := call(t, s, "resources/list", nil)
	res := resp["result"].(map[string]interface{})
	if len(res["resources"].([]interface{})) == 0 {
		t.Fatal("no resources")
	}
	for _, uri := range []string{"paperfit://docs/authoring-guide", "paperfit://examples/invoice"} {
		r := call(t, s, "resources/read", map[string]interface{}{"uri": uri})
		rr := r["result"].(map[string]interface{})
		contents := rr["contents"].([]interface{})
		if len(contents) == 0 {
			t.Fatalf("no contents for %s", uri)
		}
	}
}

func TestMCPPromptsListAndGet(t *testing.T) {
	s := testServer(t)
	resp := call(t, s, "prompts/list", nil)
	res := resp["result"].(map[string]interface{})
	if len(res["prompts"].([]interface{})) == 0 {
		t.Fatal("no prompts")
	}
	r := call(t, s, "prompts/get", map[string]interface{}{"name": "create-pdf-template"})
	if r["error"] != nil {
		t.Fatalf("prompt get error: %v", r["error"])
	}
}

func TestMCPInvalidJSON(t *testing.T) {
	s := testServer(t)
	respBytes, isNotif, err := s.HandleRequest(context.Background(), []byte("{bad json"))
	if err != nil {
		t.Fatal(err)
	}
	if isNotif {
		t.Fatal("expected response")
	}
	var resp map[string]interface{}
	_ = json.Unmarshal(respBytes, &resp)
	if resp["error"] == nil {
		t.Fatal("expected parse error")
	}
	errObj := resp["error"].(map[string]interface{})
	if int(errObj["code"].(float64)) != -32700 {
		t.Errorf("expected -32700, got %v", errObj["code"])
	}
}

func TestMCPUnknownMethodAndTool(t *testing.T) {
	s := testServer(t)
	resp := call(t, s, "foo/bar", nil)
	if resp["error"] == nil {
		t.Fatal("expected method not found")
	}
	errObj := resp["error"].(map[string]interface{})
	if int(errObj["code"].(float64)) != -32601 {
		t.Errorf("expected -32601, got %v", errObj["code"])
	}

	resp = call(t, s, "tools/call", map[string]interface{}{
		"name":      "nonexistent_tool",
		"arguments": map[string]interface{}{},
	})
	// callTool returns error -> dispatch wraps? Actually unknown tool returns error from callTool, dispatch error path? Wait callTool is called inside dispatch tools/call; if err, dispatch returns content error with isError, not RPC error. So resp result has isError.
	res := resp["result"].(map[string]interface{})
	if res["isError"] != true {
		t.Errorf("expected isError true: %v", res)
	}
}

func TestMCPLargeIntegerID(t *testing.T) {
	s := testServer(t)
	raw := []byte(`{"jsonrpc":"2.0","id":9007199254740993,"method":"ping"}`)
	respBytes, isNotif, err := s.HandleRequest(context.Background(), raw)
	if err != nil {
		t.Fatal(err)
	}
	if isNotif {
		t.Fatal("expected response")
	}
	if !strings.Contains(string(respBytes), "9007199254740993") {
		t.Fatalf("large id not preserved: %s", string(respBytes))
	}
}

func TestMCPBatchRequest(t *testing.T) {
	s := testServer(t)
	raw := []byte(`[{"jsonrpc":"2.0","id":1,"method":"ping"},{"jsonrpc":"2.0","id":2,"method":"ping"}]`)
	respBytes, isNotif, err := s.HandleRequest(context.Background(), raw)
	if err != nil {
		t.Fatal(err)
	}
	if isNotif {
		t.Fatal("expected response")
	}
	var arr []map[string]interface{}
	if err := json.Unmarshal(respBytes, &arr); err != nil {
		t.Fatalf("expected array: %v", err)
	}
	if len(arr) != 2 {
		t.Fatalf("expected 2 responses, got %d", len(arr))
	}

	// empty array
	respBytes, _, _ = s.HandleRequest(context.Background(), []byte(`[]`))
	var single map[string]interface{}
	_ = json.Unmarshal(respBytes, &single)
	if single["error"] == nil {
		t.Fatalf("expected error for empty batch: %s", string(respBytes))
	}
	errObj := single["error"].(map[string]interface{})
	if int(errObj["code"].(float64)) != -32600 {
		t.Errorf("expected -32600, got %v", errObj["code"])
	}
}

func TestMCPNotificationWithID(t *testing.T) {
	s := testServer(t)
	raw := []byte(`{"jsonrpc":"2.0","id":5,"method":"notifications/foo"}`)
	respBytes, isNotif, err := s.HandleRequest(context.Background(), raw)
	if err != nil {
		t.Fatal(err)
	}
	if isNotif {
		t.Fatal("expected response for notification with id")
	}
	var resp map[string]interface{}
	_ = json.Unmarshal(respBytes, &resp)
	if resp["id"] == nil {
		t.Fatal("missing id")
	}
	if _, ok := resp["result"].(map[string]interface{}); !ok {
		t.Fatalf("expected result object: %v", resp)
	}
}

func TestMCPHTTPRejectsOutputAndArchivePath(t *testing.T) {
	s := testServer(t) // HTTP mode (allowLocalFS false)
	// output_path
	resp := call(t, s, "tools/call", map[string]interface{}{
		"name":      "render_pdf_from_html",
		"arguments": map[string]interface{}{"html": "<html><body>hi</body></html>", "output_path": "/tmp/out.pdf"},
	})
	res := resp["result"].(map[string]interface{})
	content := res["content"].([]interface{})
	c0 := content[0].(map[string]interface{})
	if !strings.Contains(c0["text"].(string), "output_path is not allowed over HTTP MCP") {
		t.Fatalf("unexpected response: %v", res)
	}
	// archive_path
	resp = call(t, s, "tools/call", map[string]interface{}{
		"name":      "render_pdf_from_archive",
		"arguments": map[string]interface{}{"archive_path": "/tmp/a.tar.gz"},
	})
	res = resp["result"].(map[string]interface{})
	content = res["content"].([]interface{})
	c0 = content[0].(map[string]interface{})
	if !strings.Contains(c0["text"].(string), "archive_path is not allowed over HTTP MCP") {
		t.Fatalf("unexpected response: %v", res)
	}
}

func TestMCPStdioAllowsValidPaths(t *testing.T) {
	s := NewStdio(pdfgenerator.New(config.AppConfig.PDFgeneratorEndpoint))
	// output_path
	tmpOut := filepath.Join(os.TempDir(), "paperfit-test-out.pdf")
	resp := call(t, s, "tools/call", map[string]interface{}{
		"name":      "render_pdf_from_html",
		"arguments": map[string]interface{}{"html": "<html><body>hi</body></html>", "output_path": tmpOut},
	})
	if resp["error"] != nil {
		t.Fatalf("unexpected error: %v", resp["error"])
	}
	if _, err := os.Stat(tmpOut); err != nil {
		t.Fatalf("output file not written: %v", err)
	}
	_ = os.Remove(tmpOut)
}

func TestMCPStdioAllowsArchivePath(t *testing.T) {
	s := NewStdio(pdfgenerator.New(config.AppConfig.PDFgeneratorEndpoint))
	data, _ := base64.StdEncoding.DecodeString(simpleArchiveBase64(t))
	tmp := filepath.Join(os.TempDir(), "paperfit-test-archive.tar.gz")
	_ = os.WriteFile(tmp, data, 0o644)
	defer func() { _ = os.Remove(tmp) }()

	resp := call(t, s, "tools/call", map[string]interface{}{
		"name":      "inspect_template_archive",
		"arguments": map[string]interface{}{"archive_path": tmp},
	})
	if resp["error"] != nil {
		t.Fatalf("unexpected error: %v", resp["error"])
	}
	res := resp["result"].(map[string]interface{})
	content := res["content"].([]interface{})
	_ = content
}
