package main

import (
	"bytes"
	"errors"
	"io"
	"mime/multipart"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

const (
	fdir    = "src"
	maxsize = 2e7
)

type File struct {
	ID uint `json:"id"`

	Path string `json:"path" gorm:"unique"`
	Name string `json:"name"`
	Size int64  `json:"size"`

	Width  uint `json:"width"`
	Height uint `json:"height"`

	CreatedAt time.Time `json:"-"`
	UpdatedAt time.Time `json:"-"`
}

// Files assosiated with post are encoded as json string.
// So we can store multiple files for single post record.
type FilesJson struct {
	Ids []int `json:"ids"`
}

func migrateFile() error {
	return db.AutoMigrate(&File{})
}

// Read files content from form, then save it on disk.
func processFiles(ctx echo.Context) ([]File, error) {
	form, err := ctx.MultipartForm()
	if err != nil {
		return nil, err
	}
	res := make([]File, 0)
	fs := form.File["files"]
	for i := range fs {
		f, err := uploadFile(fs[i])
		if err != nil {
			return nil, err
		}
		res = append(res, *f)
	}
	return res, nil
}

func genName(fname string) string {
	// todo(zvezdochka): normal filenames generation.
	p := strings.Split(fname, ".")
	ext := ""
	if len(p) != 0 {
		ext = "." + p[len(p)-1]
	}
	return hash(uuid.NewString()) + ext
}

// Predict file's format by it's header sign.
// https://www.garykessler.net/library/file_sigs.html
var signatures = map[string]func([]byte) bool{
	"jpeg": jpeg,
	"png":  png,
	"webm": webm,
	"mp4":  mp4,
}

type FSign struct {
	// Leading bytes offset.
	Offset int
	// Leading signature.
	Leading []byte
	// Trailing signature.
	Trailing []byte
}

func (s FSign) CheckSign(f []byte) bool {
	if len(f) < s.Offset+len(s.Trailing)+len(s.Trailing) {
		return false
	}
	p := true
	for i := range s.Leading {
		p = p && (f[i+s.Offset] == s.Leading[i])
	}
	for i := range s.Trailing {
		p = p && (f[len(f)-i-1] == s.Trailing[len(s.Trailing)-i-1])
	}
	return p
}

func jpeg(f []byte) bool {
	return FSign{
		Leading:  []byte{0xff, 0xd8},
		Trailing: []byte{0xff, 0xd9},
	}.CheckSign(f)
}

func png(f []byte) bool {
	return FSign{
		Leading: []byte{
			0x89, 0x50, 0x4e, 0x47,
			0x0d, 0x0a, 0x1a, 0x0a,
		},
	}.CheckSign(f)
}

func mp4(f []byte) bool {
	return FSign{
		Offset: 4,
		Leading: []byte{
			0x66, 0x74, 0x79, 0x70,
			0x69, 0x73, 0x6f, 0x6d,
		},
	}.CheckSign(f)
}

func webm(f []byte) bool {
	return FSign{
		Leading: []byte{
			0x1A, 0x45, 0xDF, 0xA3,
		},
	}.CheckSign(f)
}

// Save flie to `src` dir if it is fits conditions.
func uploadFile(header *multipart.FileHeader) (*File, error) {
	// todo(zvezdochka): redeclare maxsize const in config.
	if header.Size > maxsize {
		return nil, errors.New("file is too large")
	}
	src, err := header.Open()
	if err != nil {
		return nil, err
	}
	defer src.Close()

	buf := new(bytes.Buffer)
	if _, err = io.Copy(buf, src); err != nil {
		return nil, err
	}

	valid := false
	for _, sign := range signatures {
		if sign(buf.Bytes()) {
			valid = true
			break
		}
	}
	if !valid {
		return nil, errors.New("invalid file signature")
	}
	e := &File{
		Path: fdir + "/" + genName(header.Filename),
		Name: header.Filename,
		Size: header.Size,
	}

	dst, err := os.Create(e.Path)
	if err != nil {
		return nil, err
	}
	defer dst.Close()

	if _, err = io.Copy(dst, buf); err != nil {
		return nil, err
	}

	return e, nil
}
