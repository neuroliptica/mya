package main

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

var (
	// global logger instance.
	logger = NewLogger()
)

func main() {
	initialization := Maybe{
		compileTemplates,
		func() error {
			return assignSqlite("dev.db")
		},
		migrate,
	}

	err := initialization.Eval()
	if err != nil {
		logger.Fatal().Msg(err.Error())
	}

	e := echo.New()
	e.Use(
		LoggingMiddleware,
		middleware.Recover(),
		middleware.CORS(),
	)

	// serve static files.
	e.Static("/static", "static")
	e.GET("/favicon.ico", func(c echo.Context) error {
		return c.Redirect(http.StatusPermanentRedirect, "/static/favicon.ico")
	})

	// serve pages.
	e.GET("/", serveMain)
	e.GET("/:board", serveBoard)
	e.GET("/:board/:id", func(c echo.Context) error {
		return c.String(http.StatusOK, c.Param("id"))
	})

	// rest api.
	api := e.Group("/api")
	api.GET("/get_boards", getBoards)
	api.POST("/post", createPost)
	// moder.
	api.POST("/create_board", createBoard)

	logger.Info().Msg(e.Start(":3000").Error())
}
