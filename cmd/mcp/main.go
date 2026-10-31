package main

import (
	"context"

	"github.com/dokufy/paperfit/config"
	"github.com/dokufy/paperfit/internal/logger"
	"github.com/dokufy/paperfit/internal/mcp"
	"github.com/dokufy/paperfit/internal/service/pdfgenerator"
)

func main() {
	config.Load()
	logger.Initialize()

	pdfService := pdfgenerator.New(config.AppConfig.PDFgeneratorEndpoint)
	server := mcp.NewStdio(pdfService)

	mcp.ServeStdio(context.Background(), server)
}
