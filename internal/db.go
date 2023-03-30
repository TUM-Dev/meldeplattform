package internal

import (
	"fmt"
	"github.com/TUM-Dev/meldeplattform/pkg/model"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func (a *App) initDB() error {
	db, err := gorm.Open(sqlite.Open("files/database.db"), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	// Migrate the schema
	err = db.AutoMigrate(
		&model.Message{},
		&model.Report{},
		&model.File{},
		&model.Topic{},
		&model.Field{},
		&model.Admin{},
	)
	if err != nil {
		return fmt.Errorf("migrate db: %w", err)
	}
	a.db = db
	return nil
}
