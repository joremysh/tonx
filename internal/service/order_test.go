package service

import (
	"context"
	"log"
	"sync"
	"testing"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/ory/dockertest/v3"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/joremysh/tonx/api"
	"github.com/joremysh/tonx/internal/model"
	"github.com/joremysh/tonx/internal/repository"
	"github.com/joremysh/tonx/pkg/cache"
	testingx "github.com/joremysh/tonx/pkg/testing"
)

var (
	gdb           *gorm.DB
	err           error
	pool          *dockertest.Pool
	resource      *dockertest.Resource
	redisResource *dockertest.Resource
	redisClient   *redis.Client
	rc            *cache.RedisClient
)

func TestMain(m *testing.M) {
	pool, resource, gdb, redisResource, redisClient, err = testingx.NewTestContainers()
	if err != nil {
		log.Fatal(err.Error())
	}
	err = repository.Migrate(gdb)
	if err != nil {
		log.Fatal(err.Error())
	}
	rc = &cache.RedisClient{Client: redisClient}

	defer func() {
		if err = pool.Purge(resource); err != nil {
			log.Fatal(err.Error())
		}
		if err = pool.Purge(redisResource); err != nil {
			log.Fatal(err.Error())
		}
	}()

	m.Run()
}

func TestOrderService_CreateOrder(t *testing.T) {
	svc := NewOrderService(gdb, rc)

	flight := &model.Flight{}
	err = gdb.First(flight).Error
	require.NoError(t, err)
	require.NotNil(t, flight)
	require.NotZero(t, flight.ID)

	customer := &model.Customer{
		Name:  gofakeit.Name(),
		Email: gofakeit.Email(),
		Phone: gofakeit.Phone(),
	}
	err = gdb.Save(customer).Error
	require.NoError(t, err)

	ctx := context.Background()
	ticketAmount := 42
	order, err := svc.CreateOrder(ctx, CreateOrderRequest{
		FlightID:     flight.ID,
		CustomerID:   customer.ID,
		TicketAmount: ticketAmount,
	})
	require.NoError(t, err)
	require.NotNil(t, order)
	require.NotZero(t, order.ID)

	check := &model.Flight{}
	err = gdb.First(check, flight.ID).Error
	require.NoError(t, err)
	require.Equal(t, flight.AvailableSeats-ticketAmount, check.AvailableSeats)

	checkOrder := &model.Order{}
	err = gdb.First(checkOrder, order.ID).Error
	require.NoError(t, err)
	require.Equal(t, order.ID, checkOrder.ID)
	require.Equal(t, flight.BasePrice*ticketAmount, checkOrder.TotalAmount)

	var availableSeats int
	err = rc.Get(ctx, flight.FlightKey(), &availableSeats)
	require.NoError(t, err)
	require.Equal(t, check.AvailableSeats, availableSeats)
}

func TestOrderService_CreateOrderWithNotEnoughTicket(t *testing.T) {
	svc := NewOrderService(gdb, rc)

	flight := &model.Flight{}
	err = gdb.First(flight).Error
	require.NoError(t, err)
	require.NotNil(t, flight)
	require.NotZero(t, flight.ID)

	customer := &model.Customer{
		Name:  gofakeit.Name(),
		Email: gofakeit.Email(),
		Phone: gofakeit.Phone(),
	}
	err = gdb.Save(customer).Error
	require.NoError(t, err)

	ctx := context.Background()
	_, err = svc.CreateOrder(ctx, CreateOrderRequest{
		FlightID:     flight.ID,
		CustomerID:   customer.ID,
		TicketAmount: flight.AvailableSeats + 1,
	})
	require.Error(t, err)
	require.ErrorIs(t, err, ErrNoAvailableSeats)

	check := &model.Flight{}
	err = gdb.First(check, flight.ID).Error
	require.NoError(t, err)
	require.Equal(t, flight.AvailableSeats, check.AvailableSeats)

	var availableSeats int
	err = rc.Get(ctx, flight.FlightKey(), &availableSeats)
	require.NoError(t, err)
	require.Equal(t, flight.AvailableSeats, availableSeats)
}

func TestOrderService_CreateOrderMultipleTimesInSerial(t *testing.T) {
	svc := NewOrderService(gdb, rc)

	flight := &model.Flight{}
	err = gdb.First(flight).Error
	require.NoError(t, err)
	require.NotNil(t, flight)
	require.NotZero(t, flight.ID)

	customer := &model.Customer{
		Name:  gofakeit.Name(),
		Email: gofakeit.Email(),
		Phone: gofakeit.Phone(),
	}
	err = gdb.Save(customer).Error
	require.NoError(t, err)

	ctx := context.Background()
	ticketAmount := 5
	times := 4
	for i := 0; i < times; i++ {
		order, err := svc.CreateOrder(ctx, CreateOrderRequest{
			FlightID:     flight.ID,
			CustomerID:   customer.ID,
			TicketAmount: ticketAmount,
		})
		require.NoError(t, err)
		require.NotNil(t, order)
		require.NotZero(t, order.ID)
	}

	check := &model.Flight{}
	err = gdb.First(check, flight.ID).Error
	require.NoError(t, err)
	require.Equal(t, flight.AvailableSeats-ticketAmount*times, check.AvailableSeats)

	var availableSeats int
	err = rc.Get(ctx, flight.FlightKey(), &availableSeats)
	require.NoError(t, err)
	require.Equal(t, check.AvailableSeats, availableSeats)
}

func TestOrderService_CreateOrder_Concurrent(t *testing.T) {
	svc := NewOrderService(gdb, rc)
	ctx := context.Background()

	var (
		numGoroutines        = 4 // Number of concurrent orders
		ticketAmount         = 1 // Each order requests 1 tickets
		totalTickets         = 2 // Total tickets to be booked
		expectedSuccessCount = 2 // Expected half of the orders to succeed due to total ticket amounts
		expectedErrorCount   = 2 // Expected half of the orders to fail
	)

	flights := make([]model.Flight, 0)
	err = gdb.Find(&flights).Error
	require.NoError(t, err)
	require.NotEmpty(t, flights)

	for i, flight := range flights {
		err = rc.Delete(ctx, flight.FlightKey())
		require.NoError(t, err)
		flights[i].AvailableSeats = totalTickets
	}
	err = gdb.Save(&flights).Error
	require.NoError(t, err)

	flight := flights[0]
	require.NotZero(t, flight.ID)

	// Create a customer
	customer := &model.Customer{
		Name:  gofakeit.Name(),
		Email: gofakeit.Email(),
		Phone: gofakeit.Phone(),
	}
	err = gdb.Save(customer).Error
	require.NoError(t, err)

	// Use wait group to wait for all goroutines
	var wg sync.WaitGroup
	// Channel to collect results
	results := make(chan error, numGoroutines)
	// Channel to collect successful orders
	orders := make(chan *model.Order, numGoroutines)

	// Start concurrent order creation
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			order, err := svc.CreateOrder(ctx, CreateOrderRequest{
				FlightID:     flight.ID,
				CustomerID:   customer.ID,
				TicketAmount: ticketAmount,
			})

			if err != nil {
				results <- err
				return
			}
			orders <- order
			results <- nil
		}()
	}

	// Wait for all goroutines to complete
	wg.Wait()
	close(results)
	close(orders)

	// Count successful and failed orders
	successCount := 0
	var errors []error
	for err = range results {
		if err != nil {
			errors = append(errors, err)
		} else {
			successCount++
		}
	}

	// Collect all orders
	var completedOrders []*model.Order
	for order := range orders {
		completedOrders = append(completedOrders, order)
	}

	// Verify results
	t.Logf("Successful orders: %d, Failed orders: %d", successCount, len(errors))
	require.Equal(t, expectedSuccessCount, successCount)
	require.Equal(t, expectedErrorCount, len(errors))

	// Verify final flight seat count in database
	var finalFlight model.Flight
	err = gdb.First(&finalFlight, flight.ID).Error
	require.NoError(t, err)
	expectedSeats := flight.AvailableSeats - (successCount * ticketAmount)
	require.Equal(t, expectedSeats, finalFlight.AvailableSeats, "Final available seats count mismatch")

	// Verify Redis seat count
	var redisSeats int
	err = rc.Get(ctx, flight.FlightKey(), &redisSeats)
	require.NoError(t, err)
	require.Equal(t, expectedSeats, redisSeats, "Redis seats count mismatch")

	// Verify order details
	for _, order := range completedOrders {
		require.NotZero(t, order.ID)
		require.Equal(t, flight.ID, order.FlightID)
		require.Equal(t, customer.ID, order.CustomerID)
		require.Equal(t, ticketAmount*flight.BasePrice, order.TotalAmount)
		require.Equal(t, string(api.OrderStatusCOMPLETED), order.Status)
	}
}
