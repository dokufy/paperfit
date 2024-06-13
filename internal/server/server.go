package server

import (
	"net/http"

	"github.com/atoncetti/combiner/config"
	"github.com/atoncetti/combiner/internal/cacher"
	"github.com/atoncetti/combiner/internal/logger"
	"github.com/atoncetti/combiner/internal/server/operation/archivegen"
	"github.com/atoncetti/combiner/internal/server/operation/htmlgen"
	"github.com/atoncetti/combiner/internal/service"
	"github.com/atoncetti/combiner/internal/service/pdfprovider"
	"github.com/atoncetti/combiner/internal/service/projectprovider"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
)

func New(port string) {
	config.Load()

	logger := logger.New()

	httpPDFprovider := pdfprovider.NewHTTPpdfProvider(config.Config.PDFgeneratorEndpoint)

	var c *cacher.Cacher
	if config.Config.CacheConfig != nil {
		opts := []cacher.CacherOptions{cacher.WithLogger(&logger)}

		if config.Config.CacheConfig.EncryptionKey != nil {
			opts = append(opts, cacher.WithEncryption(*config.Config.CacheConfig.EncryptionKey))
		}

		cacher, err := cacher.New(config.Config.CacheConfig.CacheUrl, opts...)
		if err != nil {
			logger.Error("Failed to create cacher", err)
			panic("failed to create cacher")
		}

		c = cacher
	}

	router := chi.NewMux()
	api := humachi.New(router, huma.DefaultConfig("docfusion API", "0.1.0"))

	AddRoutes(api, httpPDFprovider, c, &logger)

	if err := http.ListenAndServe("localhost:"+port, router); err != nil {
		panic("failed to start server")
	}
}

func AddRoutes(api huma.API, pdfService service.PDFserviceProvider, cacher *cacher.Cacher, logger *logger.Logger) {
	opts := []projectprovider.ArchiveProjectProviderOptions{}

	if cacher != nil {
		opts = append(opts, projectprovider.WithCache(*cacher))
	}

	archiveProjectProvider, err := projectprovider.NewArchiveProjectProvider(logger, opts...)
	if err != nil {
		logger.Error("Failed to create archive provider", err)
		panic("failed to create archive provider")
	}

	htmlgen.RegisterOperation(api, "/pdf/generate/html", pdfService)
	archivegen.RegisterOperation(api, "/pdf/generate/archive", archiveProjectProvider, pdfService, cacher)
}
