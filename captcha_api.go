package main

import (
	"bytes"
	"net/http"

	"github.com/labstack/echo/v4"
)

type CaptchaId struct {
	Id string `json:"id"`
	// Expires time.Time
}

type CaptchaError struct {
	Error string `json:"error"`
}

// GET /api/captcha/new
func newCaptcha(c echo.Context) error {
	// todo: generate random value
	id, err := captchas.Create()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, CaptchaError{
			Error: err.Error(),
		})
	}
	return c.JSON(http.StatusCreated, CaptchaId{
		Id: id,
	})
}

// GET /api/captcha/get?id={id}
func getCaptcha(c echo.Context) error {
	id := c.QueryParam("id")
	if id == "" {
		return c.JSON(http.StatusBadRequest, CaptchaError{
			Error: "no id provided",
		})
	}
	img, err := captchas.GetImage(id)
	if err != nil {
		return c.JSON(http.StatusBadRequest, CaptchaError{
			Error: err.Error(),
		})
	}
	r := bytes.NewReader(img)

	return c.Stream(http.StatusFound, "image/png", r)
}
