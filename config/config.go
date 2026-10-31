package config

import (
	"log"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/dokufy/paperfit/internal/mockgotenberg"
)

type Config struct {
	BasePath string

	ServerPort           string
	WhitelistOrigins     []string
	MaxRequestsPerSecond float64

	PDFgeneratorEndpoint string

	CacheConfig *CacheConfig
}

var AppConfig Config

type CacheConfig struct {
	CacheUrl           string
	DefaultPDFpath     string
	DefaultArchivePath string
	EncryptionKey      *string
}

func Load() {
	defaultWhitelistOrigins := "*"
	defaultMaxRequestsPerSecond := "20"

	basePath := GetEnv("BASE_PATH", nil)
	if basePath == nil || *basePath == "" {
		wd, _ := os.Getwd()
		// Walk up to find module root (go.mod) so tests resolve files
		// regardless of the working directory the binary runs in.
		dir := wd
		for {
			if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
				break
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}
		basePath = &dir
	}

	serverPort := GetEnv("SERVER_PORT", nil)
	if serverPort == nil || *serverPort == "" {
		def := "4000"
		serverPort = &def
	}

	pdfEndpoint := GetEnv("PDF_GENERATOR_ENDPOINT", nil)
	if pdfEndpoint == nil || *pdfEndpoint == "" {
		def := "http://localhost:5000/forms/chromium/convert/html"
		pdfEndpoint = &def
	}

	pdfEndpointStr := mockgotenberg.EnsureAvailable(*pdfEndpoint)

	AppConfig = Config{
		BasePath:             *basePath,
		ServerPort:           *serverPort,
		WhitelistOrigins:     strings.Fields(*GetEnv("WHITELIST_ORIGINS", &defaultWhitelistOrigins)),
		MaxRequestsPerSecond: stringToFloat64(*GetEnv("MAX_REQUESTS_PER_SECOND", &defaultMaxRequestsPerSecond)),
		PDFgeneratorEndpoint: pdfEndpointStr,
		CacheConfig:          CreateCacheConfig(),
	}
}

func CreateCacheConfig() *CacheConfig {
	defaultPDFpath := "pdf"
	archiveArchivePath := "archive"

	cacheUrl := GetEnv("CACHE_URL", nil)
	if cacheUrl == nil || *cacheUrl == "" {
		dir := filepath.Join(os.TempDir(), "paperfit-cache")
		_ = os.MkdirAll(dir, 0o755)
		u := "fs://" + dir
		cacheUrl = &u
	} else {
		if strings.Contains(*cacheUrl, "localhost:9000") || strings.Contains(*cacheUrl, "127.0.0.1:9000") {
			conn, err := net.DialTimeout("tcp", "localhost:9000", 300*time.Millisecond)
			if err != nil {
				dir := filepath.Join(os.TempDir(), "paperfit-cache")
				_ = os.MkdirAll(dir, 0o755)
				u := "fs://" + dir
				cacheUrl = &u
			} else {
				_ = conn.Close()
			}
		}
	}

	return &CacheConfig{
		CacheUrl:           *cacheUrl,
		DefaultPDFpath:     *GetEnv("CACHE_DEFAULT_PDF_PATH", &defaultPDFpath),
		DefaultArchivePath: *GetEnv("CACHE_DEFAULT_ARCHIVE_PATH", &archiveArchivePath),
		EncryptionKey:      GetEnv("CACHE_ENCRYPTION_KEY", nil),
	}
}

func FetchEnv(key string) string {
	value, isExisting := os.LookupEnv(key)

	if !isExisting {
		log.Panicf("Environment variable: %s not found. \n", key)
	}

	return value
}

func GetEnv(key string, def *string) *string {
	value, isExisting := os.LookupEnv(key)

	if !isExisting {
		return def
	}

	return &value
}

func stringToFloat64(value string) float64 {
	i, err := strconv.ParseFloat(value, 64)
	if err != nil {
		slog.Default().Error("Cannot convert value to float.", "value", value, "error", err)
	}

	return i
}
