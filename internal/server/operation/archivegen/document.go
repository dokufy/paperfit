package archivegen

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/atoncetti/combiner/internal/cacher"
	"github.com/atoncetti/combiner/internal/httptool"
	"github.com/atoncetti/combiner/internal/server/middleware"
	"github.com/atoncetti/combiner/internal/server/schema"
	"github.com/atoncetti/combiner/internal/service"
	"github.com/atoncetti/combiner/internal/service/projectprovider"
	"github.com/atoncetti/combiner/validation"
	"github.com/danielgtaylor/huma/v2"
)

func RegisterOperation(api huma.API, path string, archiveProjectProvider projectprovider.ArchiveProjectServiceProvider, pdfService service.PDFserviceProvider, cacher *cacher.Cacher) {
	huma.Register(api, definition(api, path, cacher), resolver(archiveProjectProvider, pdfService, cacher))
}

func definition(api huma.API, path string, cacher *cacher.Cacher) huma.Operation {
	return huma.Operation{
		OperationID: "generate-pdf-archive",
		Method:      http.MethodPost,
		Path:        path,
		Middlewares: huma.Middlewares{middleware.GetPdfCacheResponseMiddleware(api, cacher)},
		Summary:     "Generates a PDF document from given archive",
		Tags:        []string{"Generate from archive"},
		RequestBody: &huma.RequestBody{
			Description: "Input for document generation",
			Required:    true,
			Content: map[string]*huma.MediaType{
				"application/json": {
					Schema: &huma.Schema{
						Type: huma.TypeObject,
						Properties: map[string]*huma.Schema{
							"url": {
								Type:        huma.TypeString,
								Description: "Url of the archive",
								WriteOnly:   true,
								Format:      "uri",
							},
							"headers": {
								Type:        huma.TypeObject,
								Description: "Headers to use when making a request to the specified Url",
								Properties:  map[string]*huma.Schema{},
								WriteOnly:   true,
							},
							"input": {
								Type:        huma.TypeObject,
								Description: "Input argument options",
								Properties: map[string]*huma.Schema{
									"arguments": {
										Type:        huma.TypeString,
										Description: "Input arguments that will be replaced in the document",
										WriteOnly:   true,
									},
									"validate": {
										Type:        huma.TypeBoolean,
										Description: "Whether to validate the arguments against a json schema or not",
										WriteOnly:   true,
									},
								},
							},
						},
						Required: []string{"url"},
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

func resolver(
	archiveProjectProvider projectprovider.ArchiveProjectServiceProvider,
	pdfService service.PDFserviceProvider,
	cache *cacher.Cacher,
) func(ctx context.Context, input *schema.DocumentArchiveInput) (*huma.StreamResponse, error) {
	return func(ctx context.Context, input *schema.DocumentArchiveInput) (*huma.StreamResponse, error) {
		if input.Body.URL == "" {
			return nil, huma.Error400BadRequest("No url provided")
		}

		pdfCacheKey, err := cacher.Key([][]byte{[]byte(input.Body.URL), httptool.FlattenHeaders(input.Body.Headers), []byte(input.Body.Input.Arguments)})
		if err != nil {
			return nil, huma.Error500InternalServerError("Failed to create cache key")
		}

		archiveCacheKey, err := cacher.Key([][]byte{[]byte(input.Body.URL), httptool.FlattenHeaders(input.Body.Headers)})
		if err != nil {
			return nil, huma.Error500InternalServerError("Failed to create cache key")
		}

		if cache != nil {
			cachedPdf, err := cache.GetPDF(ctx, pdfCacheKey)
			if err != nil {
				return nil, huma.Error500InternalServerError("Failed to read cache", err)
			}

			if cachedPdf != nil {
				buf := new(bytes.Buffer)
				buf.ReadFrom(cachedPdf)

				return &huma.StreamResponse{Body: pdfBody(buf.Bytes())}, nil
			}
		}

		pr, err := archiveProjectProvider.GetProject(ctx, input.Body.URL, input.Body.Headers)
		if err != nil {
			if projectErr, ok := err.(*projectprovider.RequestError); ok {
				if projectErr.StatusCode == 404 {
					return nil, huma.Error404NotFound("Archive not found")
				}
			}

			return nil, huma.Error500InternalServerError("Failed to get archive from URL", err)
		}

		if input.Body.Input.Validate {
			jsonSchema := pr.GetJSONschema()

			if jsonSchema == nil {
				return nil, huma.Error404NotFound("No json schema found in the archive, please check that a json schema exists in the archive or disable arguments validation")
			}

			err := validation.Validate(strings.NewReader(input.Body.Input.Arguments), jsonSchema.Content)
			if err != nil {
				return nil, huma.Error400BadRequest("Provided arguments do not conform to the json schema")
			}
		}

		if input.Body.Input.Arguments != "" {
			var parsedArgs map[string]interface{}
			err = json.Unmarshal([]byte(input.Body.Input.Arguments), &parsedArgs)
			if err != nil {
				return nil, huma.Error500InternalServerError("Failed to parse arguments", err)
			}

			pr.AppendArgumentsFile(input.Body.Input.Arguments)
		}

		res, err := pdfService.Generate(&pr)
		if err != nil {
			if cache != nil {
				err2 := cache.DeleteArchive(ctx, archiveCacheKey)
				if err2 != nil {
					return nil, huma.Error500InternalServerError("Failed to cleanup archive", err2)
				}
			}

			if errors.Is(err, service.ErrInvalidRequest) {
				return nil, huma.Error400BadRequest("Bad request", err)
			}

			if errors.Is(err, service.ErrTimeout) {
				return nil, huma.Error503ServiceUnavailable("Service unavailable", err)
			}

			return nil, huma.Error500InternalServerError("Failed to generate pdf", err)
		}

		if cache != nil {
			err = cache.SetPDF(ctx, pdfCacheKey, bytes.NewReader(res), int64(len(res)))
			if err != nil {
				return nil, huma.Error500InternalServerError("Failed to set cache value", err)
			}
		}

		return &huma.StreamResponse{Body: pdfBody(res)}, nil
	}
}

func pdfBody(pdfContent []byte) func(ctx huma.Context) {
	return func(ctx huma.Context) {
		ctx.SetHeader("Content-Type", "application/pdf")

		_, err := ctx.BodyWriter().Write(pdfContent)
		if err != nil {
			return
		}
	}
}
