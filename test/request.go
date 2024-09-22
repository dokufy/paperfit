package test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/dokufy/paperfit/config"
	"github.com/dokufy/paperfit/internal/logger"
)

func FullURL(path string) string {
	return "http://localhost:" + config.AppConfig.ServerPort + path
}

func Post(url string, headers map[string]string, body io.Reader) *http.Response {
	req, err := http.NewRequest(http.MethodPost, url, body)
	if err != nil {
		logger.Log.Error("Error while creating the request", err)

		panic(err)
	}

	for key, value := range headers {
		req.Header.Set(key, value)
	}

	client := &http.Client{}
	res, error := client.Do(req)
	if error != nil {
		logger.Log.Error("Error while sending the request", error)

		panic(err)
	}

	return res
}

func PrepareBody(t *testing.T, body map[string]any) io.Reader {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}

	return bytes.NewReader(jsonBody)
}
