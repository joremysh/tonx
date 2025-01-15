package model

import (
	"fmt"
	"time"

	"github.com/joremysh/tonx/internal/constant"
)

// Flight represents a scheduled flight
type Flight struct {
	ID             uint      `json:"id" gorm:"primaryKey;autoIncrement;type:uint"`
	FlightNumber   string    `json:"flight_number" gorm:"uniqueIndex;type:varchar(20);not null"`
	Airline        string    `json:"airline" gorm:"type:varchar(100);not null"`
	DepartureCity  string    `json:"departure_city" gorm:"type:varchar(100);not null"`
	ArrivalCity    string    `json:"arrival_city" gorm:"type:varchar(100);not null"`
	DepartureTime  time.Time `json:"departure_time" gorm:"type:timestamp;not null;index"`
	ArrivalTime    time.Time `json:"arrival_time" gorm:"type:timestamp;not null"`
	Aircraft       string    `json:"aircraft" gorm:"type:varchar(50);not null"`
	Status         string    `json:"status" gorm:"type:varchar(20);not null;default:'SCHEDULED'"` // SCHEDULED, DELAYED, CANCELLED, IN_PROGRESS, COMPLETED
	TotalSeats     int       `json:"total_seats" gorm:"type:int;not null"`
	AvailableSeats int       `json:"available_seats" gorm:"type:int;not null"`
	BasePrice      int       `json:"base_price" gorm:"type:mediumint;not null"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

func (f Flight) FlightKey() string {
	return fmt.Sprintf(constant.FLIGHT_KEY, f.ID)
}
