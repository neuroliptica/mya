package main

import (
	"net/http"
	"os"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func init() {
	// Initialize default global logger.
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{
		Out:        os.Stderr,
		TimeFormat: time.DateTime,
	})

	// Debug mode by default.
	// todo(zvezdochka): explicit debug flag when start.
	zerolog.SetGlobalLevel(zerolog.DebugLevel)

	// Initialize other services.
	initialization := Maybe{
		compileTemplates,
		func() error {
			return assignSqlite("dev.db")
		},
		migrate,
		func() error {
			initCaptchas()
			return nil
		},
	}

	err := initialization.Eval()
	if err != nil {
		log.Fatal().Msg(err.Error())
	}
}

func main() {
	e := echo.New()
	e.Use(
		LoggingMiddleware,
		middleware.Recover(),
		middleware.CORS(),
	)

	// Serve static files.
	e.Static("/static", "static")
	e.Static("/src", "src")
	e.GET("/favicon.ico", func(c echo.Context) error {
		return c.Redirect(http.StatusPermanentRedirect, "/static/favicon.ico")
	})

	// Serve pages.
	e.GET("/", serveMain)
	e.GET("/:board", serveBoard)
	e.GET("/:board/:id", serveThread)

	// Rest api.
	api := e.Group("/api")
	api.GET("/get_boards", getBoards)
	api.POST("/post", createPost)

	// Captcha.
	api.GET("/captcha/new", newCaptcha)
	api.GET("/captcha/get", getCaptcha)

	// Admin routes.
	api.POST("/create_board", createBoard)
	// api.GET("/ban_yourself", bantest)

	log.Info().Msg(e.Start(":3000").Error())
}
