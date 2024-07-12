package pdfgenerator

import (
	"errors"
	"io"
	"net/http"

	"github.com/atoncetti/combiner/internal/service"
)

type HTTPpdfGenerator struct {
	endpoint string
}

func NewHTTPpdfGenerator(endpoint string) *HTTPpdfGenerator {
	return &HTTPpdfGenerator{endpoint: endpoint}
}

func (p *HTTPpdfGenerator) Generate(project *service.Project) ([]byte, error) {
	// TODO: add timeout env variable
	multipart, err := MultipartForm(project.GetFiles(), &map[string]string{
		"marginTop":         "0",
		"marginLeft":        "0",
		"marginRight":       "0",
		"marginBottom":      "0",
		"waitTimeout":       "300.0",
		"preferCssPageSize": "true",
	})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", p.endpoint, multipart.Body())
	if err != nil {
		return []byte{}, err
	}
	req.Header.Add("Content-Type", multipart.FormDataContentType())

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return []byte{}, err
	}
	defer res.Body.Close()

	switch res.StatusCode {
	case 200:
		cnt, err := io.ReadAll(res.Body)
		if err != nil {
			return []byte{}, err
		}
		return cnt, nil
	case 400:
		cnt, err := io.ReadAll(res.Body)
		if err != nil {
			return []byte{}, err
		}

		return []byte{}, errors.Join(service.ErrInvalidRequest, errors.New(string(cnt)))
	case 503:
		cnt, err := io.ReadAll(res.Body)
		if err != nil {
			return []byte{}, err
		}

		return []byte{}, errors.Join(service.ErrTimeout, errors.New(string(cnt)))
	default:
		return []byte{}, service.ErrInternalError
	}
}
