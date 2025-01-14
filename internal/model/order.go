package model

import "time"

// Order represents a flight booking order
type Order struct {
	ID          uint      `json:"id" gorm:"primaryKey;autoIncrement;type:uint"`
	FlightID    uint      `json:"flight_id" gorm:"type:uint;not null;index"`
	CustomerID  uint      `json:"customer_id" gorm:"type:uint;not null;index"`
	Status      string    `json:"status" gorm:"type:varchar(20);not null;default:'PENDING'"` // PENDING, CONFIRMED, CANCELLED, COMPLETED
	TotalAmount int       `json:"total_amount" gorm:"type:mediumint;not null"`               // In smallest currency unit (e.g., cents)
	OrderNumber string    `json:"order_number" gorm:"type:varchar(50);uniqueIndex;not null"`
	BookingTime time.Time `json:"booking_time" gorm:"type:timestamp;not null;default:CURRENT_TIMESTAMP"`
	CreatedAt   time.Time `json:"created_at" gorm:"type:timestamp;autoCreateTime"`
	UpdatedAt   time.Time `json:"updated_at" gorm:"type:timestamp;autoUpdateTime"`
	Flight      *Flight   `json:"flight" gorm:"foreignKey:FlightID"`
	Customer    *Customer `json:"customer" gorm:"foreignKey:CustomerID"`
}
