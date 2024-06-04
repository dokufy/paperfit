package server_benchmark

import (
	"strings"
	"testing"

	"github.com/atoncetti/combiner/config"
	"github.com/atoncetti/combiner/internal/cacher"
	"github.com/atoncetti/combiner/internal/logger"
	"github.com/atoncetti/combiner/internal/server"
	"github.com/atoncetti/combiner/internal/service/pdfprovider"
	"github.com/atoncetti/combiner/test/mocks"
	"github.com/danielgtaylor/huma/v2/humatest"
)

func setupApi(b *testing.B) humatest.TestAPI {
	logger := logger.New()
	cacher, err := cacher.New(config.AppConfig.CacheUrl, logger)
	if err != nil {
		b.Fatal("failed to create cache")
	}

	_, api := humatest.New(b)
	server.AddRoutes(api, nil, &cacher, &logger)

	return api
}

func BenchmarkExplicitPdfGeneration_300k_characters(b *testing.B) {
	mocks.NoMockPDFgenerator()

	contentCharacters := 300000

	api := setupApi(b)

	project := mocks.ProjectMock(createStringSizeN(contentCharacters))

	for i := 0; i < b.N; i++ {
		multipart, err := pdfprovider.MultipartForm(project.GetFiles(), nil)
		if err != nil {
			b.Fatal(err)
		}

		api.Post("/pdf/generate/html", "Content-type: "+multipart.FormDataContentType(), multipart.Body())
	}
}

// func BenchmarkExplicitPdfGeneration_1M_characters(b *testing.B) {
// 	mocks.NoMockPDFgenerator()

// 	contentCharacters := 10000000

// 	api := setupApi(b)

// 	project := mocks.ProjectMock(createStringSizeN(contentCharacters))

// 	for i := 0; i < b.N; i++ {
// 		multipart, err := pdfprovider.MultipartForm(project.GetFiles(), nil)
// 		if err != nil {
// 			b.Fatal(err)
// 		}

// 		api.Post("/pdf/generate/html", "Content-type: "+multipart.FormDataContentType(), multipart.Body())
// 	}

// }

// func BenchmarkExplicitPdfGeneration_10M_characters(b *testing.B) {
// 	mocks.NoMockPDFgenerator()

// 	contentCharacters := 10000000

// 	api := setupApi(b)

// 	project := mocks.ProjectMock(createStringSizeN(contentCharacters))

// 	for i := 0; i < b.N; i++ {
// 		multipart, err := pdfprovider.MultipartForm(project.GetFiles(), nil)
// 		if err != nil {
// 			b.Fatal(err)
// 		}

// 		api.Post("/pdf/generate/html", "Content-type: "+multipart.FormDataContentType(), multipart.Body())
// 	}
// }

// func BenchmarkExplicitPdfGeneration_20M_characters(b *testing.B) {
// 	mocks.NoMockPDFgenerator()

// 	contentCharacters := 20000000

// 	api := setupApi(b)

// 	project := mocks.ProjectMock(createStringSizeN(contentCharacters))

// 	for i := 0; i < b.N; i++ {
// 		multipart, err := pdfprovider.MultipartForm(project.GetFiles(), nil)
// 		if err != nil {
// 			b.Fatal(err)
// 		}

// 		api.Post("/pdf/generate/html", "Content-type: "+multipart.FormDataContentType(), multipart.Body())
// 	}
// }

func createStringSizeN(n int) string {
	var sb strings.Builder

	for i := 0; i < n; i++ {
		sb.WriteString("a")
	}

	return sb.String()
}
