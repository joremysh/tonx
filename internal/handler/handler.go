package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/joremysh/tonx/api"
	"github.com/joremysh/tonx/pkg/cache"
)

var _ api.ServerInterface = (*BookingSystem)(nil)
var StartUp string

type BookingSystem struct {
	gdb *gorm.DB
}

func NewBookingSystem(gdb *gorm.DB, redisClient *cache.RedisClient) *BookingSystem {

	return &BookingSystem{
		gdb: gdb,
	}
}

func (s *BookingSystem) GetLiveness(c *gin.Context) {
	c.JSON(http.StatusOK, api.Pong{
		StartTime: StartUp,
	})
}

func (s *BookingSystem) SearchFlights(c *gin.Context, params api.SearchFlightsParams) {
	// TODO implement me
	panic("implement me")
}
