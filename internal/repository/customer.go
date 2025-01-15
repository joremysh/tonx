package repository

import (
	"gorm.io/gorm"

	"github.com/joremysh/tonx/internal/model"
)

type Customer interface {
	Create(customer *model.Customer) error
}

func NewCustomerRepo(gdb *gorm.DB) Customer {
	return &customerRepo{gdb: gdb}
}

type customerRepo struct {
	gdb *gorm.DB
}

func (o *customerRepo) Create(customer *model.Customer) error {
	return o.gdb.Create(customer).Error
}
