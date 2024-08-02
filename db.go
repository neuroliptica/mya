package main

import (
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
