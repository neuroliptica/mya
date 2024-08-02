package main

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/labstack/echo/v4"
)

type MainView struct {
	Header []Board
}

type ThreadView struct {
	OP      Post
	Replies []Post
}

type BoardView struct {
	CurrentBoard Board
	Header       []Board
	Threads      []ThreadView
}

func serveMain(c echo.Context) error {
	view := new(strings.Builder)
	// Get all boards to display in header.
	var boards []Board
	// todo: cache to avoid requests to db.
	err := get(&boards)
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}
	err = templates[MainTemplate].
		Execute(view, MainView{
			Header: boards,
		})
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}

	return c.HTML(http.StatusOK, view.String())
}

func serveBoard(c echo.Context) error {
	b := c.Param("board")
	page, err := strconv.Atoi(c.QueryParam("page"))
	if err != nil || page < 0 {
		page = 0
	}
	board, err := checkRecord(&Board{Link: b})
	if err != nil {
		return c.String(http.StatusBadRequest, err.Error())
	}
	// todo: sort by last bump, get [10*page:10*(page+1)]
	var posts []Post
	err = get(&posts, "board = ? AND parent = ?", board.Link, 0)
	if err != nil {
		return c.String(http.StatusBadRequest, err.Error())
	}
	// todo: cache to avoid requests to db.
	var boards []Board
	err = get(&boards)
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}

	bv := BoardView{
		Header:       boards,
		CurrentBoard: *board,
	}
	// Create view entry for every thread on board.
	for i := range posts {
		bv.Threads = append(bv.Threads, ThreadView{
			OP: posts[i],
			// Replies: ...
		})
	}
	// Render board page.
	view := new(strings.Builder)
	err = templates[BoardTemplate].Execute(view, bv)
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}

	return c.HTML(http.StatusOK, view.String())
}
