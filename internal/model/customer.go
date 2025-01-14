package model

import "time"

// Customer represents a flight booking customer
type Customer struct {
	ID        uint      `json:"id" gorm:"primaryKey;autoIncrement;type:uint"`
	Name      string    `json:"name" gorm:"type:varchar(100);not null"`
	Email     string    `json:"email" gorm:"type:varchar(100);uniqueIndex;not null"`
	Phone     string    `json:"phone" gorm:"type:varchar(20);not null"`
	Status    string    `json:"status" gorm:"type:varchar(20);not null;default:'ACTIVE'"` // ACTIVE, INACTIVE
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
