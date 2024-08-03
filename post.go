package main

import (
	"errors"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
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

func migratePost() error {
	return db.AutoMigrate(&Post{})
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

	post.IpHash = c.RealIP()
	post.LastBump = time.Now()
	if post.Name == "" {
		post.Name = "Anonymous"
	}
	checks := Maybe{
		// Check if ip not banned.
		func() error {
			return nil
		},
		// Check if board exists.
		func() error {
			_, err := checkRecord(&Board{
				Link: post.Board,
			})
			return err
		},
		// Check if thread exists and not closed.
		func() error {
			return nil
		},
		// Check if post subject is valid.
		func() error {
			l := len(post.Subject)
			if l == 0 {
				return errors.New("empty subject")
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
