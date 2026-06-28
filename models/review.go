package models

import (
	"errors"
	"time"

	"gorm.io/gorm"
)

type Review struct {
	ID         uint           `gorm:"primaryKey" json:"id"`
	OrderID    uint           `gorm:"not null;uniqueIndex" json:"order_id"`
	PropertyID uint           `gorm:"not null;index:idx_property_created" json:"property_id"`
	TenantID   uint           `gorm:"not null;index" json:"tenant_id"`
	Rating     int            `gorm:"not null;index" json:"rating"`
	Comment    string         `gorm:"type:text" json:"comment"`
	CreatedAt  time.Time      `gorm:"index:idx_property_created,sort:desc" json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`

	Order    *Order    `gorm:"foreignKey:OrderID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT" json:"-"`
	Property *Property `gorm:"foreignKey:PropertyID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"-"`
	Tenant   *User     `gorm:"foreignKey:TenantID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT" json:"-"`
}

func (r *Review) Validate() error {
	if r.OrderID == 0 {
		return errors.New("order_id is required")
	}
	if r.PropertyID == 0 {
		return errors.New("property_id is required")
	}
	if r.TenantID == 0 {
		return errors.New("tenant_id is required")
	}
	if r.Rating < 1 || r.Rating > 5 {
		return errors.New("rating must be between 1 and 5")
	}
	return nil
}
