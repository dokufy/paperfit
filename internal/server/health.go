package server

import (
	"net/http"

	"github.com/labstack/echo/v5"
)

func (s *Server) GetApplicationHealthy(c *echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) Healthy(c *echo.Context) error {
	return s.GetApplicationHealthy(c)
}
