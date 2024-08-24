package main

import (
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

func migratePost() error {
	return db.AutoMigrate(&Post{})
}

func checkThread(board string, id uint) (*Post, error) {
	return checkRecord(&Post{}, "parent = 0 AND board = ? AND id = ?", board, id)
}

func (t Post) FormatTimestamp() string {
	return t.CreatedAt.Format("02/01/2006 15:04:05")
}

// Replace markdown tags with html tags.
func (t Post) RenderedText() template.HTML {
	return replaceMarkdown(t.Text)
}

// Check if current poster isn't banned.
func (p *Post) CheckBanned() error {
	b, err := checkRecord(&Ban{
		Hash: p.IpHash,
	})
	if err != nil {
		log.Error().Msg(err.Error())
		return nil
	}
	if b.hasExpired() {
		return nil
	}
	return fmt.Errorf("banned until %v for %s", b.Until, b.Reason)
}

// Check if post board exists.
func (p *Post) CheckBoard() error {
	_, err := checkRecord(&Board{
		Link: p.Board,
	})
	return err
}

// Check if post thread exists.
func (p *Post) CheckThread() error {
	if p.Parent == 0 {
		return nil
	}
	_, err := checkThread(p.Board, p.Parent)
	return err
}

// Check if post subject is valid.
func (p *Post) CheckSubject() error {
	l := len(p.Subject)
	if l == 0 && p.Parent == 0 {
		return ErrorEmptySubject
	}
	if l > 80 {
		p.Subject = p.Subject[:80]
	}
	return nil
}

// Check if post name is valid.
func (p *Post) CheckName() error {
	if p.Name == "" {
		p.Name = "Anonymous"
	}
	if len(p.Name) > 80 {
		return ErrorNameTooLong
	}
	return nil
}

// Validate post captcha.
func (p *Post) CheckCaptcha() error {
	valid := captchas.Check(p.CaptchaValue, p.CaptchaId)
	captchas.Delete(p.CaptchaId)
	log.Debug().Msgf(
		"captcha %s was deleted",
		p.CaptchaId,
	)
	if !valid {
		return ErrorInvalidCaptcha
	}
	return nil
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
		String("captcha_id", &post.CaptchaId).
		String("captcha_value", &post.CaptchaValue).
		Uint("parent", &post.Parent).
		Bool("sage", &post.Sage).
		BindError()

	if err != nil {
		return c.String(http.StatusBadRequest, err.Error())
	}

	post.IpHash = hash(c.RealIP())
	post.LastBump = time.Now()
	checks := Maybe{
		post.CheckBanned,
		post.CheckBoard,
		post.CheckThread,
		post.CheckSubject,
		post.CheckName,
		post.CheckCaptcha,
	}
	err = checks.Eval()
	if err != nil {
		return c.JSON(http.StatusBadRequest, Error{err.Error()})
	}

	result := db.Create(post)
	if result.Error != nil {
		e := Error{result.Error.Error()}
		log.Error().Msg(e.Error())
		return c.JSON(http.StatusBadRequest, e)
	}

	// Bump parent thread.
	if !post.Sage && post.Parent != 0 {
		err = bump(post.Parent)
		if err != nil {
			log.Error().Msg(err.Error())
			return c.JSON(http.StatusInternalServerError, ErrorBumpFailed)
		}
	}

	return c.JSON(http.StatusCreated, post)
}

func deletePost(c echo.Context) error {
	return nil
}
