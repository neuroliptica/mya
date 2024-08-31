package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
)

type Post struct {
	DataModel

	Name    string `json:"name"`
	Subject string `json:"subject"`
	Text    string `json:"text"`
	Sage    bool   `json:"sage"`
	Board   string `json:"board"`
	Parent  uint   `json:"parent"`

	LastBump time.Time `json:"last_bump"`
	IpHash   string    `json:"-"`

	CaptchaId    string `json:"-" gorm:"-:all"`
	CaptchaValue string `json:"-" gorm:"-:all"`

	FilesJson string `json:"-"`
	Files     []File `gorm:"-:all"`
}

func migratePost() error {
	return db.AutoMigrate(&Post{})
}

func (t Post) FormatTimestamp() string {
	return t.CreatedAt.Format("02/01/2006 15:04:05")
}

func (t Post) HasFiles() bool {
	return len(t.Files) != 0
}

// Replace markdown tags with html tags.
func (t Post) RenderedText() template.HTML {
	return replaceMarkdown(t.Text)
}

func (p *Post) CheckBanned() error {
	var b Ban
	err := db.Where(&Ban{Hash: p.IpHash}).First(&b).Error
	switch err {
	case nil:
		break
	case gorm.ErrRecordNotFound:
		return nil
	default:
		return err
	}
	if b.HasExpired() {
		return nil
	}
	return fmt.Errorf(
		"banned until %v for %s",
		b.Until,
		b.Reason,
	)
}

func (p *Post) CheckBoard() error {
	var b Board
	err := db.Where(&Board{Link: p.Board}).First(&b).Error
	switch err {
	case gorm.ErrRecordNotFound:
		return errors.New("invalid board")
	default:
		return err
	}
}

func (p *Post) CheckThread() error {
	if p.Parent == 0 {
		return nil
	}
	var f Post
	err := db.Where(&Post{}).
		First(&f, "parent = 0 AND board = ? AND id = ?", p.Board, p.Parent).
		Error

	switch err {
	case gorm.ErrRecordNotFound:
		return errors.New("invalid thread id")
	default:
		return err
	}
}

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

func (p *Post) GetFiles(tx *gorm.DB) ([]File, error) {
	var fs FilesJson
	if err := json.Unmarshal([]byte(p.FilesJson), &fs); err != nil {
		return nil, err
	}
	if len(fs.Ids) == 0 {
		log.Debug().Msgf("(*Post).FilesJson=%s", p.FilesJson)
		return nil, nil
	}

	var files []File
	res := tx.Find(&files, fs.Ids)

	return files, res.Error
}

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
		// todo(zvezdochka):
		// manage files as kind of `transaction` also.
		// eg. if there is fail to create record in db,
		// then files should be removed from disk.
		// also: maybe use some /temp place?

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
