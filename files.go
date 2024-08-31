package main

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"strings"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

const (
	FileDirectory = "src"
	FileMaxSize   = 2e7

	UndefinedSign = ""
)

// Files assosiated with post are encoded as json string.
type FilesJson struct {
	Ids []int `json:"ids"`
}

type File struct {
	DataModel

	// Relative path for file itself.
	Path string `json:"path" gorm:"unique"`
	// Relative path for it's jpeg thumbnail.
	Thumb string `json:"thumb" gorm:"unique"`

	Name string `json:"name"`
	Size int64  `json:"size"`
	Sign string `json:"sign"`

	Width  uint `json:"width"`
	Height uint `json:"height"`
}

func migrateFile() error {
	return db.AutoMigrate(&File{})
}

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

func (f *File) SetSignature(buf []byte) error {
	for key, sign := range signatures {
		if sign(buf) {
			f.Sign = key
			return nil
		}
	}
	return ErrorInvalidSignature
}

func (f *File) Save(buf io.Reader) error {
	dst, err := os.Create(f.Path)
	if err != nil {
		return err
	}
	defer dst.Close()

	_, err = io.Copy(dst, buf)
	return err
}

func uploadFile(header *multipart.FileHeader) (*File, error) {
	// todo(zvezdochka): redeclare maxsize const in config.
	if header.Size > FileMaxSize {
		return nil, ErrorTooLarge
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

	f := &File{
		Path: fmt.Sprintf(
			"%s/%s",
			FileDirectory,
			genName(header.Filename),
		),
		Name: header.Filename,
		Size: header.Size,
	}

	if err := f.SetSignature(buf.Bytes()); err != nil {
		return nil, err
	}

	if err := f.Save(buf); err != nil {
		return nil, err
	}

	return f, nil
}
