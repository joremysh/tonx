package repository

import (
	"log"
	"testing"
	"time"

	"github.com/ory/dockertest/v3"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/joremysh/tonx/internal/model"
	testingx "github.com/joremysh/tonx/pkg/testing"
)

var (
	gdb      *gorm.DB
	err      error
	pool     *dockertest.Pool
	resource *dockertest.Resource
)

func TestMain(m *testing.M) {
	pool, resource, gdb, err = testingx.NewMysqlInDocker()
	if err != nil {
		log.Fatal(err.Error())
	}
	err = Migrate(gdb)
	if err != nil {
		log.Fatal(err.Error())
	}

	m.Run()

	err = pool.Purge(resource)
	if err != nil {
		log.Fatal(err.Error())
	}
}

func TestFlightRepo_Create(t *testing.T) {
	tx := gdb.Begin()
	t.Cleanup(func() {
		tx.Rollback()
	})

	repo := NewFlightRepo(tx)
	flight := MockFlight()
	err = repo.Create(flight)
	require.NoError(t, err)
	require.NotZero(t, flight.ID)

	check := &model.Flight{}
	err = tx.First(check, &model.Flight{FlightNumber: flight.FlightNumber}).Error
	require.NoError(t, err)
	require.Equal(t, flight.ID, check.ID)
	require.Equal(t, flight.DepartureCity, check.DepartureCity)
	require.Equal(t, flight.ArrivalCity, check.ArrivalCity)
}

func TestFlightRepo_List(t *testing.T) {
	tx := gdb.Begin()
	t.Cleanup(func() {
		tx.Rollback()
	})

	repo := NewFlightRepo(tx)
	flight := MockFlight()
	year, month, day := time.Now().Date()
	today := time.Date(year, month, day, 10, 0, 0, 0, time.UTC)
	flight.DepartureTime = today
	flight.ArrivalTime = today.Add(2 * time.Hour)

	err = repo.Create(flight)
	require.NoError(t, err)

	flights, total, err := repo.List(&model.ListParams{
		Page:     1,
		PageSize: 10,
		Filters:  map[string]string{"flight_number": flight.FlightNumber},
	}, nil)
	require.NoError(t, err)
	require.NotZero(t, total)
	require.Len(t, flights, 1)
	require.Equal(t, flight.ID, flights[0].ID)
	require.Equal(t, flight.FlightNumber, flights[0].FlightNumber)
	require.Equal(t, flight.DepartureCity, flights[0].DepartureCity)
	require.Equal(t, flight.ArrivalCity, flights[0].ArrivalCity)
}
