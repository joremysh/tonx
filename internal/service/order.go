package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/joremysh/tonx/api"
	"github.com/joremysh/tonx/internal/constant"
	"github.com/joremysh/tonx/internal/model"
	"github.com/joremysh/tonx/pkg/cache"
)

var (
	ErrFlightNotFound     = errors.New("flight not found")
	ErrCustomerNotFound   = errors.New("customer not found")
	ErrNoAvailableSeats   = errors.New("no available seats")
	ErrInvalidFlightState = errors.New("flight is not in bookable state")
)

// Order defines the interface for order operations
type Order interface {
	// CreateOrder creates a new order with concurrency control
	CreateOrder(ctx context.Context, req CreateOrderRequest) (*model.Order, error)
	// InitializeFlightSeats initializes or updates the available seats in Redis
	InitializeFlightSeats(ctx context.Context, flightID uint, availableSeats int) error
}

// CreateOrderRequest represents the request for creating an order
type CreateOrderRequest struct {
	FlightID     uint
	CustomerID   uint
	TicketAmount int
}

// orderService implements Order
type orderService struct {
	gdb         *gorm.DB
	redisClient *cache.RedisClient
}

// NewOrderService creates a new instance of Order
func NewOrderService(gdb *gorm.DB, redisClient *cache.RedisClient) Order {
	return &orderService{
		gdb:         gdb,
		redisClient: redisClient,
	}
}

// InitializeFlightSeats initializes or updates the available seats in Redis
func (s *orderService) InitializeFlightSeats(ctx context.Context, flightID uint, availableSeats int) error {
	key := fmt.Sprintf("flight:%d:available_seats", flightID)
	return s.redisClient.Set(ctx, key, availableSeats, 24*time.Hour)
}

// CreateOrder handles the order creation process with concurrency control
func (s *orderService) CreateOrder(ctx context.Context, req CreateOrderRequest) (*model.Order, error) {
	// Start database transaction
	tx := s.gdb.Begin()
	if tx.Error != nil {
		return nil, fmt.Errorf("failed to start transaction: %w", tx.Error)
	}

	committed := false
	defer func() {
		if r := recover(); r != nil && !committed {
			tx.Rollback()
		}
	}()

	// 1. Get and validate flight
	var flight model.Flight
	if err := tx.Where("id = ?", req.FlightID).First(&flight).Error; err != nil {
		tx.Rollback()
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrFlightNotFound
		}
		return nil, fmt.Errorf("failed to get flight: %w", err)
	}

	// Validate flight status
	if flight.Status != string(api.FlightStatusSCHEDULED) {
		tx.Rollback()
		return nil, ErrInvalidFlightState
	}

	// 2. Get and validate customer
	var customer model.Customer
	if err := tx.Where("id = ? AND status = ?", req.CustomerID, "ACTIVE").First(&customer).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrCustomerNotFound
		}
		return nil, fmt.Errorf("failed to get customer: %w", err)
	}

	// 3. Check and decrement available seats using Redis Lua script
	flightKey := fmt.Sprintf("flight:%d:available_seats", flight.ID)
	result, err := s.redisClient.Client.Eval(ctx, constant.CheckAndDecrementSeatsScript, []string{flightKey}, req.TicketAmount).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to execute Redis script: %w", err)
	}

	switch result {
	case -1:
		tx.Rollback()
		return nil, fmt.Errorf("flight seats not found in Redis")
	case 0:
		tx.Rollback()
		return nil, ErrNoAvailableSeats
	}

	// 4. Create order
	order := &model.Order{
		FlightID:    flight.ID,
		CustomerID:  customer.ID,
		Status:      string(api.OrderStatusPENDING),
		TotalAmount: flight.BasePrice * req.TicketAmount,
		OrderNumber: generateOrderNumber(constant.ORD_PREFIX),
		BookingTime: time.Now(),
	}

	if err := tx.Create(order).Error; err != nil {
		tx.Rollback()
		// Try to restore the seat in Redis
		err := s.redisClient.Set(ctx, flightKey, flight.AvailableSeats+req.TicketAmount, 24*time.Hour)
		if err != nil {
			// Log the Redis error but return the original error
			fmt.Printf("failed to restore seat in Redis: %v\n", err)
		}
		return nil, fmt.Errorf("failed to create order: %w", err)
	}

	// 5. Update flight available seats in database
	if err := tx.Model(&flight).Update("available_seats", gorm.Expr("available_seats - ?", req.TicketAmount)).Error; err != nil {
		tx.Rollback()
		// Try to restore the seat in Redis
		err := s.redisClient.Set(ctx, flightKey, flight.AvailableSeats+req.TicketAmount, 24*time.Hour)
		if err != nil {
			log.Printf("failed to restore seat in Redis: %v\n", err)
		}
		return nil, fmt.Errorf("failed to update flight seats: %w", err)
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		// Try to restore the seat in Redis
		err := s.redisClient.Set(ctx, flightKey, flight.AvailableSeats+1, 24*time.Hour)
		if err != nil {
			log.Printf("failed to restore seat in Redis: %v\n", err)
		}
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	committed = true
	return order, nil
}

// generateOrderNumber generates a unique order number
func generateOrderNumber(prefix string) string {
	timestamp := time.Now().Format("20060102")
	randomStr := uuid.New().String()[:8]
	return fmt.Sprintf("%s-%s-%s", prefix, timestamp, randomStr)
}
