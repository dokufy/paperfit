package archivegen

import (
	"bytes"
	"context"
	"errors"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"

	"github.com/atoncetti/combiner/internal/archive"
	"github.com/atoncetti/combiner/internal/cache"
	"github.com/atoncetti/combiner/internal/server/schema"
	"github.com/atoncetti/combiner/internal/service"
	"github.com/atoncetti/combiner/validation"
	"github.com/danielgtaylor/huma/v2"
)

func RegisterOperation(api huma.API, path string, pdfService service.PDFserviceGenerator, cacheInstance *cache.Cache) {
	huma.Register(api, definition(path), resolver(pdfService, cacheInstance))
}

func definition(path string) huma.Operation {
	return huma.Operation{
		OperationID: "generate-pdf-archive",
		Method:      http.MethodPost,
		Path:        path,
		Summary:     "Generate PDF from archive",
		Description: `Takes an archive containing a static HTML project
		consisting of at least a <b>index.html</b> file and optional variables
		to be combined during PDF generation.<br />
		The variables are available as global variables to
		js scripts defined in the project.<br />
		Variables can be validated with a <b>json_schema.json</b>
		file contained in the archive.<br />
		The generated PDF is cached to improve performance,
        but the cache is cleared when the archive field name or variables change.<br />`,
		RequestBody: &huma.RequestBody{
			Description: "Input for document generation",
			Required:    true,
			Content: map[string]*huma.MediaType{
				"application/json": {
					Schema: &huma.Schema{
						Type: huma.TypeObject,
						Properties: map[string]*huma.Schema{
							"template": {
								Type:   huma.TypeString,
								Format: "binary",
							},
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

func resolver(
	pdfService service.PDFserviceGenerator,
	cacheInstance *cache.Cache,
) func(ctx context.Context, input *schema.DocumentArchiveInput) (*huma.StreamResponse, error) {
	return func(ctx context.Context, input *schema.DocumentArchiveInput) (*huma.StreamResponse, error) {
		if len(input.RawBody.File) == 0 {
			return nil, huma.Error400BadRequest("an archive must be provided")
		}

		var vars string = ""
		var validate bool = false

		project, archiveName, err := archiveToProject(ctx, input.RawBody.File)
		if err != nil {
			return nil, huma.Error500InternalServerError("Failed while extracting archive", err)
		}

		if len(input.RawBody.Value["variables"]) > 0 {
			vars = input.RawBody.Value["variables"][0]
		}

		if len(input.RawBody.Value["validate"]) > 0 {
			val, err := strconv.ParseBool(input.RawBody.Value["validate"][0])
			if err != nil {
				return nil, huma.Error400BadRequest("unknown value passed as validate", err)
			}

			validate = val
		}

		pdfCacheKey, err := cache.Key([][]byte{[]byte(archiveName), []byte(vars)})
		if err != nil {
			return nil, huma.Error500InternalServerError("Failed to create cache key")
		}

		archiveCacheKey, err := cache.Key([][]byte{[]byte(archiveName)})
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

		if vars != "" {
			err = project.AppendVariables(vars)
			if err != nil {
				return nil, huma.Error400BadRequest("Failed to parse variables", err)
			}
		}

		if validate {
			jsonSchema := project.GetJSONschemaFile()

			if jsonSchema == nil || jsonSchema.Size() == 0 {
				return nil, huma.Error400BadRequest("No json schema found in the archive, please make sure that a json schema exists and it is not empty or disable variables validation")
			}

			err := validation.Validate(strings.NewReader(vars), jsonSchema.Content)
			if err != nil {
				return nil, huma.Error400BadRequest("Provided variables do not conform to the json schema")
			}
		}

		res, err := pdfService.Generate(project)
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

func archiveToProject(ctx context.Context, file map[string][]*multipart.FileHeader) (*service.Project, string, error) {
	for name, rawFiles := range file {
		for _, rawFile := range rawFiles {
			file, err := rawFile.Open()
			if err != nil {
				return nil, "", err
			}

			project, err := archive.ExtractToProject(ctx, name, file)
			if err != nil {
				return nil, "", err
			}

			return &project, name, nil
		}
	}

	return nil, "", errors.New("no archive file found")
}
