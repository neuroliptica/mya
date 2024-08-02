package main

import (
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
)

type void struct{}

func serveMain(c echo.Context) error {
	view := new(strings.Builder)
	// get all boards to display in header.
	var boards []Board
	result := db.Find(&boards)
	if result.Error != nil {
		msg := result.Error.Error()
		return c.String(http.StatusInternalServerError, msg)
	}

	err := templates["main_page"].Execute(view, boards)
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}

	return c.HTML(http.StatusOK, view.String())
}
