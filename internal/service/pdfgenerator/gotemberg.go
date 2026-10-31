package pdfgenerator

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strconv"

	"github.com/dokufy/paperfit/internal/service"
)

type GotembergGenerator struct {
	endpoint string
}

func New(endpoint string) *GotembergGenerator {
	return &GotembergGenerator{endpoint: endpoint}
}

func (p *GotembergGenerator) Generate(ctx context.Context, project *service.Project) (io.Reader, int64, error) {
	fields := map[string]string{
		"marginTop":         "0",
		"marginLeft":        "0",
		"marginRight":       "0",
		"marginBottom":      "0",
		"waitTimeout":       "300.0",
		"preferCssPageSize": "true",
		"printBackground":   "true",
		"waitForExpression": "window.__PAPERFIT_READY__ !== false",
	}

	if project != nil && project.RenderOptions != nil {
		o := project.RenderOptions
		if o.Landscape != nil {
			fields["landscape"] = strconv.FormatBool(*o.Landscape)
		}
		if o.WaitDelay != nil {
			fields["waitDelay"] = *o.WaitDelay
		}
		if o.WaitForExpression != nil {
			fields["waitForExpression"] = *o.WaitForExpression
		}
		if o.PaperWidth != nil {
			fields["paperWidth"] = *o.PaperWidth
		}
		if o.PaperHeight != nil {
			fields["paperHeight"] = *o.PaperHeight
		}
		if o.MarginTop != nil {
			fields["marginTop"] = *o.MarginTop
		}
		if o.MarginBottom != nil {
			fields["marginBottom"] = *o.MarginBottom
		}
		if o.MarginLeft != nil {
			fields["marginLeft"] = *o.MarginLeft
		}
		if o.MarginRight != nil {
			fields["marginRight"] = *o.MarginRight
		}
		if o.PreferCssPageSize != nil {
			fields["preferCssPageSize"] = strconv.FormatBool(*o.PreferCssPageSize)
		}
		if o.PrintBackground != nil {
			fields["printBackground"] = strconv.FormatBool(*o.PrintBackground)
		}
	}

	// ensure required defaults if not overridden
	if _, ok := fields["preferCssPageSize"]; !ok {
		fields["preferCssPageSize"] = "true"
	}
	if _, ok := fields["printBackground"]; !ok {
		fields["printBackground"] = "true"
	}
	if _, ok := fields["waitForExpression"]; !ok {
		fields["waitForExpression"] = "window.__PAPERFIT_READY__ !== false"
	}

	multipart, err := MultipartForm("file", project.GetFiles(), &fields)
	if err != nil {
		return nil, 0, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.endpoint, multipart.Body())
	if err != nil {
		return nil, 0, err
	}
	req.Header.Add("Content-Type", multipart.FormDataContentType())

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = res.Body.Close() }()

	pdfBytes, err := io.ReadAll(res.Body)
	if err != nil {
		return bytes.NewReader(pdfBytes), 0, err
	}

	switch res.StatusCode {
	case 200:
		return bytes.NewReader(pdfBytes), int64(len(pdfBytes)), nil
	case 400:
		return bytes.NewReader(pdfBytes), 0, service.ErrInvalidRequest
	case 503:
		return bytes.NewReader(pdfBytes), 0, service.ErrTimeout
	default:
		return bytes.NewReader(pdfBytes), 0, service.ErrInternalError
	}
}
