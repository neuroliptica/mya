package main

import (
	"bytes"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog/log"
)

type CaptchaId struct {
	Id string `json:"id"`
	// Expires time.Time
}

// GET /api/captcha/new
func newCaptcha(c echo.Context) error {
	id, err := captchas.Create()
	if err != nil {
		log.Error().Msg(err.Error())
		return c.JSON(http.StatusInternalServerError, ErrorNewCaptcha)
	}
	return c.JSON(http.StatusCreated, CaptchaId{
		Id: id,
	})
}

// GET /api/captcha/get?id={id}
func getCaptcha(c echo.Context) error {
	id := c.QueryParam("id")
	if id == "" {
		return c.JSON(http.StatusBadRequest, ErrorNoCaptchaId)
	}
	img, err := captchas.GetImage(id)
	if err != nil {
		log.Debug().Msg(err.Error())
		return c.JSON(http.StatusBadRequest, ErrorInvalidCaptchaId)
	}
	r := bytes.NewReader(img)

	return c.Stream(http.StatusFound, "image/png", r)
}
