package server

import (
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/dokufy/paperfit/config"
	"github.com/dokufy/paperfit/internal/cache"
	"github.com/dokufy/paperfit/internal/logger"
	"github.com/dokufy/paperfit/internal/server/operation/archivegen"
	"github.com/dokufy/paperfit/internal/server/operation/archiveurlgen"
	"github.com/dokufy/paperfit/internal/server/operation/htmlgen"
	"github.com/dokufy/paperfit/internal/service"
	"github.com/dokufy/paperfit/internal/service/pdfgenerator"
	"github.com/dokufy/paperfit/internal/service/projectprovider"
	"github.com/go-chi/chi/v5"
)

func New(port string) {
	config.Load()

	logger := logger.New()

	httpPDFprovider := pdfgenerator.NewHTTPpdfGenerator(config.Config.PDFgeneratorEndpoint)

	var c *cache.Cache
	if config.Config.CacheConfig != nil {
		opts := []cache.CacherOptions{cache.WithLogger(&logger)}

		if config.Config.CacheConfig.EncryptionKey != nil {
			opts = append(opts, cache.WithEncryption(*config.Config.CacheConfig.EncryptionKey))
		}

		cache, err := cache.New(config.Config.CacheConfig.CacheUrl, opts...)
		if err != nil {
			logger.Error("Failed to create cache", err)
			panic("failed to create cache")
		}

		c = cache
	}

	router := chi.NewMux()
	api := humachi.New(router, huma.DefaultConfig("paperfit API", "0.1.0"))

	AddRoutes(api, httpPDFprovider, c, &logger)

	if err := http.ListenAndServe("localhost:"+port, router); err != nil {
		panic("failed to start server")
	}

	logger.Info("listening on localhost:" + port)
}

func AddRoutes(api huma.API, pdfService service.PDFserviceGenerator, cache *cache.Cache, logger *logger.Logger) {
	opts := []projectprovider.ArchiveProjectProviderOptions{}

	if cache != nil {
		opts = append(opts, projectprovider.WithCache(*cache))
	}

	archiveProjectProvider, err := projectprovider.NewArchiveProjectProvider(logger, opts...)
	if err != nil {
		logger.Error("Failed to create archive provider", err)
		panic("failed to create archive provider")
	}

	htmlgen.RegisterOperation(api, "/pdf/generate/html", pdfService)
	archiveurlgen.RegisterOperation(api, "/pdf/generate/url", archiveProjectProvider, pdfService, cache)
	archivegen.RegisterOperation(api, "/pdf/generate/archive", pdfService, cache)
}
