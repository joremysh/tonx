package repository

import (
	"gorm.io/gorm"

	"github.com/joremysh/tonx/internal/model"
)

func Migrate(gdb *gorm.DB) error {
	err := gdb.AutoMigrate(&model.Flight{})
	if err != nil {
		return err
	}

	for _, seed := range All() {
		if err = seed.Run(gdb); err != nil {
			return err
		}
	}
	return nil
}
