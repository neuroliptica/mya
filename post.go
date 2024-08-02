package main

import (
	"time"

	"github.com/labstack/echo/v4"
)

type Post struct {
	ID      uint   `json:"id"`
	Subject string `json:"subject"`
	Text    string `json:"text"`
	Sage    bool   `json:"sage"`

	Board    string    `json:"board"`
	Parent   uint      `json:"parent"`
	LastBump time.Time `json:"last_bump"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"-"`
	IpHash    string    `json:"-"`

	CaptchaId    string `json:"-" gorm:"-:all"`
	CaptchaValue string `json:"-" gorm:"-:all"`
}

func migratePost() error {
	return db.AutoMigrate(&Post{})
}

func createPost(c echo.Context) error {
	return nil
}

func deletePost(c echo.Context) error {
	return nil
}
