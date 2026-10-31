# paperfit

**HTML in. Beautiful PDFs out.**

paperfit turns versioned HTML templates into pixel-perfect, print-ready PDFs at scale.
Bring any frontend stack, build with paperfit.js and Paged.js, and let any AI model drive it through MCP.

[![CI](https://github.com/dokufy/paperfit/actions/workflows/ci.yml/badge.svg)](https://github.com/dokufy/paperfit/actions/workflows/ci.yml)
[![Container](https://img.shields.io/badge/ghcr.io-dokufy%2Fpaperfit-blue)](https://github.com/dokufy/paperfit/pkgs/container/paperfit)

[Quick start](#quick-start) · [Features](#features) · [Build a template](#build-a-template-with-paperfitjs) · [AI ready](#ai-ready-with-any-model) · [Docker](#docker) · [API](#api) · [Development](#development)

## Features

| Feature | What you get |
| --- | --- |
| **Any template stack** | React, Vue, Svelte, Astro, plain HTML, even `<canvas>`. |
| **Print-perfect layout** | paperfit.js and Paged.js give you running headers, footers, page numbers, and page breaks. |
| **AI ready** | Built-in MCP server and `/llms.txt`. Any model or agent can render PDFs, validate variables, and scaffold templates. |
| **Model agnostic** | No provider lock-in. Use OpenAI, Anthropic, Gemini, a local model through Ollama or vLLM, or your own agent. |
| **Validated variables** | JSON Schema validation before rendering. Variables are injected as `window.__PAPERFIT_INPUT__`. |
| **Built for speed** | Performance target of 20,000 pages in under 60 seconds, with optional PDF caching. |
| **Cache anywhere** | AWS S3, GCS, Minio, Dropbox, and more, or the local file system. |
| **Versioned templates** | Publish templates as archives and request a specific URL or version. |
| **Small container** | Around 38 MB, non-root, and configured entirely with environment variables. |

## How it works

<img src="docs/paperfit-concept.jpg" width="800" alt="paperfit flow: an app requests a PDF, paperfit loads an HTML template or versioned archive, renders it with Chromium, and returns the PDF" />

A typical paperfit instance runs alongside your backend application.

HTML templates can be built with any modern frontend stack.
You can send the HTML directly to paperfit.
Publishing the template archive to a package registry such as GitLab or GitHub is optional, but it is a convenient way to keep versioned documents.
To create a PDF, your application sends the raw HTML or the template archive URL together with the variables.
paperfit merges the variables into the template, renders it with Gotenberg and Chromium, optionally caches the PDF, and streams it back.

## Quick start

Run paperfit next to Gotenberg with Docker Compose:

```yaml
services:
  paperfit:
    image: ghcr.io/dokufy/paperfit:latest
    ports:
      - "4000:4000"
    environment:
      PDF_GENERATOR_ENDPOINT: http://gotenberg:3000/forms/chromium/convert/html
  gotenberg:
    image: gotenberg/gotenberg:8.39.0-chromium
    command: gotenberg --api-timeout=300s
```

```sh
docker compose up -d
curl http://localhost:4000/application/healthy
```

You should see `{"status":"ok"}`.
Now you are ready to render your first template.

## Build a template with paperfit.js

Use case examples for Vanilla JavaScript, Vue, React, and Angular are available in the [pdf_examples repository](https://github.com/dokufy/pdf_examples).

A template is just an HTML file, an optional `json_schema.json`, and any assets you need.
Here is a small vanilla JavaScript invoice:

```html
<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8" />
  <script src="https://unpkg.com/pagedjs/dist/paged.polyfill.js"></script>
  <script src="paperfit.js"></script>
  <script>window.__PAPERFIT_READY__ = false;</script>
  <style>
    body { font-family: system-ui, sans-serif; color: #111; }
    table { width: 100%; border-collapse: collapse; margin-top: 24px; }
    th, td { padding: 8px 0; border-bottom: 1px solid #ddd; text-align: left; }
    th:last-child, td:last-child { text-align: right; }
  </style>
</head>
<body>
  <article id="invoice">
    <h1>Invoice #<span data-paperfit-text="number"></span></h1>
    <p>Bill to <strong data-paperfit-text="customer"></strong></p>

    <table>
      <thead>
        <tr><th>Item</th><th>Qty</th><th>Price</th></tr>
      </thead>
      <tbody data-paperfit-each="items">
        <template>
          <tr>
            <td data-paperfit-text="name"></td>
            <td data-paperfit-text="qty"></td>
            <td data-paperfit-text="price"></td>
          </tr>
        </template>
      </tbody>
    </table>
  </article>

  <script>
    (async () => {
      configurePage({ size: "A4", margin: "20mm", pageNumbers: "bottom-right" });
      await bindPaperfitVariables(document, getPaperfitInput());
      await renderPaperfit({ source: "#invoice" });
      window.__PAPERFIT_READY__ = true;
    })();
  </script>
</body>
</html>
```

Add `json_schema.json` next to it to validate the input:

```json
{
  "type": "object",
  "required": ["number", "customer", "items"],
  "properties": {
    "number": { "type": "string" },
    "customer": { "type": "string" },
    "items": {
      "type": "array",
      "items": {
        "type": "object",
        "required": ["name", "qty", "price"],
        "properties": {
          "name": { "type": "string" },
          "qty": { "type": "number" },
          "price": { "type": "string" }
        }
      }
    }
  }
}
```

Make sure `paperfit.js` is available in the template directory, or let `paperfit-build` or `paperfitVitePlugin` bundle it for you.
If you use a bundler, import the same functions from `paperfit.js` instead of relying on globals.

Package the template and render it:

```sh
tar -czf invoice.tar.gz index.html json_schema.json paperfit.js

cat > input.json <<'EOF'
{
  "validate": true,
  "variables": {
    "number": "INV-2026-001",
    "customer": "Acme Corp",
    "items": [
      { "name": "Design", "qty": 1, "price": "$1,200" },
      { "name": "Development", "qty": 40, "price": "$6,000" }
    ]
  }
}
EOF

curl -X POST http://localhost:4000/pdf/from/archive/file \
  -F file=@invoice.tar.gz \
  -F 'input=<input.json' \
  -o invoice.pdf
```

The server injects `window.__PAPERFIT_INPUT__` into the page, and Gotenberg waits for `window.__PAPERFIT_READY__` before rendering.

## AI ready with any model

paperfit speaks [Model Context Protocol](https://modelcontextprotocol.io/), so any MCP-compatible model or agent can use it without custom glue.

- HTTP transport: `POST /mcp`
- stdio transport: `go run ./cmd/mcp`
- Prompt-friendly documentation: `GET /llms.txt`

Available MCP tools:

- `render_pdf_from_html`
- `render_pdf_from_archive`
- `render_pdf_from_archive_url`
- `validate_template_variables`
- `inspect_template_archive`
- `scaffold_paperfit_template`

Point your MCP client at paperfit and use whichever model you prefer.
There are no API keys, model names, or providers baked into the service.

## Docker

Build the paperfit container:

```sh
docker build -t paperfit .
```

The image does not include Gotenberg.
Run it on the same Docker network as a Gotenberg instance and point `PDF_GENERATOR_ENDPOINT` at it:

```sh
docker run --rm -p 4000:4000 \
  -e PDF_GENERATOR_ENDPOINT=http://gotenberg:3000/forms/chromium/convert/html \
  paperfit
```

All `ENV` values in the image are defaults and can be overridden at runtime with `docker run -e`, `--env-file`, or the compose `environment` key.

| Variable | Default | Description |
| --- | --- | --- |
| `SERVER_PORT` | `4000` | HTTP listen port. Publish the same port with `-p`. |
| `PDF_GENERATOR_ENDPOINT` | `http://gotenberg:3000/forms/chromium/convert/html` | Gotenberg conversion endpoint. |
| `WHITELIST_ORIGINS` | `*` | Allowed CORS origins, space separated. |
| `MAX_REQUESTS_PER_SECOND` | `20` | Request rate limit. |
| `CACHE_URL` | temporary local directory | Cache backend, for example `fs:///data/cache` or a Minio/S3 URL. |
| `CACHE_ENCRYPTION_KEY` | unset | Key used to encrypt cached files. |
| `CACHE_DEFAULT_PDF_PATH` | `pdf` | Cache prefix for PDFs. |
| `CACHE_DEFAULT_ARCHIVE_PATH` | `archive` | Cache prefix for archives. |

For example, to run on port 8080 with a persistent file cache:

```sh
docker run --rm -p 8080:8080 \
  -e SERVER_PORT=8080 \
  -e PDF_GENERATOR_ENDPOINT=http://gotenberg:3000/forms/chromium/convert/html \
  -e CACHE_URL=fs:///data/cache \
  -v paperfit-cache:/data \
  paperfit
```

Tagged releases are published to `ghcr.io/dokufy/paperfit`.

## API

- `GET /application/healthy` - Health check.
- `POST /pdf/from/html` - Render a PDF directly from an HTML string.
- `POST /pdf/from/html/file` - Render from an uploaded HTML file.
- `POST /pdf/from/archive/file` - Render from an uploaded archive.
- `POST /pdf/from/archive/url` - Render from an archive URL.
- `POST /mcp` - Model Context Protocol endpoint.
- `GET /llms.txt` - LLM-friendly specification.

### POST /pdf/from/html

```json
{
  "html": "<html>...</html>",
  "assets": { "style.css": "body{}" },
  "jsonSchema": { "type": "object" },
  "input": { "validate": true, "variables": { "title": "Hello" } },
  "options": {
    "landscape": false,
    "waitForExpression": "window.__PAPERFIT_READY__ !== false"
  }
}
```

Variables are injected as `window.__PAPERFIT_INPUT__`.

### paperfit.js API

- `getPaperfitInput`
- `setPaperfitDefaultInput`
- `bindPaperfitVariables`
- `configurePage`
- `renderPaperfit`
- `renderAfterFrameworkMount`
- `createPaperfitTemplate`
- `<paperfit-document>`
- `<paperfit-header>`
- `<paperfit-footer>`
- `<paperfit-page-break>`
- `<paperfit-avoid-break>`
- `<paperfit-page-number>`
- `paperfit-build` CLI
- `paperfitVitePlugin`

## Caching (CACHE_URL)

paperfit supports caching PDFs for better performance.
Set the `CACHE_URL` environment variable to enable a cache backend.
Use `fs://` for local file system cache or a Minio/S3 URL for object storage.
When `CACHE_URL` is unset, paperfit falls back to a temporary local directory.

## Development

Prerequisites:

- Go 1.26 or newer
- Node.js and npm, for the TypeSpec schema
- Docker with Compose, for Gotenberg and Minio
- [Task](https://taskfile.dev/)

Setup:

```sh
go install github.com/go-task/task/v3/cmd/task@latest
export PATH="$(go env GOPATH)/bin:$PATH"

task install
```

Start the development stack in one terminal:

```sh
task docker
```

Then start the watch server in another terminal:

```sh
task dev
```

Run the checks:

```sh
task format
task lint
task test
```

Tests run against an in-process Gotenberg mock when no Gotenberg instance is reachable, so no external services are required.

## Limitations

- Built template assets must be flattened, meaning paths such as `/dirA/dirB/file.js` become `/file.js`.
  This requirement comes from using [Gotenberg](https://gotenberg.dev/) for PDF generation.
- Encryption and decryption for cached files are still in testing.
  Most object storage solutions already offer built-in support for encryption.
