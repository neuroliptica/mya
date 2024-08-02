package main

import (
	"errors"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var (
	// global db entry.
	db *gorm.DB
)

func assignSqlite(name string) error {
	var err error
	db, err = gorm.Open(
		sqlite.Open(name),
		&gorm.Config{},
	)
	return err
}

func migrate() error {
	// declaring migrations.
	migrations := Maybe{
		migrateBoard,
		migratePost,
	}

	return migrations.Eval()
}

// Check if record with provided conditions exists.
func checkRecord[T any](schema *T) (*T, error) {
	var res T
	result := db.
		Where(schema).
		First(&res)

	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, errors.New("no records")
	}

	return &res, nil
}

// Generic selector.
func get[T any](schema *T, conds ...any) error {
	result := db.Find(schema, conds...)
	return result.Error
}
