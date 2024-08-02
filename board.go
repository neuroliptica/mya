package main

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
)

// Board model for single board entry.
type Board struct {
	ID uint `json:"id"`

	Name string `gorm:"unique" json:"name"`
	Link string `gorm:"unique" json:"link"`

	CreatedAt time.Time
	UpdatedAt time.Time
}

func migrateBoard() error {
	return db.AutoMigrate(&Board{})
}

// todo: check permission, etc.
func createBoard(c echo.Context) error {
	board := new(Board)
	if err := c.Bind(board); err != nil {
		return c.String(http.StatusBadRequest, "bad request")
	}
	if board.Name == "" || board.Link == "" {
		return c.String(http.StatusBadRequest, "empty name or link")
	}
	br := Board{
		Name: board.Name,
		Link: board.Link,
	}

	result := db.Create(&br)
	if result.Error != nil {
		// todo: not to expose internal db error.
		return c.String(http.StatusBadRequest, result.Error.Error())
	}

	return c.JSON(http.StatusOK, &br)
}

func getBoards(c echo.Context) error {
	var boards []Board
	result := db.Find(&boards)
	if result.Error != nil {
		// todo: not to expose internal db error.
		return c.String(http.StatusBadRequest, result.Error.Error())
	}

	return c.JSON(http.StatusOK, boards)
}
