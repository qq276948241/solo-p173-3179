package models

import (
	"errors"
	"time"

	"gorm.io/gorm"
)

type OrderStatus string

const (
	OrderUnpaid   OrderStatus = "unpaid"
	OrderPaid     OrderStatus = "paid"
	OrderCheckedIn OrderStatus = "checked_in"
	OrderCheckedOut OrderStatus = "checked_out"
	OrderCanceled OrderStatus = "canceled"
)

type Order struct {
	ID           uint           `gorm:"primaryKey" json:"id"`
	OrderNo      string         `gorm:"uniqueIndex;size:32;not null" json:"order_no"`
	PropertyID   uint           `gorm:"not null;index" json:"property_id"`
	TenantID     uint           `gorm:"not null;index" json:"tenant_id"`
	LandlordID   uint           `gorm:"not null;index" json:"landlord_id"`
	CheckInDate  time.Time      `gorm:"not null" json:"check_in_date"`
	CheckOutDate time.Time      `gorm:"not null" json:"check_out_date"`
	Days         int            `gorm:"not null" json:"days"`
	TotalAmount  float64        `gorm:"not null" json:"total_amount"`
	Status       OrderStatus    `gorm:"size:16;not null;default:unpaid;index" json:"status"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`

	Property *Property `gorm:"foreignKey:PropertyID" json:"-"`
	Tenant   *User     `gorm:"foreignKey:TenantID" json:"-"`
	Landlord *User     `gorm:"foreignKey:LandlordID" json:"-"`
}

func (o *Order) Validate() error {
	if o.PropertyID == 0 {
		return errors.New("property_id is required")
	}
	if o.TenantID == 0 {
		return errors.New("tenant_id is required")
	}
	if o.LandlordID == 0 {
		return errors.New("landlord_id is required")
	}
	if o.CheckInDate.IsZero() {
		return errors.New("check_in_date is required")
	}
	if o.CheckOutDate.IsZero() {
		return errors.New("check_out_date is required")
	}
	if !o.CheckOutDate.After(o.CheckInDate) {
		return errors.New("check_out_date must be after check_in_date")
	}
	if o.Days <= 0 {
		return errors.New("days must be positive")
	}
	if o.TotalAmount <= 0 {
		return errors.New("total_amount must be positive")
	}
	return nil
}
