package server_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"testing"

	"github.com/dokufy/paperfit/config"
	"github.com/dokufy/paperfit/internal/server"
	"github.com/dokufy/paperfit/internal/service"
	"github.com/dokufy/paperfit/internal/service/pdfgenerator"
	"github.com/dokufy/paperfit/test"
	"github.com/dokufy/paperfit/test/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type VariableSustitutionTestSuite struct {
	suite.Suite
	ctx    context.Context
	server *server.Server
	assert *assert.Assertions
}

func (s *VariableSustitutionTestSuite) SetupSuite() {
	os.Setenv("CACHE_URL", "")

	config.Load()
	mocks.PassPDFgenerator()
	mocks.PassPath("/pdf/from/archive/file")

	server, err := server.New()
	if err != nil {
		s.T().Fatal(err)
	}

	go server.Start(config.Config.ServerPort)

	s.ctx = context.Background()
	s.server = server
	s.assert = assert.New(s.T())
}

func (s *VariableSustitutionTestSuite) TearDownSuite() {
	s.server.Stop()
}

func (s *VariableSustitutionTestSuite) Test_Substitute_Array_Boolean__Succeeds() {
	file := test.File("archive/valid/variable_substitution/boolean.tar.gz")
	isDone := true

	multipart, _ := pdfgenerator.MultipartForm(
		"file",
		[]*service.File{&file},
		&map[string]string{
			"input": fmt.Sprintf(`{"variables": {"isDone": %t}}`, isDone),
		},
	)

	resp := test.Post(
		test.FullURL("/pdf/from/archive/file"),
		map[string]string{
			"Content-Type": multipart.FormDataContentType(),
		},
		multipart.Body(),
	)

	s.assert.Contains(test.PdfToString(resp.Body), fmt.Sprintf("%t", isDone))
	s.assert.Equal(http.StatusOK, resp.StatusCode)
}

func (s *VariableSustitutionTestSuite) Test_Substitute_String_Variable__Succeeds() {
	file := test.File("archive/valid/variable_substitution/string.tar.gz")
	title := "this is a title"

	multipart, _ := pdfgenerator.MultipartForm(
		"file",
		[]*service.File{&file},
		&map[string]string{
			"input": fmt.Sprintf(`{"variables": {"title": "%s"}}`, title),
		},
	)

	resp := test.Post(
		test.FullURL("/pdf/from/archive/file"),
		map[string]string{
			"Content-Type": multipart.FormDataContentType(),
		},
		multipart.Body(),
	)

	s.assert.Contains(test.PdfToString(resp.Body), title)
	s.assert.Equal(http.StatusOK, resp.StatusCode)
}

func (s *VariableSustitutionTestSuite) Test_Substitute_Number_Variable__Succeeds() {
	file := test.File("archive/valid/variable_substitution/number.tar.gz")
	amount := 5

	multipart, _ := pdfgenerator.MultipartForm(
		"file",
		[]*service.File{&file},
		&map[string]string{
			"input": fmt.Sprintf(`{"variables": {"amount": %d}}`, amount),
		},
	)

	resp := test.Post(
		test.FullURL("/pdf/from/archive/file"),
		map[string]string{
			"Content-Type": multipart.FormDataContentType(),
		},
		multipart.Body(),
	)

	s.assert.Contains(test.PdfToString(resp.Body), fmt.Sprintf("%d", amount))
	s.assert.Equal(http.StatusOK, resp.StatusCode)
}

func (s *VariableSustitutionTestSuite) Test_Substitute_Array_Variable__Succeeds() {
	file := test.File("archive/valid/variable_substitution/array.tar.gz")
	fruits := []string{"orange", "apple", "lemon"}
	json, _ := json.Marshal(fruits)

	multipart, _ := pdfgenerator.MultipartForm(
		"file",
		[]*service.File{&file},
		&map[string]string{
			"input": fmt.Sprintf(`{"variables": {"fruits": %s}}`, json),
		},
	)

	resp := test.Post(
		test.FullURL("/pdf/from/archive/file"),
		map[string]string{
			"Content-Type": multipart.FormDataContentType(),
		},
		multipart.Body(),
	)

	content := test.PdfToString(resp.Body)

	s.assert.Contains(content, fruits[0])
	s.assert.Contains(content, fruits[1])
	s.assert.Contains(content, fruits[2])
	s.assert.Equal(http.StatusOK, resp.StatusCode)
}

func (s *VariableSustitutionTestSuite) Test_Substitute_Array_Object__Succeeds() {
	file := test.File("archive/valid/variable_substitution/object.tar.gz")

	title := "citrus fruit list"
	fruits := []string{"orange", "mandarin", "lemon"}
	haveStones := true
	ph := 5

	info := map[string]any{
		"title":      title,
		"fruits":     fruits,
		"haveStones": haveStones,
		"ph":         ph,
	}
	json, _ := json.Marshal(info)

	multipart, _ := pdfgenerator.MultipartForm(
		"file",
		[]*service.File{&file},
		&map[string]string{
			"input": fmt.Sprintf(`{"variables": {"info": %s}}`, json),
		},
	)

	resp := test.Post(
		test.FullURL("/pdf/from/archive/file"),
		map[string]string{
			"Content-Type": multipart.FormDataContentType(),
		},
		multipart.Body(),
	)

	content := test.PdfToString(resp.Body)

	s.assert.Contains(content, title)
	s.assert.Contains(content, fruits[0])
	s.assert.Contains(content, fruits[1])
	s.assert.Contains(content, fruits[2])
	s.assert.Contains(content, fmt.Sprintf("%t", haveStones))
	s.assert.Contains(content, fmt.Sprintf("%d", ph))
	s.assert.Equal(http.StatusOK, resp.StatusCode)
}

func TestVariableSubstitution(t *testing.T) {
	t.Parallel()

	suite.Run(t, new(VariableSustitutionTestSuite))
}
