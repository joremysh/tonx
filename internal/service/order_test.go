package service

import (
	"context"
	"log"
	"testing"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/ory/dockertest/v3"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

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
	tx := gdb

	svc := NewOrderService(tx, &cache.RedisClient{Client: redisClient})
	flight := &model.Flight{}
	err = tx.First(flight).Error
	require.NoError(t, err)
	require.NotNil(t, flight)
	require.NotZero(t, flight.ID)
	customer := &model.Customer{
		Name:  gofakeit.Name(),
		Email: gofakeit.Email(),
		Phone: gofakeit.Phone(),
	}
	err = tx.Save(customer).Error
	require.NoError(t, err)

	ctx := context.Background()
	order, err := svc.CreateOrder(ctx, CreateOrderRequest{
		FlightID:   flight.ID,
		CustomerID: customer.ID,
	})
	require.NoError(t, err)
	require.NotNil(t, order)
	require.NotZero(t, order.ID)
}
