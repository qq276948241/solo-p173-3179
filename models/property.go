package models

import (
	"errors"
	"time"

	"gorm.io/gorm"
)

type PropertyStatus string

const (
	PropertyOffline PropertyStatus = "offline"
	PropertyOnline  PropertyStatus = "online"
)

type Property struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	LandlordID  uint           `gorm:"not null;index" json:"landlord_id"`
	Title       string         `gorm:"size:128;not null" json:"title"`
	Description string         `gorm:"type:text" json:"description"`
	City        string         `gorm:"size:64;not null;index" json:"city"`
	Address     string         `gorm:"size:256;not null" json:"address"`
	PricePerDay float64        `gorm:"not null;index" json:"price_per_day"`
	Status      PropertyStatus `gorm:"size:16;not null;default:offline;index" json:"status"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`

	Landlord *User `gorm:"foreignKey:LandlordID" json:"-"`
}

func (p *Property) Validate() error {
	if p.LandlordID == 0 {
		return errors.New("landlord_id is required")
	}
	if p.Title == "" {
		return errors.New("title is required")
	}
	if p.City == "" {
		return errors.New("city is required")
	}
	if p.Address == "" {
		return errors.New("address is required")
	}
	if p.PricePerDay <= 0 {
		return errors.New("price_per_day must be positive")
	}
	if p.Status != "" && p.Status != PropertyOnline && p.Status != PropertyOffline {
		return errors.New("invalid property status")
	}
	return nil
}

func (p *Property) IsOnline() bool {
	return p.Status == PropertyOnline
}
