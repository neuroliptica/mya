package main

import (
	"bytes"
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	cgen "github.com/steambap/captcha"
)

const (
	CaptchaTimeLimit = 15 * time.Second
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
	data, err := cgen.New(150, 50)
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
	_, ok := c.Get(id)
	if !ok {
		return
	}
	c.Mu.Lock()
	defer c.Mu.Unlock()

	delete(c.Map, id)
}

// Should be started in separate goroutine when init.
func (c *Storage) Cleanup() {
	log.Info().Msgf(
		"captcha cleanup daemon initialized with timeout %v.",
		CleanupTimeout,
	)
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

		log.Info().Msgf("cleanup for %d captcha records.", len(r))
		// Removing expired captchas.
		c.Mu.Lock()
		for i := range r {
			delete(c.Map, r[i])
		}
		c.Mu.Unlock()
	}
}
