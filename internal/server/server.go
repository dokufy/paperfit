package server

import (
	"encoding/json"
	"strconv"

	"github.com/dokufy/paperfit/config"
	"github.com/dokufy/paperfit/internal/apigen"
	"github.com/dokufy/paperfit/internal/cache"
	"github.com/dokufy/paperfit/internal/logger"
	"github.com/dokufy/paperfit/internal/service/pdfgenerator"
	"github.com/dokufy/paperfit/internal/service/projectprovider"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	// oapiMiddleware "github.com/oapi-codegen/echo-middleware"
	"golang.org/x/time/rate"
)

type Server struct {
	cacheInstance          *cache.Cache
	archiveProjectProvider *projectprovider.ArchiveProjectProvider
	pdfService             *pdfgenerator.GotembergGenerator
	echoInstance           *echo.Echo
	logger                 logger.Logger
}

func New() (*Server, error) {
	config.Load()
	logger.Initialize()

	e := echo.New()
	e.HideBanner = true

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

	swaggerSpec, err := apigen.GetSwagger()
	if err != nil {
		e.Logger.Error("Failed to get swagger spec", err)
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
		LogError:     true,
		LogMethod:    true,
		LogValuesFunc: func(_ echo.Context, v middleware.RequestLoggerValues) error {
			server.logger.Info("Request", "method", v.Method, "uri", v.URI, "status", v.Status,
				"id", v.RequestID, "error", v.Error, "latency", v.Latency)
			return nil
		},
	}))
	e.Use(middleware.RateLimiter(middleware.NewRateLimiterMemoryStore(
		rate.Limit(config.AppConfig.MaxRequestsPerSecond),
	)))
	e.Use(middleware.RequestID())
	openapi3filter.RegisterBodyDecoder("multipart/form-data", openapi3filter.FileBodyDecoder)
	// e.Use(oapiMiddleware.OapiRequestValidatorWithOptions(swaggerSpec, &oapiMiddleware.Options{
	// 	Options: openapi3filter.Options{
	// 		AuthenticationFunc: openapi3filter.NoopAuthenticationFunc,
	// 	},
	// }))

	apigen.RegisterHandlers(e, server)

	logger.Log.Info("Server initialized")

	return server, nil
}

func (s *Server) Start(port string) error {
	err := s.echoInstance.Start(":" + port)
	if err != nil {
		return err
	}

	return nil
}

func (s *Server) Stop() {
	s.logger.Info("Stopping API server gracefully...")
	err := s.echoInstance.Close()
	if err != nil {
		s.logger.Error("failed to stop API server", err)
		panic("failed to stop API server")
	}
	s.logger.Info("Stopped API server")
}

func unmarshallInput(jsonBody string) (apigen.Input, error) {
	var input apigen.Input

	var tmp map[string]string
	err := json.Unmarshal([]byte(jsonBody), &tmp)
	if err != nil {
		return input, err
	}

	input.Validate, err = strconv.ParseBool(tmp["validate"])
	if err != nil {
		return input, err
	}

	// input.Variables = tmp["variables"]

	return input, nil
}
