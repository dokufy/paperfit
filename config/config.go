package config

import (
	"log"
	"log/slog"
	"os"
	"strconv"
	"strings"
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

	AppConfig = Config{
		BasePath:             FetchEnv("BASE_PATH"),
		ServerPort:           FetchEnv("SERVER_PORT"),
		WhitelistOrigins:     strings.Fields(*GetEnv("WHITELIST_ORIGINS", &defaultWhitelistOrigins)),
		MaxRequestsPerSecond: stringToFloat64(*GetEnv("MAX_REQUESTS_PER_SECOND", &defaultMaxRequestsPerSecond)),
		PDFgeneratorEndpoint: FetchEnv("PDF_GENERATOR_ENDPOINT"),
		CacheConfig:          CreateCacheConfig(),
	}
}

func CreateCacheConfig() *CacheConfig {
	defaultPDFpath := "pdf"
	archiveArchivePath := "archive"

	cacheUrl := GetEnv("CACHE_URL", nil)
	if cacheUrl == nil || *cacheUrl == "" {
		return nil
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
