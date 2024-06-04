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

func New() {
	config.Load()

	logger := logger.New()

	httpPDFprovider := pdfprovider.NewHTTPpdfProvider(config.AppConfig.PDFgeneratorEndpoint)

	cacher, err := cacher.New(config.AppConfig.CacheUrl, logger)
	if err != nil {
		logger.Error("Failed to create cacher", err)
		panic("failed to create cacher")
	}

	router := chi.NewMux()
	api := humachi.New(router, huma.DefaultConfig("pdfuse API", "1.0.0"))

	AddRoutes(api, httpPDFprovider, &cacher, &logger)

	http.ListenAndServe(":8888", router)
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
