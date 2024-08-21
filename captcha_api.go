package main

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

type CaptchaId struct {
	Id string `json:"id"`
	// Expires time.Time
}

// GET /api/captcha/new
func newCaptcha(c echo.Context) error {
	// todo: generate random value
	value := "123"
	id := captchas.Create(value)

	return c.JSON(http.StatusCreated, CaptchaId{
		Id: id,
	})
}

type CaptchaResponse struct {
	Id     string `json:"id"`
	Base64 string `json:"base64_image"`
}

// GET /api/captcha/get?id={id}
func getCaptcha(c echo.Context) error {
	id := c.QueryParam("id")
	if id == "" {
		return c.String(http.StatusBadRequest, "no id provided")
	}
	img, err := captchas.GetImage(id)
	if err != nil {
		return c.String(http.StatusBadRequest, err.Error())
	}

	return c.JSON(http.StatusFound, CaptchaResponse{
		Id:     id,
		Base64: string(img),
	})
}
