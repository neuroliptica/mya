package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog"
)

type Logger struct {
	zerolog.Logger
}

func NewLogger() Logger {
	output := zerolog.ConsoleWriter{
		Out:        os.Stdout,
		TimeFormat: time.DateTime,
		FormatLevel: func(i interface{}) string {
			return strings.ToUpper(fmt.Sprintf("| %-6s|", i))
		},
		FormatFieldName: func(i interface{}) string {
			return fmt.Sprintf("%s:", i)
		},
		FormatFieldValue: func(i interface{}) string {
			return fmt.Sprintf("%s", i)
		},
		FormatErrFieldName: func(i interface{}) string {
			return fmt.Sprintf("%s: ", i)
		},
	}

	return Logger{
		zerolog.New(output).
			With().
			Caller().
			Timestamp().
			Logger(),
	}
}

func LoggingMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		logger.Info().Fields(map[string]interface{}{
			"method": c.Request().Method,
			"url":    c.Request().URL.Path,
			"query":  c.Request().URL.RawQuery,
		}).Msg("request")

		err := next(c)
		if err != nil {
			logger.Error().Fields(map[string]interface{}{
				"error": err.Error(),
			}).Msg("response")
			return err
		}
		return nil
	}
}
