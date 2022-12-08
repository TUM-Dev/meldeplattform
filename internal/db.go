package internal

import (
	"fmt"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func (a *App) initDB() error {
	db, err := gorm.Open(sqlite.Open("database.db"), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	// Migrate the schema
	err = db.Debug().AutoMigrate(
		&Message{},
		&Report{},
		&File{},
	)
	if err != nil {
		return fmt.Errorf("migrate db: %w", err)
	}
	a.db = db
	return nil
}
