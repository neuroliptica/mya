package main

import (
	"bytes"
	"crypto/md5"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
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

type FileJson struct {
	Ids []string `json:"ids"`
}

func migrateFile() error {
	return db.AutoMigrate(&File{})
}

func genName(fname string) string {
	// todo(zvezdochka): normal filenames generation.
	p := strings.Split(fname, ".")
	ext := ""
	if len(p) != 0 {
		ext = "." + p[len(p)-1]
	}
	fid := uuid.NewString()
	return fmt.Sprintf(
		"%x%s",
		md5.Sum([]byte(fid)),
		ext,
	)
}

// Predict file's format by it's header sign.
// https://www.garykessler.net/library/file_sigs.html
var signatures = map[string]func([]byte) bool{
	"jpeg": jpeg,
	"png":  png,
	"webm": webm,
}

func jpeg(f []byte) bool {
	if len(f) < 4 {
		return false
	}
	// 2-byte jpg leading sign.
	fb := (f[0] == 0xff && f[1] == 0xd8)
	// 2-byte jpg trailing sign.
	lb := (f[len(f)-2] == 0xff && f[len(f)-1] == 0xd9)
	return fb && lb
}

func png(f []byte) bool {
	// 8-byte png leading sign.
	sign := []byte{
		0x89,
		0x50, 0x4e, 0x47,
		0x0d, 0x0a,
		0x1a,
		0x0a,
	}
	if len(f) < len(sign) {
		return false
	}
	p := true
	for i := range sign {
		p = p && (sign[i] == f[i])
	}
	return p
}

func webm(f []byte) bool {
	offset := 4
	sign := []byte{
		0x66, 0x74, 0x79, 0x70, 0x4D, 0x53, 0x4E, 0x56,
	}
	if len(f) < offset+len(sign) {
		return false
	}
	p := true
	for i := range sign {
		p = p && (f[offset+i] == sign[i])
	}
	return p
}

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
