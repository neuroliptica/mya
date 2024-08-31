package main

import (
	"errors"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var (
	db *gorm.DB
)

type DataModel struct {
	ID        uint      `json:"id"`
	CreatedAt time.Time `json:"-"`
	UpdatedAt time.Time `json:"-"`
}

func assignSqlite(name string) error {
	var err error
	db, err = gorm.Open(
		sqlite.Open(name),
		&gorm.Config{},
	)
	return err
}

func migrate() error {
	migrations := Maybe{
		migrateBoard,
		migratePost,
		migrateBan,
		migrateFile,
	}
	return migrations.Eval()
}

func checkRecord[T any](schema *T, conds ...any) (*T, error) {
	var res T
	result := db.
		Where(schema).
		First(&res, conds...)

	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, errors.New("no records")
	}

	return &res, nil
}
