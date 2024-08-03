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
		// Check if thread exists.
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

	// If not sage and parent != 0, update last bump for parent.
	result := db.Create(post)
	if result.Error != nil {
		return c.String(http.StatusBadRequest, err.Error())
	}

	return c.JSON(http.StatusCreated, post)
}

func deletePost(c echo.Context) error {
	return nil
}
