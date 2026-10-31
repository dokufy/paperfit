package server

import (
	"context"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"

	"github.com/dokufy/paperfit/config"
	"github.com/dokufy/paperfit/internal/apigen"
	"github.com/dokufy/paperfit/internal/cache"
	"github.com/dokufy/paperfit/internal/logger"
	"github.com/dokufy/paperfit/internal/mcp"
	"github.com/dokufy/paperfit/internal/service/pdfgenerator"
	"github.com/dokufy/paperfit/internal/service/projectprovider"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
)

type Server struct {
	cacheInstance          *cache.Cache
	archiveProjectProvider *projectprovider.ArchiveProjectProvider
	pdfService             *pdfgenerator.GotembergGenerator
	mcpServer              *mcp.Server
	echoInstance           *echo.Echo
	logger                 logger.Logger
	ctx                    context.Context
	cancel                 context.CancelFunc
	done                   chan struct{}
	started                atomic.Bool
}

func New() (*Server, error) {
	config.Load()
	logger.Initialize()

	e := echo.New()

	var c *cache.Cache
	if config.AppConfig.CacheConfig != nil {
		opts := []cache.CacherOptions{}

		if config.AppConfig.CacheConfig.EncryptionKey != nil {
			opts = append(opts, cache.WithEncryption(*config.AppConfig.CacheConfig.EncryptionKey))
		}

		cache, err := cache.New(config.AppConfig.CacheConfig.CacheUrl, opts...)
		if err != nil {
			logger.Log.Error("Failed to create cache", err)
			panic("failed to create cache")
		}

		c = cache
	}

	opts := []projectprovider.ArchiveProjectProviderOptions{}

	if c != nil {
		opts = append(opts, projectprovider.WithCache(*c))
	}

	archiveProjectProvider, err := projectprovider.NewArchiveProjectProvider(&logger.Log, opts...)
	if err != nil {
		logger.Log.Error("Failed to create archive provider", err)
		panic("failed to create archive provider")
	}

	server := &Server{
		archiveProjectProvider: archiveProjectProvider,
		pdfService:             pdfgenerator.New(config.AppConfig.PDFgeneratorEndpoint),
		cacheInstance:          c,
		echoInstance:           e,
		logger:                 logger.Log,
	}
	server.mcpServer = mcp.New(server.pdfService)

	server.ctx, server.cancel = context.WithCancel(context.Background())
	server.done = make(chan struct{})

	swaggerSpec, err := apigen.GetSpec()
	if err != nil {
		logger.Log.Error("Failed to get swagger spec", err)
		panic("failed to get swagger spec")
	}
	swaggerSpec.Servers = nil

	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: config.AppConfig.WhitelistOrigins,
	}))
	e.Use(middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogURI:       true,
		LogStatus:    true,
		LogLatency:   true,
		LogRequestID: true,
		LogMethod:    true,
		LogValuesFunc: func(_ *echo.Context, v middleware.RequestLoggerValues) error {
			server.logger.Info("Request", "method", v.Method, "uri", v.URI, "status", v.Status,
				"id", v.RequestID, "error", v.Error, "latency", v.Latency)
			return nil
		},
	}))
	e.Use(middleware.RateLimiter(middleware.NewRateLimiterMemoryStore(
		config.AppConfig.MaxRequestsPerSecond,
	)))
	e.Use(middleware.RequestID())
	openapi3filter.RegisterBodyDecoder("multipart/form-data", openapi3filter.FileBodyDecoder)

	apigen.RegisterHandlers(e, server)

	e.POST("/mcp", server.HandleMCP)
	e.GET("/mcp", server.HandleMCPGet)
	e.GET("/llms.txt", server.HandleLLMs)

	logger.Log.Info("Server initialized")

	return server, nil
}

func (s *Server) Start(port string) error {
	s.started.Store(true)
	defer close(s.done)

	ctx, stop := signal.NotifyContext(s.ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	startConfig := echo.StartConfig{
		Address:    ":" + port,
		HideBanner: true,
	}

	return startConfig.Start(ctx, s.echoInstance)
}

func (s *Server) Stop() {
	s.logger.Info("Stopping API server gracefully...")

	s.cancel()

	if s.started.Load() {
		<-s.done
	}

	s.logger.Info("Stopped API server")
}
