package database

import (
	"project173/models"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var DB *gorm.DB

func Init(dbPath string) error {
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		return err
	}
	DB = db
	return DB.AutoMigrate(&models.User{}, &models.Property{}, &models.Order{})
}

func InitTest() (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	err = db.AutoMigrate(&models.User{}, &models.Property{}, &models.Order{})
	if err != nil {
		return nil, err
	}
	return db, nil
}
