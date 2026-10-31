package server

import (
	"net/http"

	"github.com/labstack/echo/v5"
)

func (s *Server) HandleLLMs(c *echo.Context) error {
	return c.String(http.StatusOK, llmsText)
}

var llmsText = `# Paperfit

Paperfit is a fast HTML-to-PDF engine with first-class support for paperfit.js and Paged.js templates.

## Endpoints

- POST /pdf/from/html - Render a PDF directly from an HTML string.
- POST /pdf/from/archive/file - Render from an uploaded archive.
- POST /pdf/from/archive/url - Render from an archive URL.
- POST /pdf/from/html/file - Render from an uploaded HTML file (where available).
- GET /application/healthy - Health check.
- POST /mcp - Model Context Protocol endpoint.
- GET /llms.txt - LLM-friendly specification.

## Direct HTML rendering

POST /pdf/from/html accepts a JSON body:

{
  "html": "<html>...</html>",
  "assets": { "style.css": "body{}" },
  "jsonSchema": { "type": "object" },
  "input": { "validate": true, "variables": { "title": "Hello" } },
  "options": { "landscape": false, "waitForExpression": "window.__PAPERFIT_READY__ !== false" }
}

Variables are injected as window.__PAPERFIT_INPUT__.

## paperfit.js / Paged.js

Templates can use Paged.js for multi-page layout and paperfit.js for running headers, footers, and page numbers.
Gotenberg waits for window.__PAPERFIT_READY__ !== false before rendering.

paperfit injects window.__PAPERFIT_INPUT__ as the first script in <head>.

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

## MCP server

Run the stdio MCP server:

go run ./cmd/mcp

Or use the HTTP endpoint at POST /mcp.

Available tools: render_pdf_from_html, render_pdf_from_archive, render_pdf_from_archive_url,
validate_template_variables, inspect_template_archive, scaffold_paperfit_template.
`
