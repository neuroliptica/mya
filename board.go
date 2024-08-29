package main

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog/log"
)

// Board model for single board entry.
type Board struct {
	DataModel

	Name string `gorm:"unique" json:"name"`
	Link string `gorm:"unique" json:"link"`
}

func migrateBoard() error {
	return db.AutoMigrate(&Board{})
}

// todo: check permission, etc.
func createBoard(c echo.Context) error {
	board := new(Board)
	if err := c.Bind(board); err != nil {
		log.Error().Msg(err.Error())
		return c.JSON(http.StatusBadRequest, ErrorBadRequest)
	}
	if board.Name == "" {
		return c.JSON(http.StatusBadRequest, ErrorEmptyName)
	}
	if board.Link == "" {
		return c.JSON(http.StatusBadRequest, ErrorEmptyLink)
	}

	br := Board{
		Name: board.Name,
		Link: board.Link,
	}
	result := db.Create(&br)
	if result.Error != nil {
		log.Error().Msg(result.Error.Error())
		return c.JSON(http.StatusBadRequest, ErrorCreateFailed)
	}

	return c.JSON(http.StatusOK, &br)
}

func getBoards(c echo.Context) error {
	var boards []Board
	err := get(&boards)
	if err != nil {
		log.Error().Msg(err.Error())
		return c.JSON(http.StatusBadRequest, ErrorGetFailed)
	}

	return c.JSON(http.StatusOK, boards)
}
