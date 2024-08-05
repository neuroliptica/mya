package main

import (
	"math"
	"net/http"
	"sort"
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

	// Total replies amount minus last three.
	Missed int64
}

type Paging struct {
	Current uint
	Pages   []uint
}

type BoardView struct {
	CurrentBoard Board
	Header       []Board
	Threads      []ThreadView

	Pages Paging
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

	// Check if board exists.
	board, err := checkRecord(&Board{Link: b})
	if err != nil {
		return c.String(http.StatusBadRequest, err.Error())
	}

	// Get all threads for current board.
	var posts []Post
	err = get(&posts, "board = ? AND parent = ?", board.Link, 0)
	if err != nil {
		return c.String(http.StatusBadRequest, err.Error())
	}

	// Sorting by last bump.
	sort.Slice(posts, func(i, j int) bool {
		return posts[i].LastBump.After(posts[j].LastBump)
	})

	// Get total pages on board.
	pages := len(posts)/10 + 1
	if len(posts)%10 == 0 && len(posts) != 0 {
		pages--
	}

	// Get threads with offset for current page.
	lb := page * 10
	rb := (page + 1) * 10
	rb = int(math.Min(float64(rb), float64(len(posts))))
	if rb < lb {
		// todo: redirect or normal error message.
		return c.String(http.StatusBadRequest, "no such page")
	}
	posts = posts[lb:rb]

	// Get all boards for header.
	// todo: cache to avoid requests to db.
	var boards []Board
	err = get(&boards)
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}

	bv := BoardView{
		Header:       boards,
		CurrentBoard: *board,
		Pages: Paging{
			Current: uint(page),
			Pages:   make([]uint, pages),
		},
	}
	// Generate total pages indexes.
	for i := range bv.Pages.Pages {
		bv.Pages.Pages[i] = uint(i)
	}

	// Create view entry for every thread on board.
	for i := range posts {
		tv := ThreadView{
			OP:     posts[i],
			Missed: 0,
		}
		// Get last 3 replies.
		res := db.Where(&Post{
			Board:  board.Link,
			Parent: posts[i].ID,
		}).
			Order("id desc").
			Limit(3).
			Find(&tv.Replies)

		if res.Error != nil {
			return c.String(http.StatusInternalServerError, err.Error())
		}
		// Reverse the last 3 replies order.
		sort.Slice(tv.Replies, func(i, j int) bool {
			return tv.Replies[i].ID < tv.Replies[j].ID
		})

		// Get total amount of replies.
		res = db.Model(&Post{}).
			Where("board = ? AND parent = ?", board.Link, tv.OP.ID).
			Count(&tv.Missed)

		if res.Error != nil {
			return c.String(http.StatusInternalServerError, err.Error())
		}

		tv.Missed -= int64(len(tv.Replies))
		bv.Threads = append(bv.Threads, tv)
	}

	// Render board page.
	view := new(strings.Builder)
	err = templates[BoardTemplate].Execute(view, bv)
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}

	return c.HTML(http.StatusOK, view.String())
}
