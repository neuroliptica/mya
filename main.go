package main

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

var (
	// global logger instance, can be used in any part.
	logger = NewLogger()
)

func main() {
	e := echo.New()
	e.Use(
		LoggingMiddleware,
		middleware.Recover(),
		middleware.CORS(),
	)

	// pages.
	e.GET("/", func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	})

	// rest api.
	api := e.Group("/api")
	api.GET("/ping", func(c echo.Context) error {
		return c.String(http.StatusOK, "pong")
	})

	logger.Info().Msg(e.Start(":3000").Error())
}
