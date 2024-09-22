package main

import (
	"github.com/dokufy/paperfit/config"
	"github.com/dokufy/paperfit/internal/logger"
	"github.com/dokufy/paperfit/internal/server"
)

func main() {
	config.Load()

	server, err := server.New()
	if err != nil {
		logger.Log.Error("Failed to initialize API", err)
		panic("failed to initialize API server")
	}

	err = server.Start(config.AppConfig.ServerPort)
	if err != nil {
		logger.Log.Error("Failed to start API server", err)
		panic("failed to start API server")
	}
}
