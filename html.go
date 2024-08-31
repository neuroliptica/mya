package main

import (
	"math"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
)

const BoardViewReplies = 3

// Top sticked navbar.
type NavbarInfo struct {
	Navbar       []Board
	CurrentBoard Board
}

// Get all avaible boards from db.
func (nav *NavbarInfo) SetNavbar(tx *gorm.DB) error {
	return tx.Find(&nav.Navbar).Error
}

func (nav *NavbarInfo) SetCurrentBoard(tx *gorm.DB, link string) error {
	return tx.Where(&Board{Link: link}).
		First(&nav.CurrentBoard).
		Error
}

func (nav *NavbarInfo) SetHeader(link string) error {
	if err := nav.SetCurrentBoard(db, link); err != nil {
		return err
	}
	return nav.SetNavbar(db)
}

// Embed in every view with form.
type FormInfo struct {
	FormReply int
}

type MainView struct {
	NavbarInfo
}

func serveMain(c echo.Context) error {
	view := new(strings.Builder)
	mv := MainView{}

	init := Maybe{
		func() error {
			return mv.SetNavbar(db)
		},
		func() error {
			return templates.ExecuteTemplate(view, "main_page.tmpl", mv)
		},
	}

	if err := init.Eval(); err != nil {
		log.Error().Msg(err.Error())
		return c.JSON(http.StatusInternalServerError, Error{err.Error()})
	}

	return c.HTML(http.StatusOK, view.String())
}

type Paging struct {
	Current uint
	Total   []uint
}

type BoardView struct {
	NavbarInfo
	FormInfo

	Threads []ThreadView
	Pages   Paging
}

func (b BoardView) LastId() int {
	return len(b.Threads) - 1
}

func serveBoard(c echo.Context) error {
	board := c.Param("board")
	page, err := strconv.Atoi(c.QueryParam("page"))
	if err != nil || page < 0 {
		page = 0
	}

	bv := BoardView{}
	if err := bv.SetHeader(board); err != nil {
		log.Error().Msg(err.Error())
		return c.JSON(http.StatusBadRequest, Error{err.Error()})
	}

	posts := []Post{}
	if err := db.Find(&posts, "board = ? AND parent = 0", board).Error; err != nil {
		log.Error().Msg(err.Error())
		return c.JSON(http.StatusBadRequest, Error{err.Error()})
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
		return c.JSON(http.StatusBadRequest, Error{"no such page"})
	}

	posts = posts[lb:rb]
	bv.Pages.Current = uint(page)
	bv.Pages.Total = make([]uint, pages)

	// Generate total pages indexes.
	for i := range bv.Pages.Total {
		bv.Pages.Total[i] = uint(i)
	}

	// Create view entry for every thread on board.
	for i := range posts {
		tv := ThreadView{
			OP:      posts[i],
			Omitted: 0,
		}
		if err := tv.BoardEntry(); err != nil {
			log.Error().Msg(err.Error())
			return c.JSON(http.StatusBadRequest, Error{err.Error()})
		}

		tv.Omitted -= int64(len(tv.Replies))
		bv.Threads = append(bv.Threads, tv)
	}

	// Render board page.
	view := new(strings.Builder)

	if err := templates.ExecuteTemplate(view, "board.tmpl", bv); err != nil {
		log.Error().Msg(err.Error())
		return c.JSON(http.StatusInternalServerError, Error{err.Error()})
	}

	return c.HTML(http.StatusOK, view.String())
}

type ThreadView struct {
	NavbarInfo
	FormInfo

	OP      Post
	Replies []Post
	Omitted int64
}

// OP field should be set when called.
func (t *ThreadView) BoardEntry() error {
	// Get last 3 replies.
	err := db.Where(&Post{
		Board:  t.OP.Board,
		Parent: t.OP.ID,
	}).
		Order("id desc").
		Limit(BoardViewReplies).
		Find(&t.Replies).
		Error

	if err != nil {
		return err
	}

	// Reverse it's order.
	sort.Slice(t.Replies, func(i, j int) bool {
		return t.Replies[i].ID < t.Replies[j].ID
	})

	// Get total amount of replies.
	return db.Model(&Post{}).
		Where("board = ? AND parent = ?", t.OP.Board, t.OP.ID).
		Count(&t.Omitted).
		Error
}

func (t *ThreadView) ThreadEntry(board string, id uint) error {
	t.FormReply = int(id)
	// Set all replies.
	if err := db.Find(&t.Replies, "board = ? AND parent = ?", board, id).Error; err != nil {
		return err
	}

	// Set op post itself.
	return db.Where(&Post{}).
		First(&t.OP, "parent = 0 AND board = ? AND id = ?", board, id).
		Error
}

func serveThread(c echo.Context) error {
	board := c.Param("board")
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, Error{"invalid thread id"})
	}

	tv := ThreadView{}
	if err := tv.SetHeader(board); err != nil {
		return c.JSON(http.StatusBadRequest, Error{err.Error()})
	}

	if err := tv.ThreadEntry(board, uint(id)); err != nil {
		log.Error().Msg(err.Error())
		return c.JSON(http.StatusInternalServerError, Error{err.Error()})
	}

	view := new(strings.Builder)

	if err := templates.ExecuteTemplate(view, "thread.tmpl", tv); err != nil {
		log.Error().Msg(err.Error())
		return c.JSON(http.StatusInternalServerError, Error{err.Error()})
	}

	return c.HTML(http.StatusOK, view.String())
}
