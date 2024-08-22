package main

import (
	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog/log"
)

func LoggingMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		log.Info().Fields(map[string]interface{}{
			"method": c.Request().Method,
			"url":    c.Request().URL.Path,
			"query":  c.Request().URL.RawQuery,
		}).Msg("request")

		err := next(c)
		if err != nil {
			log.Error().Fields(map[string]interface{}{
				"error": err.Error(),
			}).Msg("response")
			return err
		}
		return nil
	}
}
