package main

import (
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"
)

const (
	CaptchaTimeLimit = 15 * time.Second
	CleanupTimeout   = 30 * time.Second
)

var (
	captchas *CaptchaMap
)

func initCaptchas() {
	captchas = NewCaptchaMap()
	go captchas.Cleanup()
}

type captcha struct {
	value string
	image []byte

	created time.Time
}

type CaptchaMap struct {
	Map map[string]*captcha
	Mu  *sync.RWMutex
}

func NewCaptchaMap() *CaptchaMap {
	return &CaptchaMap{
		Map: make(map[string]*captcha),
		Mu:  &sync.RWMutex{},
	}
}

// Get captcha by id, thread safe.
func (c *CaptchaMap) Get(id string) (*captcha, bool) {
	c.Mu.RLock()
	defer c.Mu.RUnlock()

	v, ok := c.Map[id]
	return v, ok
}

// Check if provided value is valid captcha value.
func (c *CaptchaMap) Check(value string, id string) bool {
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
func (c *CaptchaMap) GetImage(id string) ([]byte, error) {
	v, ok := c.Get(id)
	if !ok || v == nil {
		return nil, errors.New("invalid id or captcha expired")
	}

	return v.image, nil
}

// Create new captcha record and return it's id.
func (c *CaptchaMap) Create(value string) string {
	v := &captcha{
		value: value,
		// todo generate image.
		image:   []byte("empty"),
		created: time.Now(),
	}
	c.Mu.Lock()
	defer c.Mu.Unlock()

	id := uuid.New().String()
	c.Map[id] = v

	return id
}

// Delete record from captcha map by id.
func (c *CaptchaMap) Delete(id string) {
	_, ok := c.Get(id)
	if !ok {
		return
	}
	c.Mu.Lock()
	defer c.Mu.Unlock()

	delete(c.Map, id)
}

// Should be started in separate goroutine when init.
func (c *CaptchaMap) Cleanup() {
	logger.Info().Msgf(
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

		logger.Info().Msgf("cleanup for %d captcha records.", len(r))
		// Write lock for removing expired captchas.
		c.Mu.Lock()
		for i := range r {
			delete(c.Map, r[i])
		}
		c.Mu.Unlock()
	}
}
