package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

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
	key := fmt.Sprintf(constant.FLIGHT_KEY, flightID)
	return s.redisClient.Set(ctx, key, availableSeats, 24*time.Hour)
}

func (s *orderService) CreateOrder(ctx context.Context, req CreateOrderRequest) (*model.Order, error) {
	flightKey := fmt.Sprintf("flight:%d:available_seats", req.FlightID)

	// 1. Check available seats in Redis first
	originalSeats, err := s.redisClient.Client.Get(ctx, flightKey).Int()
	if err != nil {
		if !errors.Is(err, redis.Nil) {
			return nil, fmt.Errorf("failed to get available seats from Redis: %w", err)
		}
		// Key doesn't exist, get flight info from database
		var flight model.Flight
		if err = s.gdb.Where("id = ?", req.FlightID).First(&flight).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, ErrFlightNotFound
			}
			return nil, fmt.Errorf("failed to get flight: %w", err)
		}

		originalSeats = flight.AvailableSeats
		log.Println("originalSeats from DB", "originalSeats", originalSeats)

		// Initialize Redis with current available seats using SetNX
		if _, err = s.redisClient.Client.SetNX(ctx, flightKey, originalSeats, 24*time.Hour).Result(); err != nil {
			return nil, fmt.Errorf("failed to initialize Redis with available seats: %w", err)
		}
		log.Println("set originalSeats from DB to Redis successfully.", "originalSeats", originalSeats)
	} else if originalSeats == 0 {
		// Key exists but no seats available
		return nil, ErrNoAvailableSeats
	}

	// 2. Check and decrement available seats using Redis Lua script
	result, err := s.redisClient.Client.Eval(ctx, constant.CheckAndDecrementSeatsScript, []string{flightKey}, req.TicketAmount).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to execute Redis script: %w", err)
	}

	resultInt, ok := result.(int64)
	if !ok {
		return nil, fmt.Errorf("failed to parse Redis script result: not an integer")
	}

	switch resultInt {
	case -1:
		return nil, fmt.Errorf("flight seats not found in Redis")
	case 0:
		return nil, ErrNoAvailableSeats
	}
	if resultInt != 1 {
		return nil, fmt.Errorf("invalid Redis script result: %d", resultInt)
	}

	// Prepare to restore Redis seats if anything fails after this point
	seatRestored := false
	defer func() {
		if !seatRestored {
			// Restore to the original value before decrement
			if err = s.redisClient.Client.Set(ctx, flightKey, originalSeats, 24*time.Hour).Err(); err != nil {
				log.Printf("failed to restore seat in Redis: %v\n", err)
			}
		}
	}()

	// 3. Start database transaction only for writing data
	tx := s.gdb.Begin()
	if tx.Error != nil {
		return nil, fmt.Errorf("failed to start transaction: %w", tx.Error)
	}

	// Ensure transaction rollback on failure
	committed := false
	defer func() {
		if !committed {
			tx.Rollback()
		}
	}()

	// 4. Lock and get flight for final update
	var flight model.Flight
	if err = tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&flight, req.FlightID).Error; err != nil {
		return nil, fmt.Errorf("failed to lock flight record: %w", err)
	}

	// Double-check available seats
	if flight.AvailableSeats < req.TicketAmount {
		return nil, ErrNoAvailableSeats
	}

	// 5. Create order
	order := &model.Order{
		FlightID:    flight.ID,
		CustomerID:  req.CustomerID,
		Status:      string(api.OrderStatusCOMPLETED),
		TotalAmount: flight.BasePrice * req.TicketAmount,
		OrderNumber: generateOrderNumber(constant.ORD_PREFIX),
		BookingTime: time.Now(),
	}

	if err = tx.Create(order).Error; err != nil {
		return nil, fmt.Errorf("failed to create order: %w", err)
	}

	// 6. Update flight available seats in database
	if err = tx.Model(&flight).Update("available_seats", gorm.Expr("available_seats - ?", req.TicketAmount)).Error; err != nil {
		return nil, fmt.Errorf("failed to update flight seats: %w", err)
	}

	// 7. Commit transaction
	if err = tx.Commit().Error; err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	committed = true
	seatRestored = true // No need to restore Redis seats on success
	return order, nil
}

// generateOrderNumber generates a unique order number
func generateOrderNumber(prefix string) string {
	timestamp := time.Now().Format("20060102")
	randomStr := uuid.New().String()[:8]
	return fmt.Sprintf("%s-%s-%s", prefix, timestamp, randomStr)
}
