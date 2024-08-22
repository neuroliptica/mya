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

// Top sticked navbar.
type BoardsInfo struct {
	Header       []Board
	CurrentBoard Board
}

func serveMain(c echo.Context) error {
	view := new(strings.Builder)
	var boards []Board
	serve := Maybe{
		func() error {
			// get boards for header.
			// todo: cache to avoid requests to db.
			return get(&boards)
		},
		func() error {
			return templates[MainTemplate].
				Execute(view, MainView{Header: boards})
		},
	}
	err := serve.Eval()
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}

	return c.HTML(http.StatusOK, view.String())
}

type Paging struct {
	Current uint
	Pages   []uint
}

type BoardView struct {
	BoardsInfo
	Threads []ThreadView

	Pages Paging
}

func (b BoardView) LastId() int {
	return len(b.Threads) - 1
}

func serveBoard(c echo.Context) error {
	b := c.Param("board")
	page, err := strconv.Atoi(c.QueryParam("page"))
	if err != nil || page < 0 {
		page = 0
	}

	var (
		board  *Board
		posts  []Post
		boards []Board
	)

	initialization := Maybe{
		// Check if board exists.
		func() (err error) {
			board, err = checkRecord(&Board{Link: b})
			return err
		},
		// Get all threads for current board.
		func() error {
			return get(&posts, "board = ? AND parent = 0", board.Link)
		},
		// Get all boards for header.
		// todo: cache to avoid requests to db.
		func() error {
			return get(&boards)
		},
	}
	err = initialization.Eval()
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

	bv := BoardView{
		BoardsInfo: BoardsInfo{
			Header:       boards,
			CurrentBoard: *board,
		},
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
			OP:      posts[i],
			Omitted: 0,
		}
		replies := Maybe{
			// Get last 3 replies.
			func() error {
				res := db.Where(&Post{
					Board:  board.Link,
					Parent: posts[i].ID,
				}).
					Order("id desc").
					Limit(3).
					Find(&tv.Replies)

				return res.Error
			},
			// Get total amount of replies.
			func() error {
				res := db.Model(&Post{}).
					Where("board = ? AND parent = ?", board.Link, tv.OP.ID).
					Count(&tv.Omitted)

				return res.Error
			},
		}
		err := replies.Eval()
		if err != nil {
			return c.String(http.StatusInternalServerError, err.Error())
		}

		// Reverse the last 3 replies order.
		sort.Slice(tv.Replies, func(i, j int) bool {
			return tv.Replies[i].ID < tv.Replies[j].ID
		})

		tv.Omitted -= int64(len(tv.Replies))
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

type ThreadView struct {
	BoardsInfo

	OP      Post
	Replies []Post
	Omitted int64
}

func serveThread(c echo.Context) error {
	b := c.Param("board")

	var (
		id    int
		board *Board
		op    *Post
	)

	initialization := Maybe{
		func() (err error) {
			id, err = strconv.Atoi(c.Param("id"))
			return err
		},
		func() (err error) {
			board, err = checkRecord(&Board{Link: b})
			return err
		},
		func() (err error) {
			op, err = checkThread(board.Link, uint(id))
			return err
		},
	}

	err := initialization.Eval()
	if err != nil {
		return c.String(http.StatusBadRequest, err.Error())
	}

	tv := ThreadView{
		BoardsInfo: BoardsInfo{
			CurrentBoard: *board,
		},
		OP: *op,
	}

	requests := Maybe{
		func() (err error) {
			return get(&tv.Header)
		},
		func() (err error) {
			return get(&tv.Replies, "board = ? AND parent = ?", b, op.ID)
		},
	}

	err = requests.Eval()
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}

	view := new(strings.Builder)
	err = templates[ThreadTemplate].Execute(view, tv)
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}

	return c.HTML(http.StatusOK, view.String())
}
