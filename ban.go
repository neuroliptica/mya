package main

import (
	"crypto/md5"
	"fmt"
	"time"

	"gorm.io/gorm"
)

type Ban struct {
	ID uint

	// Ip hash, not ip itself.
	Hash   string `gorm:"unique"`
	Reason string
	Until  time.Time

	CreatedAt time.Time
	UpdatedAt time.Time
}

func migrateBan() error {
	return db.AutoMigrate(&Ban{})
}

func hash(ip string) string {
	// todo(zvezdochka): better hash function.
	// also add some salt from local config i guess.
	return fmt.Sprintf("%x", md5.Sum([]byte(ip)))
}

func (b *Ban) HasExpired() bool {
	return time.Now().After(b.Until)
}

func ban(ip string, reason string, until time.Time) error {
	// todo(zvezdochka): rewrite using REPLACE sql query.
	h := hash(ip)
	var b Ban
	err := db.Where("hash = ?", h).First(&b).Error
	switch err {
	case nil:
		break
	case gorm.ErrRecordNotFound:
		return db.Create(&Ban{
			Hash:   h,
			Until:  until,
			Reason: reason,
		}).Error
	default:
		return err
	}

	return db.Model(b).Updates(Ban{
		Until:  until,
		Reason: reason,
	}).Error
}
