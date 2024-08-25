package main

import (
	"bytes"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	cgen "github.com/steambap/captcha"
)

const (
	CaptchaTimeLimit = 30 * time.Second
	CleanupTimeout   = 30 * time.Second
)

var (
	captchas *Storage
)

func initCaptchas() {
	captchas = NewStorage()
	go captchas.Cleanup()
}

type captcha struct {
	value string
	image []byte

	created time.Time
}

type Storage struct {
	Map map[string]*captcha
	Mu  *sync.RWMutex
}

func NewStorage() *Storage {
	return &Storage{
		Map: make(map[string]*captcha),
		Mu:  &sync.RWMutex{},
	}
}

// Get captcha by id, thread safe.
func (c *Storage) Get(id string) (*captcha, bool) {
	c.Mu.RLock()
	defer c.Mu.RUnlock()

	v, ok := c.Map[id]
	return v, ok
}

// Check if provided value is valid captcha value.
func (c *Storage) Check(value string, id string) bool {
	v, ok := c.Get(id)
	if !ok || v == nil {
		return false
	}
	if time.Since(v.created) >= CaptchaTimeLimit {
		return false
	}

	return v.value == value
}

// Get image for provided id.
func (c *Storage) GetImage(id string) ([]byte, error) {
	v, ok := c.Get(id)
	if !ok || v == nil {
		return nil, errors.New("invalid id or captcha expired")
	}

	return v.image, nil
}

// Create new captcha record and return it's id.
func (c *Storage) Create() (string, error) {
	//opt := func(c *cgen.Options) {
	//	c.BackgroundColor = color.White
	//}
	data, err := cgen.New(170, 60)
	if err != nil {
		return "", err
	}
	log.Debug().Msgf("captcha => %s", data.Text)

	buf := new(bytes.Buffer)
	err = data.WriteImage(buf)
	if err != nil {
		return "", err
	}

	v := &captcha{
		value:   data.Text,
		image:   buf.Bytes(),
		created: time.Now(),
	}
	c.Mu.Lock()
	defer c.Mu.Unlock()

	id := uuid.New().String()
	c.Map[id] = v

	return id, nil
}

// Delete record from captcha map by id.
func (c *Storage) Delete(id string) {
	c.Mu.Lock()
	defer c.Mu.Unlock()
	_, ok := c.Map[id]
	if ok {
		delete(c.Map, id)
	}
}

// Should be started in separate goroutine when init.
func (c *Storage) Cleanup() {
	log.Info().Fields(map[string]interface{}{
		"cleanup-timeout": fmt.Sprintf("%v", CleanupTimeout),
		"captcha-timeout": fmt.Sprintf("%v", CaptchaTimeLimit),
	}).Msg("captcha")

	for {
		time.Sleep(CleanupTimeout)

		c.Mu.RLock()
		r := []string{}
		for key, value := range c.Map {
			if value == nil || time.Since(value.created) >= CaptchaTimeLimit {
				r = append(r, key)
			}
		}
		c.Mu.RUnlock()

		if len(r) == 0 {
			continue
		}
		log.Info().Fields(map[string]interface{}{
			"cleanup": len(r),
		}).Msg("captcha")

		// Removing expired captchas.
		for i := range r {
			c.Delete(r[i])
		}
	}
}
