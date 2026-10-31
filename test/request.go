package test

import (
	"bytes"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/dokufy/paperfit/config"
	"github.com/dokufy/paperfit/internal/logger"
)

// initialCacheURL is the CACHE_URL the test run started with. Suites that
// exercise the cache use UseCacheURL to restore it after another suite has
// disabled the cache.
var initialCacheURL = os.Getenv("CACHE_URL")

func UseCacheURL() {
	_ = os.Setenv("CACHE_URL", initialCacheURL)
}

func FullURL(path string) string {
	return "http://localhost:" + config.AppConfig.ServerPort + path
}

// WaitForServer blocks until the server accepts TCP connections on the
// configured port or fails the test after a timeout.
func WaitForServer(tb testing.TB) {
	tb.Helper()

	address := "localhost:" + config.AppConfig.ServerPort
	deadline := time.Now().Add(10 * time.Second)

	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", address, 100*time.Millisecond)
		if err == nil {
			_ = conn.Close()

			return
		}

		time.Sleep(10 * time.Millisecond)
	}

	tb.Fatalf("server at %s did not become ready", address)
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
	res, err := client.Do(req)
	if err != nil {
		logger.Log.Error("Error while sending the request", err)

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
