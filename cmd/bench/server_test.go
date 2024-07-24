package server_benchmark

import (
	"strings"
	"testing"

	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/dokufy/paperfit/config"
	"github.com/dokufy/paperfit/internal/cache"
	"github.com/dokufy/paperfit/internal/logger"
	"github.com/dokufy/paperfit/internal/server"
	"github.com/dokufy/paperfit/internal/service/pdfgenerator"
	"github.com/dokufy/paperfit/test/mocks"
)

func setupApi(b *testing.B) humatest.TestAPI {
	logger := logger.New()
	cacheInstance, err := cache.New(config.Config.CacheConfig.CacheUrl, nil)
	if err != nil {
		b.Fatal("failed to create cache")
	}

	_, api := humatest.New(b)
	server.AddRoutes(api, nil, cacheInstance, &logger)

	return api
}

func BenchmarkExplicitPdfGeneration_300k_characters(b *testing.B) {
	mocks.PassPDFgenerator()

	contentCharacters := 300000

	api := setupApi(b)

	project := mocks.ProjectMock(createStringSizeN(contentCharacters))

	for i := 0; i < b.N; i++ {
		multipart, err := pdfgenerator.MultipartForm(project.GetFiles(), nil)
		if err != nil {
			b.Fatal(err)
		}

		api.Post("/pdf/generate/html", "Content-type: "+multipart.FormDataContentType(), multipart.Body())
	}
}

// func BenchmarkExplicitPdfGeneration_1M_characters(b *testing.B) {
// 	mocks.PassPDFgenerator()

// 	contentCharacters := 10000000

// 	api := setupApi(b)

// 	project := mocks.ProjectMock(createStringSizeN(contentCharacters))

// 	for i := 0; i < b.N; i++ {
// 		multipart, err := pdfgenerator.MultipartForm(project.GetFiles(), nil)
// 		if err != nil {
// 			b.Fatal(err)
// 		}

// 		api.Post("/pdf/generate/html", "Content-type: "+multipart.FormDataContentType(), multipart.Body())
// 	}

// }

// func BenchmarkExplicitPdfGeneration_10M_characters(b *testing.B) {
// 	mocks.PassPDFgenerator()

// 	contentCharacters := 10000000

// 	api := setupApi(b)

// 	project := mocks.ProjectMock(createStringSizeN(contentCharacters))

// 	for i := 0; i < b.N; i++ {
// 		multipart, err := pdfgenerator.MultipartForm(project.GetFiles(), nil)
// 		if err != nil {
// 			b.Fatal(err)
// 		}

// 		api.Post("/pdf/generate/html", "Content-type: "+multipart.FormDataContentType(), multipart.Body())
// 	}
// }

// func BenchmarkExplicitPdfGeneration_20M_characters(b *testing.B) {
// 	mocks.PassPDFgenerator()

// 	contentCharacters := 20000000

// 	api := setupApi(b)

// 	project := mocks.ProjectMock(createStringSizeN(contentCharacters))

// 	for i := 0; i < b.N; i++ {
// 		multipart, err := pdfgenerator.MultipartForm(project.GetFiles(), nil)
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
