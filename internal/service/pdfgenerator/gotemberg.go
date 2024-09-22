package pdfgenerator

import (
	"io"
	// "log"
	"net/http"

	"github.com/dokufy/paperfit/internal/service"
)

type GotembergGenerator struct {
	endpoint string
}

func New(endpoint string) *GotembergGenerator {
	return &GotembergGenerator{endpoint: endpoint}
}

func (p *GotembergGenerator) Generate(project *service.Project) (io.Reader, int64, error) {
	// TODO: add timeout env variable
	multipart, err := MultipartForm("file", project.GetFiles(), &map[string]string{
		"marginTop":         "0",
		"marginLeft":        "0",
		"marginRight":       "0",
		"marginBottom":      "0",
		"waitTimeout":       "300.0",
		"preferCssPageSize": "true",
	})
	if err != nil {
		return nil, 0, err
	}

	req, err := http.NewRequest(http.MethodPost, p.endpoint, multipart.Body())
	if err != nil {
		return nil, 0, err
	}
	req.Header.Add("Content-Type", multipart.FormDataContentType())

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return nil, 0, err
	}

	switch res.StatusCode {
	case 200:
		return res.Body, res.ContentLength, nil
	case 400:
		return res.Body, 0, service.ErrInvalidRequest
	case 503:
		return res.Body, 0, service.ErrTimeout
	default:
		return res.Body, 0, service.ErrInternalError
	}
}
