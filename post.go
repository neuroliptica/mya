package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
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

	FilesJson string `json:"-"`
	Files     []File `gorm:"-:all"`
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

// Get files assigned with post.
func (p *Post) GetFiles(tx *gorm.DB) ([]File, error) {
	var fs FilesJson
	if err := json.Unmarshal([]byte(p.FilesJson), &fs); err != nil {
		return nil, err
	}
	if len(fs.Ids) == 0 {
		log.Debug().Msgf("(post.FilesJson)=%s", p.FilesJson)
		return nil, nil
	}

	var files []File
	res := tx.Find(&files, fs.Ids)

	return files, res.Error
}

// Assign files by it's ids with post.
func (p *Post) SetFiles(tx *gorm.DB, fs []File) error {
	ids := FilesJson{make([]int, 0)}
	for i := range fs {
		ids.Ids = append(ids.Ids, int(fs[i].ID))
	}
	m, err := json.Marshal(ids)
	if err != nil {
		return err
	}
	res := tx.Model(p).Updates(Post{
		FilesJson: string(m),
	})

	return res.Error
}

// Bump parent thread if neither sage nor creating.
func (p *Post) BumpParent(tx *gorm.DB) error {
	if p.Sage || p.Parent == 0 {
		return nil
	}
	res := tx.Model(&Post{}).
		Where("id = ?", p.Parent).
		Update("last_bump", time.Now())

	return res.Error
}

// Create post record using database transation.
func (p *Post) Create(ctx echo.Context) error {
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(p).Error; err != nil {
			return err
		}
		// Bump parent thread optionally.
		if err := p.BumpParent(tx); err != nil {
			return err
		}
		// Upload files on disk.
		fs, err := processFiles(ctx)
		if err != nil {
			return err
		}
		// Create files records in db.
		for i := range fs {
			if err := tx.Create(&fs[i]).Error; err != nil {
				return err
			}
		}
		// Assosiate files with current post.
		return p.SetFiles(tx, fs)
	})
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
		log.Error().Msg(err.Error())
		return c.JSON(http.StatusBadRequest, ErrorBadRequest)
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

	if err := checks.Eval(); err != nil {
		log.Debug().Msg(err.Error())
		return c.JSON(http.StatusBadRequest, Error{err.Error()})
	}

	if err := post.Create(c); err != nil {
		log.Error().Msg(err.Error())
		jsonerr := Error{
			E: fmt.Sprintf("transaction failed: %v", err),
		}
		return c.JSON(http.StatusInternalServerError, jsonerr)
	}

	return c.JSON(http.StatusCreated, post)
}
