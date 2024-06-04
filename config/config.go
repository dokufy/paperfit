package config

import (
	"log"
	"os"
)

type Config struct {
	BasePath string

	ServerGraphqlPort             string
	ServerGraphqlWhitelistOrigins string

	PDFgeneratorEndpoint string

	CacheUrl *string
}

var AppConfig Config

func Load() {
	AppConfig = Config{
		BasePath:                      FetchEnv("BASE_PATH"),
		ServerGraphqlPort:             FetchEnv("SERVER_GRAPHQL_PORT"),
		ServerGraphqlWhitelistOrigins: FetchEnv("SERVER_GRAPHQL_WHITELIST_ORIGINS"),
		PDFgeneratorEndpoint:          FetchEnv("PDF_GENERATOR_ENDPOINT"),
		CacheUrl:                      GetEnv("CACHE_URL", nil),
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
