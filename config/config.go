package config

import (
	"log"
	"os"
)

type EnvConfig struct {
	BasePath string

	ServerRESTPort string

	PDFgeneratorEndpoint string

	CacheConfig *CacheConfig
}

var Config EnvConfig

type CacheConfig struct {
	CacheUrl      string
	EncryptionKey *string
}

func Load() {
	Config = EnvConfig{
		BasePath:             FetchEnv("BASE_PATH"),
		ServerRESTPort:       FetchEnv("SERVER_REST_PORT"),
		PDFgeneratorEndpoint: FetchEnv("PDF_GENERATOR_ENDPOINT"),
		CacheConfig:          CreateCacheConfig(),
	}
}

func CreateCacheConfig() *CacheConfig {
	cacheUrl := GetEnv("CACHE_URL", nil)
	if cacheUrl == nil {
		return nil
	}

	return &CacheConfig{
		CacheUrl:      *cacheUrl,
		EncryptionKey: GetEnv("CACHE_ENCRYPTION_KEY", nil),
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
