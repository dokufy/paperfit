package htmlgen

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/atoncetti/combiner/internal/server/schema"
	"github.com/atoncetti/combiner/internal/service"
	"github.com/danielgtaylor/huma/v2"
)

func RegisterOperation(api huma.API, path string, pdfService service.PDFserviceProvider) {
	huma.Register(api, definition(path), resolver(pdfService))
}

func definition(path string) huma.Operation {
	return huma.Operation{
		OperationID: "generate-pdf-explicit",
		Method:      http.MethodPost,
		Path:        path,
		Summary:     "Generates a PDF document from given files",
		Tags:        []string{"Generate from files"},
		RequestBody: &huma.RequestBody{
			Description: "Input for document generation",
			Required:    true,
			Content: map[string]*huma.MediaType{
				"multipart/form-data": {
					Schema: &huma.Schema{
						Type: huma.TypeObject,
						Properties: map[string]*huma.Schema{
							"template": {
								Type: huma.TypeString,
							},
							"variables": {
								Type:        huma.TypeString,
								Description: "Input variables that will be replaced in the document",
							},
						},
						Required: []string{"template"},
					},
				},
			},
		},
		Responses: map[string]*huma.Response{
			"200": {
				Description: "Successfully returns a PDF file",
				Headers: map[string]*huma.Param{
					"Content-Type": {
						Schema: &huma.Schema{
							Default: "application/pdf",
						},
						Description: "Content-Type of the response",
						In:          "header",
						Required:    true,
					},
				},
				Content: map[string]*huma.MediaType{
					"application/pdf": {
						Schema: &huma.Schema{
							Description: "PDF file",
							Type:        huma.TypeString,
							Format:      "binary",
							Required:    []string{"file"},
						},
					},
				},
			},
			"400": {
				Description: "Malformed request",
			},
			"422": {
				Description: "PDF document could not be generated",
			},
			"500": {
				Description: "Internal server error",
			},
		},
	}
}

func resolver(pdfService service.PDFserviceProvider) func(ctx context.Context, input *schema.DocumentFileInput) (*schema.DocumentOutput, error) {
	return func(ctx context.Context, input *schema.DocumentFileInput) (*schema.DocumentOutput, error) {
		resp := &schema.DocumentOutput{}

		pr := service.NewProject([]*service.File{})

		if len(input.RawBody.File) == 0 {
			return nil, huma.Error400BadRequest("at least one file must be provided")
		}

		for name, rawFiles := range input.RawBody.File {
			for _, rawFile := range rawFiles {
				file, err := rawFile.Open()
				if err != nil {
					// logger.Logger.Error("failed to open file: " + name)
					return nil, huma.Error500InternalServerError("Failed to open file", err)
				}

				buf := make([]byte, rawFile.Size)
				_, err = file.Read(buf)
				if err != nil {
					// logger.Logger.Error("failed while reading file: " + name)
					return nil, huma.Error500InternalServerError("Failed while reading file", err)
				}

				pr.AppendFile(name, bytes.NewReader(buf), rawFile.Size)
			}
		}

		vars := input.RawBody.Value["variables"]
		if vars != nil && vars[0] != "" {
			var parsedArgs map[string]any
			err := json.Unmarshal([]byte(input.RawBody.Value["variables"][0]), &parsedArgs)
			if err != nil {
				return nil, huma.Error400BadRequest("Failed to parse variables", err)
			}

			pr.AppendVariables(vars[0])
		}

		res, err := pdfService.Generate(&pr)
		if err != nil {
			if errors.Is(err, service.ErrInvalidRequest) {
				return nil, huma.Error400BadRequest("Bad request", err)
			}

			if errors.Is(err, service.ErrTimeout) {
				return nil, huma.Error503ServiceUnavailable("Service unavailable", err)
			}

			return nil, huma.Error500InternalServerError("Failed to generate pdf", err)
		}

		resp.Body = res

		return resp, nil
	}
}
