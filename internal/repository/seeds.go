package repository

import (
	"strconv"
	"time"

	"github.com/brianvoe/gofakeit/v7"
	"gorm.io/gorm"

	"github.com/joremysh/tonx/api"
	"github.com/joremysh/tonx/internal/model"
)

type Seed struct {
	Name string
	Run  func(*gorm.DB) error
}

func All() []Seed {
	seeds := make([]Seed, 5)
	for i := 0; i < len(seeds); i++ {
		flight := MockFlight()
		seeds[i].Name = flight.FlightNumber
		seeds[i].Run = func(gdb *gorm.DB) error {
			repo := NewFlightRepo(gdb)
			return repo.Create(flight)
		}
	}

	return seeds
}

func MockFlight() *model.Flight {
	minute := gofakeit.Minute()
	minute = minute - minute%5
	departureTime := time.Date(2025, time.Month(gofakeit.Month()), gofakeit.Day(), gofakeit.Hour(), minute, 0, 0, time.Local)
	arrivalTime := departureTime.Add(time.Duration(30*gofakeit.IntRange(4, 8)) * time.Minute)
	return &model.Flight{
		FlightNumber:   "BR" + strconv.Itoa(gofakeit.IntRange(100, 999)),
		Airline:        "EVA AIR",
		DepartureCity:  gofakeit.City(),
		ArrivalCity:    gofakeit.City(),
		DepartureTime:  departureTime,
		ArrivalTime:    arrivalTime,
		Aircraft:       "Airbus A330",
		Status:         string(api.FlightStatusSCHEDULED),
		TotalSeats:     100,
		AvailableSeats: 120,
		BasePrice:      6000,
	}
}
