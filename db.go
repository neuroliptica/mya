package main

import (
	"errors"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var (
	db *gorm.DB
)

func assignSqlite(name string) error {
	var err error
	db, err = gorm.Open(
		sqlite.Open(name),
		&gorm.Config{
			//Logger: logger.Default.LogMode(logger.Silent),
		},
	)
	return err
}

func migrate() error {
	migrations := Maybe{
		migrateBoard,
		migratePost,
		migrateBan,
	}
	return migrations.Eval()
}

// Check if record with provided conditions exists.
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

// Generic selector.
func get[T any](schema *T, conds ...any) error {
	result := db.Find(schema, conds...)
	return result.Error
}
