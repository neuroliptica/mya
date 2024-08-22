package main

import (
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog/log"
)

type Post struct {
	ID uint `json:"id"`

	Name    string `json:"name"`
	Subject string `json:"subject"`
	Text    string `json:"text"`
	Sage    bool   `json:"sage"`
	Board   string `json:"board"`
	Parent  uint   `json:"parent"`

	LastBump  time.Time `json:"last_bump"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"-"`
	IpHash    string    `json:"-"`

	CaptchaId    string `json:"-" gorm:"-:all"`
	CaptchaValue string `json:"-" gorm:"-:all"`
}

func (t Post) FormatTimestamp() string {
	return t.CreatedAt.Format("02/01/2006 15:04:05")
}

// Replace markdown tags with html tags.
func (t Post) RenderedText() template.HTML {
	return replaceMarkdown(t.Text)
}

func migratePost() error {
	return db.AutoMigrate(&Post{})
}

func checkThread(board string, id uint) (*Post, error) {
	return checkRecord(&Post{}, "parent = 0 AND board = ? AND id = ?", board, id)
}

// Update lastbump field.
func bump(thread uint) error {
	result := db.Model(&Post{}).
		Where("id = ?", thread).
		Update("last_bump", time.Now())

	return result.Error
}

func createPost(c echo.Context) error {
	post := new(Post)
	err := echo.FormFieldBinder(c).
		String("name", &post.Name).
		String("subject", &post.Subject).
		String("text", &post.Text).
		String("board", &post.Board).
		Uint("parent", &post.Parent).
		Bool("sage", &post.Sage).
		BindError()

	if err != nil {
		return c.String(http.StatusBadRequest, err.Error())
	}

	post.IpHash = hash(c.RealIP())
	post.LastBump = time.Now()
	// todo(zvezdochka): check len for post.name
	if post.Name == "" {
		post.Name = "Anonymous"
	}
	checks := Maybe{
		// Check if ip not banned.
		func() error {
			b, err := checkRecord(&Ban{
				Hash: post.IpHash,
			})
			if err != nil {
				// We will get non-nil error also if there is no record.
				log.Debug().Msg(err.Error())
				return nil
			}
			if b.hasExpired() {
				return nil
			}
			return fmt.Errorf("banned until %v for %s", b.Until, b.Reason)
		},
		// Check if board exists.
		func() error {
			_, err := checkRecord(&Board{
				Link: post.Board,
			})
			return err
		},
		// Check if thread exists and not closed (todo).
		func() error {
			if post.Parent == 0 {
				return nil
			}
			_, err := checkThread(post.Board, post.Parent)
			return err
		},
		// Check if post subject is valid.
		func() error {
			l := len(post.Subject)
			if l == 0 && post.Parent == 0 {
				return errors.New("empty subject for thread")
			}
			if l > 80 {
				post.Subject = post.Subject[:80]
			}
			return nil
		},
	}
	err = checks.Eval()
	if err != nil {
		return c.String(http.StatusBadRequest, err.Error())
	}

	result := db.Create(post)
	if result.Error != nil {
		return c.String(http.StatusBadRequest, err.Error())
	}

	// Bump parent thread.
	if !post.Sage && post.Parent != 0 {
		err = bump(post.Parent)
		if err != nil {
			return c.String(http.StatusInternalServerError, err.Error())
		}
	}

	return c.JSON(http.StatusCreated, post)
}

func deletePost(c echo.Context) error {
	return nil
}
