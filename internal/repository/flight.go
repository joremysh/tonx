package repository

import (
	"time"

	"gorm.io/gorm"

	"github.com/joremysh/tonx/internal/model"
)

type Flight interface {
	Create(*model.Flight) error
	List(params *model.ListParams, departureDate *time.Time) ([]model.Flight, int64, error)
}

func NewFlightRepo(gdb *gorm.DB) Flight {
	return &flightRepo{gdb: gdb}
}

type flightRepo struct {
	gdb *gorm.DB
}

func (f *flightRepo) Create(flight *model.Flight) error {
	return f.gdb.Create(flight).Error
}

func (f *flightRepo) List(params *model.ListParams, departureDate *time.Time) ([]model.Flight, int64, error) {
	query := f.gdb
	var listFilterColumnNames = []string{"flight_number", "airline", "departure_city", "arrival_city"}

	if departureDate == nil {
		year, month, day := time.Now().Date()
		today := time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
		departureDate = &today
	}
	query = query.Where("departure_time >= ?", departureDate)

	// Apply filters
	for _, field := range listFilterColumnNames {
		if s, ok := params.Filters[field]; ok {
			condition := field + " like ?"
			query = query.Where(condition, s)
		}
	}
	countQuery := query

	var totalCount int64
	if err := countQuery.Model(&model.Flight{}).Count(&totalCount).Error; err != nil {
		return nil, 0, err
	}

	// Apply sorting
	if params.SortBy != "" {
		order := params.SortBy
		if params.SortOrder == "desc" {
			order += " DESC"
		}
		query = query.Order(order)
	}

	offset := (params.Page - 1) * params.PageSize
	query = query.Offset(offset).Limit(params.PageSize)

	var flights []model.Flight
	if err := query.Find(&flights).Error; err != nil {
		return nil, 0, err
	}
	return flights, totalCount, nil
}
