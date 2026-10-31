package mcp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/dokufy/paperfit/internal/archive"
	"github.com/dokufy/paperfit/internal/service"
	"github.com/dokufy/paperfit/internal/service/pdfgenerator"
	"github.com/dokufy/paperfit/internal/validation"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

type RPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

type Request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type Response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Result  interface{}     `json:"result,omitempty"`
	Error   *RPCError       `json:"error,omitempty"`
}

type Server struct {
	pdfService   *pdfgenerator.GotembergGenerator
	allowLocalFS bool
}

func New(pdfService *pdfgenerator.GotembergGenerator) *Server {
	return &Server{pdfService: pdfService, allowLocalFS: false}
}

func NewStdio(pdfService *pdfgenerator.GotembergGenerator) *Server {
	return &Server{pdfService: pdfService, allowLocalFS: true}
}

func (s *Server) WithLocalFS(allow bool) *Server {
	s.allowLocalFS = allow
	return s
}

func (s *Server) HandleRequest(ctx context.Context, raw []byte) ([]byte, bool, error) {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 {
		return nil, true, nil
	}
	// JSON-RPC batch
	if len(raw) > 0 && raw[0] == '[' {
		var items []json.RawMessage
		if err := json.Unmarshal(raw, &items); err != nil {
			resp := Response{JSONRPC: "2.0", Error: &RPCError{Code: -32700, Message: "Parse error"}}
			b, _ := json.Marshal(resp)
			return b, false, nil
		}
		if len(items) == 0 {
			resp := Response{JSONRPC: "2.0", Error: &RPCError{Code: -32600, Message: "Invalid Request"}}
			b, _ := json.Marshal(resp)
			return b, false, nil
		}
		var out []interface{}
		for _, item := range items {
			rb, isNotif, _ := s.HandleRequest(ctx, item)
			if isNotif || rb == nil {
				continue
			}
			var v interface{}
			if err := json.Unmarshal(rb, &v); err == nil {
				out = append(out, v)
			}
		}
		if len(out) == 0 {
			return nil, true, nil
		}
		b, err := json.Marshal(out)
		if err != nil {
			return nil, false, err
		}
		return b, false, nil
	}

	var req Request
	if err := json.Unmarshal(raw, &req); err != nil {
		resp := Response{JSONRPC: "2.0", Error: &RPCError{Code: -32700, Message: "Parse error"}}
		b, _ := json.Marshal(resp)
		return b, false, nil
	}
	if req.JSONRPC == "" {
		req.JSONRPC = "2.0"
	}

	if strings.HasPrefix(req.Method, "notifications/") {
		if len(req.ID) == 0 || string(req.ID) == "null" {
			return nil, true, nil
		}
		resp := Response{JSONRPC: "2.0", ID: req.ID, Result: map[string]interface{}{}}
		b, _ := json.Marshal(resp)
		return b, false, nil
	}

	result, rpcErr := s.dispatch(ctx, req)
	resp := Response{JSONRPC: "2.0", ID: req.ID}
	if rpcErr != nil {
		resp.Error = rpcErr
	} else {
		resp.Result = result
	}
	b, err := json.Marshal(resp)
	if err != nil {
		return nil, false, err
	}
	return b, false, nil
}

func (s *Server) dispatch(ctx context.Context, req Request) (interface{}, *RPCError) {
	switch req.Method {
	case "initialize":
		proto := "2025-03-26"
		var params struct {
			ProtocolVersion string `json:"protocolVersion"`
		}
		_ = json.Unmarshal(req.Params, &params)
		if params.ProtocolVersion == "2024-11-05" || params.ProtocolVersion == "2025-03-26" {
			proto = params.ProtocolVersion
		}
		return map[string]interface{}{
			"protocolVersion": proto,
			"capabilities": map[string]interface{}{
				"tools":     map[string]interface{}{},
				"resources": map[string]interface{}{},
				"prompts":   map[string]interface{}{},
			},
			"serverInfo": map[string]interface{}{
				"name":    "paperfit",
				"version": "1.0.0",
			},
		}, nil
	case "ping":
		return map[string]interface{}{}, nil
	case "tools/list":
		return map[string]interface{}{"tools": toolDefs()}, nil
	case "tools/call":
		var params struct {
			Name      string                 `json:"name"`
			Arguments map[string]interface{} `json:"arguments"`
		}
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return nil, &RPCError{Code: -32602, Message: "Invalid params"}
		}
		res, err := s.callTool(ctx, params.Name, params.Arguments)
		if err != nil {
			return map[string]interface{}{
				"content": []map[string]interface{}{{"type": "text", "text": err.Error()}},
				"isError": true,
			}, nil
		}
		b, _ := json.Marshal(res)
		return map[string]interface{}{
			"content":           []map[string]interface{}{{"type": "text", "text": string(b)}},
			"structuredContent": res,
		}, nil
	case "resources/list":
		return map[string]interface{}{"resources": resourceDefs()}, nil
	case "resources/read":
		var params struct {
			URI string `json:"uri"`
		}
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return nil, &RPCError{Code: -32602, Message: "Invalid params"}
		}
		content, err := readResource(params.URI)
		if err != nil {
			return nil, &RPCError{Code: -32602, Message: err.Error()}
		}
		return map[string]interface{}{
			"contents": []map[string]interface{}{
				{"uri": params.URI, "mimeType": "text/markdown", "text": content},
			},
		}, nil
	case "prompts/list":
		return map[string]interface{}{"prompts": promptDefs()}, nil
	case "prompts/get":
		var params struct {
			Name string `json:"name"`
		}
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return nil, &RPCError{Code: -32602, Message: "Invalid params"}
		}
		msgs, err := getPrompt(params.Name)
		if err != nil {
			return nil, &RPCError{Code: -32602, Message: err.Error()}
		}
		return map[string]interface{}{"messages": msgs}, nil
	default:
		return nil, &RPCError{Code: -32601, Message: "Method not found"}
	}
}

func (s *Server) callTool(ctx context.Context, name string, args map[string]interface{}) (interface{}, error) {
	switch name {
	case "render_pdf_from_html":
		return s.toolRenderHTML(ctx, args)
	case "render_pdf_from_archive":
		return s.toolRenderArchive(ctx, args)
	case "render_pdf_from_archive_url":
		return s.toolRenderArchiveURL(ctx, args)
	case "validate_template_variables":
		return s.toolValidateVars(args)
	case "inspect_template_archive":
		return s.toolInspectArchive(ctx, args)
	case "scaffold_paperfit_template":
		return s.toolScaffold(args)
	default:
		return nil, fmt.Errorf("unknown tool: %s", name)
	}
}

func getString(args map[string]interface{}, key string) string {
	if v, ok := args[key]; ok && v != nil {
		switch t := v.(type) {
		case string:
			return t
		default:
			b, _ := json.Marshal(t)
			return string(b)
		}
	}
	return ""
}

func getBool(args map[string]interface{}, key string) bool {
	if v, ok := args[key]; ok && v != nil {
		switch t := v.(type) {
		case bool:
			return t
		}
	}
	return false
}

func getMap(args map[string]interface{}, key string) map[string]interface{} {
	if v, ok := args[key]; ok && v != nil {
		if m, ok := v.(map[string]interface{}); ok {
			return m
		}
	}
	return nil
}

func getMapString(args map[string]interface{}, key string) map[string]string {
	m := getMap(args, key)
	if m == nil {
		return nil
	}
	out := map[string]string{}
	for k, v := range m {
		out[k] = fmt.Sprintf("%v", v)
	}
	return out
}

func (s *Server) toolRenderHTML(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	htmlStr := getString(args, "html")
	if htmlStr == "" {
		return nil, fmt.Errorf("html is required")
	}
	if outPath := getString(args, "output_path"); outPath != "" {
		if !s.allowLocalFS {
			return nil, fmt.Errorf("output_path is not allowed over HTTP MCP; use pdf_base64 from the response")
		}
		for _, seg := range strings.Split(outPath, "/") {
			if seg == ".." {
				return nil, fmt.Errorf("output_path contains traversal component")
			}
		}
	}
	vars := getMap(args, "variables")
	assets := getMapString(args, "assets")
	var schema interface{}
	if v, ok := args["json_schema"]; ok {
		schema = v
	}
	validate := getBool(args, "validate")

	// validate assets
	for name := range assets {
		if name == "index.html" || name == "json_schema.json" || strings.Contains(name, "..") {
			return nil, fmt.Errorf("invalid asset name: %s", name)
		}
	}

	project := service.NewProject([]*service.File{})
	_ = project.AppendFile("index.html", bytes.NewReader([]byte(htmlStr)), int64(len(htmlStr)))
	for name, content := range assets {
		_ = project.AppendFile(name, bytes.NewReader([]byte(content)), int64(len(content)))
	}
	if schema != nil {
		var schemaBytes []byte
		var err error
		switch v := schema.(type) {
		case string:
			schemaBytes = []byte(v)
		default:
			schemaBytes, err = json.Marshal(v)
			if err != nil {
				return nil, err
			}
		}
		_ = project.AppendFile("json_schema.json", bytes.NewReader(schemaBytes), int64(len(schemaBytes)))
	}

	// options
	if o, ok := args["options"].(map[string]interface{}); ok {
		project.RenderOptions = mapToRenderOptions(o)
	}

	var varsStr string
	if len(vars) > 0 {
		b, _ := json.Marshal(vars)
		varsStr = string(b)
	}

	if varsStr != "" {
		if err := project.AppendVariables(varsStr); err != nil {
			return nil, err
		}
	}
	if validate {
		js := project.GetJSONschemaFile()
		if js == nil || js.Size() == 0 {
			return nil, fmt.Errorf("no json schema found")
		}
		if _, err := validation.Validate(varsStr, js.Content); err != nil {
			return nil, err
		}
	}

	res, size, err := s.pdfService.Generate(ctx, &project)
	if err != nil {
		return nil, err
	}
	pdfBytes, _ := io.ReadAll(res)
	outPath := getString(args, "output_path")
	if outPath != "" {
		if !s.allowLocalFS {
			return nil, fmt.Errorf("output_path is not allowed over HTTP MCP; use pdf_base64 from the response")
		}
		if strings.Contains(outPath, "..") {
			// traversal check components
			bad := false
			for _, seg := range strings.Split(outPath, "/") {
				if seg == ".." {
					bad = true
				}
			}
			if bad {
				return nil, fmt.Errorf("output_path contains traversal component")
			}
		}
		if err := os.WriteFile(outPath, pdfBytes, 0o644); err != nil {
			return nil, err
		}
	}
	return map[string]interface{}{
		"size":        size,
		"pdf_base64":  base64.StdEncoding.EncodeToString(pdfBytes),
		"output_path": outPath,
		"metadata":    map[string]interface{}{"files": len(project.GetFiles())},
	}, nil
}

func mapToRenderOptions(m map[string]interface{}) *service.RenderOptions {
	o := &service.RenderOptions{}
	if v, ok := m["landscape"].(bool); ok {
		o.Landscape = &v
	}
	if v, ok := m["waitDelay"].(string); ok {
		o.WaitDelay = &v
	}
	if v, ok := m["waitForExpression"].(string); ok {
		o.WaitForExpression = &v
	}
	if v, ok := m["paperWidth"].(string); ok {
		o.PaperWidth = &v
	}
	if v, ok := m["paperHeight"].(string); ok {
		o.PaperHeight = &v
	}
	if v, ok := m["marginTop"].(string); ok {
		o.MarginTop = &v
	}
	if v, ok := m["marginBottom"].(string); ok {
		o.MarginBottom = &v
	}
	if v, ok := m["marginLeft"].(string); ok {
		o.MarginLeft = &v
	}
	if v, ok := m["marginRight"].(string); ok {
		o.MarginRight = &v
	}
	if v, ok := m["preferCssPageSize"].(bool); ok {
		o.PreferCssPageSize = &v
	}
	if v, ok := m["printBackground"].(bool); ok {
		o.PrintBackground = &v
	}
	return o
}

func (s *Server) toolRenderArchive(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	var project service.Project
	var err error
	_ = err
	if outPath := getString(args, "output_path"); outPath != "" && !s.allowLocalFS {
		return nil, fmt.Errorf("output_path is not allowed over HTTP MCP; use pdf_base64 from the response")
	}
	if outPath := getString(args, "output_path"); outPath != "" {
		for _, seg := range strings.Split(outPath, "/") {
			if seg == ".." {
				return nil, fmt.Errorf("output_path contains traversal component")
			}
		}
	}
	if p := getString(args, "archive_path"); p != "" {
		if !s.allowLocalFS {
			return nil, fmt.Errorf("archive_path is not allowed over HTTP MCP; use archive_base64 or url instead")
		}
		for _, seg := range strings.Split(p, "/") {
			if seg == ".." {
				return nil, fmt.Errorf("archive_path contains traversal component")
			}
		}
		f, err := os.Open(p)
		if err != nil {
			return nil, err
		}
		defer func() { _ = f.Close() }()
		project, err = archiveExtract(ctx, p, f)
		if err != nil {
			return nil, err
		}
	} else if b64 := getString(args, "archive_base64"); b64 != "" {
		data, err := base64.StdEncoding.DecodeString(b64)
		if err != nil {
			return nil, err
		}
		filename := getString(args, "filename")
		if filename == "" {
			filename = "archive.tar.gz"
		}
		project, err = archiveExtractBytes(ctx, filename, data)
		if err != nil {
			return nil, err
		}
	} else {
		return nil, fmt.Errorf("archive_path or archive_base64 required")
	}

	vars := getMap(args, "variables")
	validate := getBool(args, "validate")

	var varsStr string
	if len(vars) > 0 {
		b, _ := json.Marshal(vars)
		varsStr = string(b)
	}
	if varsStr != "" {
		if err := project.AppendVariables(varsStr); err != nil {
			return nil, err
		}
	}
	if validate {
		js := project.GetJSONschemaFile()
		if js == nil || js.Size() == 0 {
			return nil, fmt.Errorf("no json schema found")
		}
		if _, err := validation.Validate(varsStr, js.Content); err != nil {
			return nil, err
		}
	}

	res, size, err := s.pdfService.Generate(ctx, &project)
	if err != nil {
		return nil, err
	}
	pdfBytes, _ := io.ReadAll(res)
	outPath := getString(args, "output_path")
	if outPath != "" {
		if err := os.WriteFile(outPath, pdfBytes, 0o644); err != nil {
			return nil, err
		}
	}
	return map[string]interface{}{
		"size":        size,
		"pdf_base64":  base64.StdEncoding.EncodeToString(pdfBytes),
		"output_path": outPath,
	}, nil
}

// helper to avoid import cycle: use archive package
func archiveExtract(ctx context.Context, filename string, r interface{ Read([]byte) (int, error) }) (service.Project, error) {
	// We need io.Reader; cast via io.Reader
	ir := r.(io.Reader)
	return extractToProject(ctx, filename, ir)
}

func archiveExtractBytes(ctx context.Context, filename string, data []byte) (service.Project, error) {
	return extractToProject(ctx, filename, bytes.NewReader(data))
}

func extractToProject(ctx context.Context, filename string, r io.Reader) (service.Project, error) {
	return archive.ExtractToProject(ctx, filename, r)
}

// We'll define actual call in a separate file with import.
func (s *Server) toolRenderArchiveURL(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	urlStr := getString(args, "url")
	if urlStr == "" {
		return nil, fmt.Errorf("url is required")
	}
	u, err := url.Parse(urlStr)
	if err != nil {
		return nil, err
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, fmt.Errorf("invalid url scheme")
	}
	headers := getMapString(args, "additional_headers")

	if outPath := getString(args, "output_path"); outPath != "" && !s.allowLocalFS {
		return nil, fmt.Errorf("output_path is not allowed over HTTP MCP; use pdf_base64 from the response")
	}
	if outPath := getString(args, "output_path"); outPath != "" {
		for _, seg := range strings.Split(outPath, "/") {
			if seg == ".." {
				return nil, fmt.Errorf("output_path contains traversal component")
			}
		}
	}

	req, err := http.NewRequestWithContext(ctx, "GET", urlStr, nil)
	if err != nil {
		return nil, err
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	client := &http.Client{Timeout: 60 * time.Second}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = res.Body.Close() }()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("failed to fetch archive: %s", res.Status)
	}
	data, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	filename := "archive.tar.gz"
	project, err := extractToProject(ctx, filename, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	vars := getMap(args, "variables")
	validate := getBool(args, "validate")
	var varsStr string
	if len(vars) > 0 {
		b, _ := json.Marshal(vars)
		varsStr = string(b)
	}
	if varsStr != "" {
		if err := project.AppendVariables(varsStr); err != nil {
			return nil, err
		}
	}
	if validate {
		js := project.GetJSONschemaFile()
		if js == nil || js.Size() == 0 {
			return nil, fmt.Errorf("no json schema found")
		}
		if _, err := validation.Validate(varsStr, js.Content); err != nil {
			return nil, err
		}
	}

	res2, size, err := s.pdfService.Generate(ctx, &project)
	if err != nil {
		return nil, err
	}
	pdfBytes, _ := io.ReadAll(res2)
	outPath := getString(args, "output_path")
	if outPath != "" {
		if err := os.WriteFile(outPath, pdfBytes, 0o644); err != nil {
			return nil, err
		}
	}
	return map[string]interface{}{
		"size":        size,
		"pdf_base64":  base64.StdEncoding.EncodeToString(pdfBytes),
		"output_path": outPath,
	}, nil
}

func (s *Server) toolValidateVars(args map[string]interface{}) (interface{}, error) {
	vars := getMap(args, "variables")
	if vars == nil {
		return nil, fmt.Errorf("variables required")
	}
	schema := args["json_schema"]
	if schema == nil {
		return nil, fmt.Errorf("json_schema required")
	}
	var schemaBytes []byte
	var err error
	switch v := schema.(type) {
	case string:
		schemaBytes = []byte(v)
	default:
		schemaBytes, err = json.Marshal(v)
		if err != nil {
			return nil, err
		}
	}
	varsBytes, _ := json.Marshal(vars)
	_, err = validation.Validate(string(varsBytes), bytes.NewReader(schemaBytes))
	if err != nil {
		return map[string]interface{}{"valid": false, "errors": err.Error()}, nil
	}
	return map[string]interface{}{"valid": true}, nil
}

func (s *Server) toolInspectArchive(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	var project service.Project
	var err error
	_ = err
	if outPath := getString(args, "output_path"); outPath != "" && !s.allowLocalFS {
		return nil, fmt.Errorf("output_path is not allowed over HTTP MCP; use pdf_base64 from the response")
	}
	if outPath := getString(args, "output_path"); outPath != "" {
		for _, seg := range strings.Split(outPath, "/") {
			if seg == ".." {
				return nil, fmt.Errorf("output_path contains traversal component")
			}
		}
	}
	if p := getString(args, "archive_path"); p != "" {
		if !s.allowLocalFS {
			return nil, fmt.Errorf("archive_path is not allowed over HTTP MCP; use archive_base64 or url instead")
		}
		for _, seg := range strings.Split(p, "/") {
			if seg == ".." {
				return nil, fmt.Errorf("archive_path contains traversal component")
			}
		}
		f, err := os.Open(p)
		if err != nil {
			return nil, err
		}
		defer func() { _ = f.Close() }()
		project, err = extractToProject(ctx, p, f)
		if err != nil {
			return nil, err
		}
	} else if b64 := getString(args, "archive_base64"); b64 != "" {
		data, err := base64.StdEncoding.DecodeString(b64)
		if err != nil {
			return nil, err
		}
		project, err = extractToProject(ctx, "archive.tar.gz", bytes.NewReader(data))
		if err != nil {
			return nil, err
		}
	} else if u := getString(args, "url"); u != "" {
		pu, err := url.Parse(u)
		if err != nil {
			return nil, err
		}
		if pu.Scheme != "http" && pu.Scheme != "https" {
			return nil, fmt.Errorf("invalid url scheme")
		}
		req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
		if err != nil {
			return nil, err
		}
		client := &http.Client{Timeout: 60 * time.Second}
		res, err := client.Do(req)
		if err != nil {
			return nil, err
		}
		defer func() { _ = res.Body.Close() }()
		data, _ := io.ReadAll(res.Body)
		project, err = extractToProject(ctx, "archive.tar.gz", bytes.NewReader(data))
		if err != nil {
			return nil, err
		}
	} else {
		return nil, fmt.Errorf("archive_path or url required")
	}

	files := []string{}
	for _, f := range project.GetFiles() {
		files = append(files, f.Name())
	}
	hasIndex := project.GetIndexFile() != nil
	js := project.GetJSONschemaFile()
	hasSchema := js != nil
	var schema interface{}
	if hasSchema {
		_ = json.Unmarshal(readAll(js.Content), &schema)
	}
	// detect paperfit.js / window.__PAPERFIT_INPUT__
	usesInput := false
	usesPaperfit := false
	if idx := project.GetIndexFile(); idx != nil {
		content := string(readAll(idx.Content))
		if strings.Contains(content, "window.__PAPERFIT_INPUT__") {
			usesInput = true
		}
		if strings.Contains(content, "paperfit.js") {
			usesPaperfit = true
		}
	}
	return map[string]interface{}{
		"files":               files,
		"has_index_html":      hasIndex,
		"has_json_schema":     hasSchema,
		"json_schema":         schema,
		"uses_paperfit_input": usesInput,
		"uses_paperfit_js":    usesPaperfit,
	}, nil
}

func readAll(r io.Reader) []byte {
	b, _ := io.ReadAll(r)
	switch v := r.(type) {
	case *bytes.Reader:
		v.Reset(b)
	case *bytes.Buffer:
		v.Reset()
		v.Write(b)
	}
	return b
}

func (s *Server) toolScaffold(args map[string]interface{}) (interface{}, error) {
	docType := getString(args, "document_type")
	if docType == "" {
		docType = "custom"
	}
	framework := getString(args, "framework")
	if framework == "" {
		framework = "vanilla"
	}
	pageSize := getString(args, "page_size")
	if pageSize == "" {
		pageSize = "A4"
	}
	orientation := getString(args, "orientation")
	if orientation == "" {
		orientation = "portrait"
	}

	title := cases.Title(language.English).String(docType)
	pageRule := "@page { size: " + pageSize + " " + orientation + "; margin: 20mm; }"

	// Framework-specific bootstrap snippet using paperfit.js APIs.
	var bootstrap string
	switch framework {
	case "react":
		bootstrap = `import { configurePage, renderAfterFrameworkMount } from 'paperfit.js';
configurePage({ size: '` + pageSize + `', margin: '20mm', pageNumbers: 'bottom-right' });
renderAfterFrameworkMount(undefined, { source: '#root' });`
	case "vue":
		bootstrap = `import { configurePage, renderAfterFrameworkMount } from 'paperfit.js';
configurePage({ size: '` + pageSize + `', margin: '20mm', pageNumbers: 'bottom-right' });
renderAfterFrameworkMount(undefined, { source: '#app' });`
	case "angular":
		bootstrap = `import { configurePage, renderAfterFrameworkMount } from 'paperfit.js';
configurePage({ size: '` + pageSize + `', margin: '20mm', pageNumbers: 'bottom-right' });
renderAfterFrameworkMount(undefined, { source: 'app-root' });`
	default:
		bootstrap = `import { configurePage, getPaperfitInput, bindPaperfitVariables, renderPaperfit } from 'paperfit.js';
configurePage({ size: '` + pageSize + `', margin: '20mm', pageNumbers: 'bottom-right' });
const input = getPaperfitInput();
bindPaperfitVariables(document, input);
renderPaperfit({ source: '#app' });`
	}

	// Document-type-specific body content.
	var bodyContent string
	switch docType {
	case "invoice":
		bodyContent = `<h1>Invoice</h1>
  <table>
    <thead>
      <tr><th>Item</th><th>Qty</th><th>Price</th></tr>
    </thead>
    <tbody data-paperfit-each="items">
      <template>
        <tr><td data-paperfit-text="name"></td><td data-paperfit-text="qty"></td><td data-paperfit-text="price"></td></tr>
      </template>
    </tbody>
  </table>`
	case "report":
		bodyContent = `<h1>Report</h1>
  <section data-paperfit-if="summary">
    <h2>Summary</h2>
    <p data-paperfit-text="summary"></p>
  </section>`
	case "contract":
		bodyContent = `<h1>Contract</h1>
  <p>Between <span data-paperfit-text="partyA"></span> and <span data-paperfit-text="partyB"></span>.</p>`
	case "letter":
		bodyContent = `<h1>Letter</h1>
  <p data-paperfit-text="body"></p>`
	default:
		bodyContent = `<h1>` + title + `</h1>
  <p data-paperfit-text="title"></p>`
	}

	html := `<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8" />
  <title>` + docType + `</title>
  <script src="https://unpkg.com/pagedjs/dist/paged.polyfill.js"></script>
  <script type="module">
    ` + bootstrap + `
    window.__PAPERFIT_READY__ = true;
  </script>
  <style>
    ` + pageRule + `
    body { font-family: sans-serif; }
    .header { position: running(header); }
    .footer { position: running(footer); }
    @page {
      @top-center { content: element(header); }
      @bottom-center { content: element(footer); }
    }
  </style>
</head>
<body>
  <div class="header">` + docType + `</div>
  <main id="app">
    ` + bodyContent + `
  </main>
  <div class="footer">Page <paperfit-page-number></paperfit-page-number></div>
  <script>
    const input = window.__PAPERFIT_INPUT__ || {};
    window.__PAPERFIT_READY__ = true;
  </script>
</body>
</html>`

	schema := `{
  "type": "object",
  "properties": {
    "title": { "type": "string" }
  },
  "required": ["title"]
}`
	vars := `{"title": "Sample"}`

	return map[string]interface{}{
		"files": map[string]string{
			"index.html":       html,
			"json_schema.json": schema,
		},
		"sample_variables": vars,
		"framework":        framework,
		"document_type":    docType,
		"page_size":        pageSize,
		"orientation":      orientation,
	}, nil
}

func toolDefs() []map[string]interface{} {
	return []map[string]interface{}{
		{"name": "render_pdf_from_html", "description": "Render PDF from HTML", "inputSchema": map[string]interface{}{"type": "object", "properties": map[string]interface{}{"html": map[string]interface{}{"type": "string"}, "variables": map[string]interface{}{"type": "object"}, "assets": map[string]interface{}{"type": "object"}, "json_schema": map[string]interface{}{}, "validate": map[string]interface{}{"type": "boolean"}, "options": map[string]interface{}{"type": "object"}, "output_path": map[string]interface{}{"type": "string"}}, "required": []string{"html"}}},
		{"name": "render_pdf_from_archive", "description": "Render PDF from archive", "inputSchema": map[string]interface{}{"type": "object", "properties": map[string]interface{}{"archive_path": map[string]interface{}{"type": "string"}, "archive_base64": map[string]interface{}{"type": "string"}, "filename": map[string]interface{}{"type": "string"}, "variables": map[string]interface{}{"type": "object"}, "validate": map[string]interface{}{"type": "boolean"}, "output_path": map[string]interface{}{"type": "string"}}}},
		{"name": "render_pdf_from_archive_url", "description": "Render PDF from archive URL", "inputSchema": map[string]interface{}{"type": "object", "properties": map[string]interface{}{"url": map[string]interface{}{"type": "string"}, "variables": map[string]interface{}{"type": "object"}, "validate": map[string]interface{}{"type": "boolean"}, "additional_headers": map[string]interface{}{"type": "object"}, "output_path": map[string]interface{}{"type": "string"}}, "required": []string{"url"}}},
		{"name": "validate_template_variables", "description": "Validate variables against schema", "inputSchema": map[string]interface{}{"type": "object", "properties": map[string]interface{}{"variables": map[string]interface{}{"type": "object"}, "json_schema": map[string]interface{}{}}, "required": []string{"variables", "json_schema"}}},
		{"name": "inspect_template_archive", "description": "Inspect template archive", "inputSchema": map[string]interface{}{"type": "object", "properties": map[string]interface{}{"archive_path": map[string]interface{}{"type": "string"}, "url": map[string]interface{}{"type": "string"}}}},
		{"name": "scaffold_paperfit_template", "description": "Scaffold paperfit template", "inputSchema": map[string]interface{}{"type": "object", "properties": map[string]interface{}{"document_type": map[string]interface{}{"type": "string"}, "framework": map[string]interface{}{"type": "string"}, "page_size": map[string]interface{}{"type": "string"}, "orientation": map[string]interface{}{"type": "string"}}}},
	}
}

func resourceDefs() []map[string]interface{} {
	return []map[string]interface{}{
		{"uri": "paperfit://docs/authoring-guide", "name": "authoring-guide", "mimeType": "text/markdown"},
		{"uri": "paperfit://docs/openapi", "name": "openapi", "mimeType": "text/markdown"},
		{"uri": "paperfit://examples/invoice", "name": "invoice", "mimeType": "text/markdown"},
	}
}

func readResource(uri string) (string, error) {
	switch uri {
	case "paperfit://docs/authoring-guide":
		return authoringGuide, nil
	case "paperfit://docs/openapi":
		return "OpenAPI specification available at /openapi.json on the running Paperfit server.", nil
	case "paperfit://examples/invoice":
		return invoiceExample, nil
	default:
		return "", fmt.Errorf("unknown resource: %s", uri)
	}
}

func promptDefs() []map[string]interface{} {
	return []map[string]interface{}{
		{"name": "create-pdf-template", "description": "Guide to create a PDF template"},
		{"name": "generate-pdf-from-data", "description": "Guide to generate PDF from data"},
	}
}

func getPrompt(name string) ([]map[string]interface{}, error) {
	switch name {
	case "create-pdf-template":
		return []map[string]interface{}{
			{"role": "user", "content": map[string]interface{}{"type": "text", "text": "Design a complete Paperfit HTML template + JSON Schema for the requested document."}},
		}, nil
	case "generate-pdf-from-data":
		return []map[string]interface{}{
			{"role": "user", "content": map[string]interface{}{"type": "text", "text": "Inspect the template, validate data, and render the PDF."}},
		}, nil
	default:
		return nil, fmt.Errorf("unknown prompt: %s", name)
	}
}

var authoringGuide = `# Paperfit Authoring Guide

Paperfit is an HTML-to-PDF engine with first-class support for paperfit.js and Paged.js templates.

## REST API endpoints

- POST /pdf/from/html - Render a PDF directly from an HTML string.
- POST /pdf/from/archive/file - Render from an uploaded archive.
- POST /pdf/from/archive/url - Render from an archive URL.
- POST /pdf/from/html/file - Render from an uploaded HTML file (where available).
- GET /application/healthy - Health check.
- POST /mcp - Model Context Protocol endpoint.
- GET /llms.txt - LLM-friendly specification.

## Variable injection

paperfit injects window.__PAPERFIT_INPUT__ as the first script in <head>.
It waits for window.__PAPERFIT_READY__ !== false in Gotenberg before rendering.

Use AppendVariables to inject variables into index.html.

## paperfit.js APIs

- getPaperfitInput
- setPaperfitDefaultInput
- bindPaperfitVariables
- configurePage
- renderPaperfit
- renderAfterFrameworkMount
- createPaperfitTemplate
- <paperfit-document>
- <paperfit-header>
- <paperfit-footer>
- <paperfit-page-break>
- <paperfit-avoid-break>
- <paperfit-page-number>
- paperfit-build CLI
- paperfitVitePlugin

## MCP tools

- render_pdf_from_html
- render_pdf_from_archive
- render_pdf_from_archive_url
- validate_template_variables
- inspect_template_archive
- scaffold_paperfit_template

## Resources

- paperfit://docs/authoring-guide
- paperfit://docs/openapi
- paperfit://examples/invoice
`

var invoiceExample = `# Paperfit Invoice Example

A complete, realistic multi-page HTML invoice template.

## index.html

<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8" />
  <title>Invoice</title>
  <script src="https://unpkg.com/pagedjs/dist/paged.polyfill.js"></script>
  <script src="paperfit.js"></script>
  <style>
    @page { size: A4 portrait; margin: 20mm; }
    body { font-family: sans-serif; }
    .header { position: running(header); }
    .footer { position: running(footer); }
    @page {
      @top-center { content: element(header); }
      @bottom-center { content: element(footer); }
    }
    table { width: 100%; border-collapse: collapse; }
    thead th { border-bottom: 1px solid #000; text-align: left; }
  </style>
</head>
<body>
  <div class="header">Invoice #<span data-paperfit-text="invoiceNumber"></span></div>
  <h1>Invoice</h1>
  <p>Bill to: <span data-paperfit-text="customer.name"></span></p>
  <table>
    <thead>
      <tr><th>Description</th><th>Qty</th><th>Unit Price</th><th>Total</th></tr>
    </thead>
    <tbody data-paperfit-each="items">
      <template>
        <tr>
          <td data-paperfit-text="description"></td>
          <td data-paperfit-text="qty"></td>
          <td data-paperfit-text="unitPrice"></td>
          <td data-paperfit-text="total"></td>
        </tr>
      </template>
    </tbody>
  </table>
  <div class="footer">Page <span class="pageNumber"></span> of <span class="totalPages"></span></div>
  <script>
    const input = window.__PAPERFIT_INPUT__ || {};
    window.__PAPERFIT_READY__ = true;
  </script>
</body>
</html>

## json_schema.json

{
  "type": "object",
  "properties": {
    "invoiceNumber": { "type": "string" },
    "customer": {
      "type": "object",
      "properties": { "name": { "type": "string" } },
      "required": ["name"]
    },
    "items": {
      "type": "array",
      "items": {
        "type": "object",
        "properties": {
          "description": { "type": "string" },
          "qty": { "type": "number" },
          "unitPrice": { "type": "number" },
          "total": { "type": "number" }
        }
      }
    }
  },
  "required": ["invoiceNumber", "customer", "items"]
}
`

// Stdio transport helper
func ServeStdio(ctx context.Context, s *Server) {
	reader := bufio.NewReader(os.Stdin)
	writer := os.Stdout
	for {
		line, err := reader.ReadString('\n')
		if err != nil && line == "" {
			break
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		respBytes, isNotif, _ := s.HandleRequest(ctx, []byte(line))
		if isNotif || respBytes == nil {
			continue
		}
		_, _ = writer.Write(respBytes)
		_, _ = writer.Write([]byte("\n"))
	}
}
