package server

import (
	"bytes"
	"net/http"

	"github.com/labstack/echo/v5"
)

func (s *Server) HandleMCP(c *echo.Context) error {
	var raw []byte
	if c.Request().Body != nil {
		buf := new(bytes.Buffer)
		_, _ = buf.ReadFrom(c.Request().Body)
		raw = buf.Bytes()
	}
	respBytes, isNotif, err := s.mcpServer.HandleRequest(c.Request().Context(), raw)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, err.Error())
	}
	if isNotif || respBytes == nil {
		return c.NoContent(http.StatusAccepted)
	}
	return c.Blob(http.StatusOK, "application/json", respBytes)
}

func (s *Server) HandleMCPGet(c *echo.Context) error {
	// SSE not implemented; return info
	return c.JSON(http.StatusOK, map[string]string{"status": "mcp endpoint ready, use POST"})
}
