package server_benchmark

import (
	"strings"
	"testing"

	"github.com/dokufy/paperfit/config"
	"github.com/dokufy/paperfit/internal/server"
	"github.com/dokufy/paperfit/internal/service/pdfgenerator"
	"github.com/dokufy/paperfit/test"
	"github.com/dokufy/paperfit/test/mocks"
)

func setupServer(b *testing.B) *server.Server {
	config.Load()
	mocks.PassPDFgenerator()
	mocks.PassPath("/pdf/from/html/file")

	server, err := server.New()
	if err != nil {
		b.Fatal(err)
	}

	go server.Start(config.AppConfig.ServerPort)

	return server
}

// func BenchmarkExplicitPdfGeneration_300k_characters(b *testing.B) {
// 	contentCharacters := 300000

// 	server := setupServer(b)
// 	defer server.Stop()

// 	project := mocks.ProjectMock(createStringSizeN(contentCharacters))

// 	for i := 0; i < b.N; i++ {
// 		multipart, err := pdfgenerator.MultipartForm("files", project.GetFiles(), nil)
// 		if err != nil {
// 			b.Fatal(err)
// 		}

// 		test.Post(
// 			test.FullURL("/pdf/from/html/file"),
// 			map[string]string{
// 				"Content-Type": multipart.FormDataContentType(),
// 			},
// 			multipart.Body(),
// 		)
// 	}
// }

// func BenchmarkExplicitPdfGeneration_1M_characters(b *testing.B) {
// 	contentCharacters := 10000000

// 	server := setupServer(b)
// 	defer server.Stop()

// 	project := mocks.ProjectMock(createStringSizeN(contentCharacters))

// 	for i := 0; i < b.N; i++ {
// 		multipart, err := pdfgenerator.MultipartForm("files", project.GetFiles(), nil)
// 		if err != nil {
// 			b.Fatal(err)
// 		}

// 		test.Post(
// 			test.FullURL("/pdf/from/html/file"),
// 			map[string]string{
// 				"Content-Type": multipart.FormDataContentType(),
// 			},
// 			multipart.Body(),
// 		)
// 	}

// }

func BenchmarkExplicitPdfGeneration_10M_characters(b *testing.B) {
	contentCharacters := 10000000

	server := setupServer(b)
	defer server.Stop()

	project := mocks.ProjectMock(createStringSizeN(contentCharacters))

	for i := 0; i < b.N; i++ {
		multipart, err := pdfgenerator.MultipartForm("files", project.GetFiles(), nil)
		if err != nil {
			b.Fatal(err)
		}

		test.Post(
			test.FullURL("/pdf/from/html/file"),
			map[string]string{
				"Content-Type": multipart.FormDataContentType(),
			},
			multipart.Body(),
		)
	}
}

// func BenchmarkExplicitPdfGeneration_20M_characters(b *testing.B) {
// 	contentCharacters := 20000000

// 	server := setupServer(b)
// 	defer server.Stop()

// 	project := mocks.ProjectMock(createStringSizeN(contentCharacters))

// 	for i := 0; i < b.N; i++ {
// 		multipart, err := pdfgenerator.MultipartForm("files", project.GetFiles(), nil)
// 		if err != nil {
// 			b.Fatal(err)
// 		}

// 		test.Post(
// 			test.FullURL("/pdf/from/html/file"),
// 			map[string]string{
// 				"Content-Type": multipart.FormDataContentType(),
// 			},
// 			multipart.Body(),
// 		)
// 	}
// }

func createStringSizeN(n int) string {
	var sb strings.Builder

	for i := 0; i < n; i++ {
		sb.WriteString("a")
	}

	return sb.String()
}
