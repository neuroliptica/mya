package main

import (
	"bytes"
	"fmt"
	pnglib "image/png"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"

	"github.com/bakape/thumbnailer/v2"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

const (
	FileDirectory  = "src"
	ThumbDirectory = "thumb"
	FileMaxSize    = 2e7

	ThumbHeight = 250
	ThumbWidth  = 250
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
	return hash(uuid.NewString()) + filepath.Ext(fname)
}

// https://www.garykessler.net/library/file_sigs.html
var signatures = map[string]func([]byte) bool{
	"image/jpeg": jpeg,
	"image/png":  png,
	"video/webm": webm,
	"video/mp4":  mp4,
}

type FSign struct {
	// Leading bytes offset.
	Offset int

	Leading  []byte
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
		Leading:  []byte("\xff\xd8"),
		Trailing: []byte("\xff\xd9"),
	}.CheckSign(f)
}

func png(f []byte) bool {
	return FSign{
		Leading: []byte("\x89\x50\x4e\x47\x0d\x0a\x1a\x0a"),
	}.CheckSign(f)
}

func mp4(f []byte) bool {
	return FSign{
		Offset:  4,
		Leading: []byte("\x66\x74\x79\x70\x69\x73\x6f\x6d"),
	}.CheckSign(f)
}

func webm(f []byte) bool {
	return FSign{
		Leading: []byte("\x1A\x45\xDF\xA3"),
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

// Saving thumbnails as pngs in it's own directory.
func (f *File) SaveThumb(dst *os.File) error {
	// todo(zvezdochka): check about fs descriptors limits.
	f.Thumb = fmt.Sprintf("%s/thumb_%s.png", ThumbDirectory, f.Name)
	fs, err := os.Create(f.Thumb)
	if err != nil {
		return err
	}
	defer fs.Close()

	opts := thumbnailer.Options{
		ThumbDims: thumbnailer.Dims{ThumbWidth, ThumbHeight},
	}
	src, th, err := thumbnailer.Process(dst, opts)
	if err != nil {
		return err
	}

	f.Height = src.Height
	f.Width = src.Width

	return pnglib.Encode(fs, th)
}

// Save file and it's thumbnail on disk.
func (f *File) Save(buf io.Reader) error {
	dst, err := os.Create(f.Path)
	if err != nil {
		return err
	}
	defer dst.Close()

	if _, err := io.Copy(dst, buf); err != nil {
		return err
	}

	return f.SaveThumb(dst)
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

	fn := genName(header.Filename)
	f := &File{
		Path: fmt.Sprintf(
			"%s/%s",
			FileDirectory,
			fn,
		),
		Name: fn,
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
