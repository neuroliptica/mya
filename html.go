package main

import (
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
)

type void struct{}

func serveMain(c echo.Context) error {
	view := new(strings.Builder)
	err := templates["main_page"].Execute(view, void{})
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}

	return c.HTML(http.StatusOK, view.String())
}
