package repository

import (
	"gorm.io/gorm"

	"github.com/joremysh/tonx/internal/model"
)

type Order interface {
	Create(*model.Order) error
	List(params *model.ListParams) ([]model.Order, int64, error)
}

func NewOrderRepo(gdb *gorm.DB) Order {
	return &orderRepo{gdb: gdb}
}

type orderRepo struct {
	gdb *gorm.DB
}

func (o *orderRepo) Create(order *model.Order) error {
	return o.gdb.Create(order).Error
}

func (o *orderRepo) List(params *model.ListParams) ([]model.Order, int64, error) {
	query := o.gdb
	var listFilterColumnNames = []string{"status", "booking_time", "order_number"}

	// Apply filters
	for _, field := range listFilterColumnNames {
		if s, ok := params.Filters[field]; ok {
			condition := field + " like ?"
			query = query.Where(condition, s)
		}
	}
	countQuery := query

	var totalCount int64
	if err := countQuery.Model(&model.Order{}).Count(&totalCount).Error; err != nil {
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

	var orders []model.Order
	if err := query.Find(&orders).Error; err != nil {
		return nil, 0, err
	}
	return orders, totalCount, nil
}
