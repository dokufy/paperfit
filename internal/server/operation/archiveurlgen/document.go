package archiveurlgen

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/atoncetti/combiner/internal/cache"
	"github.com/atoncetti/combiner/internal/httptool"
	"github.com/atoncetti/combiner/internal/server/schema"
	"github.com/atoncetti/combiner/internal/service"
	"github.com/atoncetti/combiner/internal/service/projectprovider"
	"github.com/atoncetti/combiner/validation"
	"github.com/danielgtaylor/huma/v2"
)

func RegisterOperation(api huma.API, path string, archiveProjectProvider projectprovider.ArchiveProjectServiceProvider, pdfService service.PDFserviceGenerator, cacheInstance *cache.Cache) {
	huma.Register(api, definition(path), resolver(archiveProjectProvider, pdfService, cacheInstance))
}

func definition(path string) huma.Operation {
	return huma.Operation{
		OperationID: "generate-pdf-archive-url",
		Method:      http.MethodPost,
		Path:        path,
		Summary:     "Generate PDF from archive url",
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
									"variables": {
										Type:        huma.TypeString,
										Description: "Input variables that will be replaced in the document",
										WriteOnly:   true,
									},
									"validate": {
										Type:        huma.TypeBoolean,
										Description: "Whether to validate the variables against a json schema or not",
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
	pdfService service.PDFserviceGenerator,
	cacheInstance *cache.Cache,
) func(ctx context.Context, input *schema.DocumentArchiveUrlInput) (*huma.StreamResponse, error) {
	return func(ctx context.Context, input *schema.DocumentArchiveUrlInput) (*huma.StreamResponse, error) {
		if input.Body.URL == "" {
			return nil, huma.Error400BadRequest("No url provided")
		}

		pdfCacheKey, err := cache.Key([][]byte{[]byte(input.Body.URL), httptool.FlattenHeaders(input.Body.Headers), []byte(input.Body.Input.Variables)})
		if err != nil {
			return nil, huma.Error500InternalServerError("Failed to create cache key")
		}

		archiveCacheKey, err := cache.Key([][]byte{[]byte(input.Body.URL), httptool.FlattenHeaders(input.Body.Headers)})
		if err != nil {
			return nil, huma.Error500InternalServerError("Failed to create cache key")
		}

		if cacheInstance != nil {
			cachedPdf, err := cacheInstance.GetPDF(ctx, pdfCacheKey)
			if err != nil {
				return nil, huma.Error500InternalServerError("Failed to read cache", err)
			}

			if cachedPdf != nil {
				buf := new(bytes.Buffer)
				buf.ReadFrom(cachedPdf)

				return &huma.StreamResponse{Body: pdfBody(buf.Bytes())}, nil
			}
		}

		project, err := archiveProjectProvider.GetProject(ctx, input.Body.URL, input.Body.Headers)
		if err != nil {
			if projectErr, ok := err.(*projectprovider.RequestError); ok {
				if projectErr.StatusCode == 404 {
					return nil, huma.Error404NotFound("Archive not found")
				}
			}

			return nil, huma.Error500InternalServerError("Failed to get archive from URL", err)
		}

		if input.Body.Input.Validate {
			jsonSchema := project.GetJSONschemaFile()

			if jsonSchema == nil {
				return nil, huma.Error404NotFound("No json schema found in the archive, please make sure that a json schema exists in the archive or disable variables validation")
			}

			err := validation.Validate(strings.NewReader(input.Body.Input.Variables), jsonSchema.Content)
			if err != nil {
				return nil, huma.Error400BadRequest("Provided variables do not conform to the json schema")
			}
		}

		err = project.AppendVariables(input.Body.Input.Variables)
		if err != nil {
			return nil, huma.Error500InternalServerError("Failed to parse variables", err)
		}

		res, err := pdfService.Generate(&project)
		if err != nil {
			if cacheInstance != nil {
				if err2 := cacheInstance.DeleteArchive(ctx, archiveCacheKey); err2 != nil {
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

		if cacheInstance != nil {
			err = cacheInstance.SetPDF(ctx, pdfCacheKey, bytes.NewReader(res), int64(len(res)))
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
