package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/joremysh/tonx/api"
	"github.com/joremysh/tonx/internal/model"
	"github.com/joremysh/tonx/internal/repository"
	"github.com/joremysh/tonx/internal/service"
	"github.com/joremysh/tonx/pkg/cache"
)

var _ api.ServerInterface = (*BookingSystem)(nil)
var StartUp string

type BookingSystem struct {
	gdb           *gorm.DB
	flightService service.Flight
}

func NewBookingSystem(gdb *gorm.DB, redisClient *cache.RedisClient) *BookingSystem {
	flightRepo := repository.NewFlightRepo(gdb)
	return &BookingSystem{
		gdb:           gdb,
		flightService: service.NewFlightService(flightRepo, redisClient),
	}
}

func (s *BookingSystem) GetLiveness(c *gin.Context) {
	c.JSON(http.StatusOK, api.Pong{
		StartTime: StartUp,
	})
}

func (s *BookingSystem) SearchFlights(c *gin.Context, params api.SearchFlightsParams) {
	result, err := s.flightService.ListFlights(c.Request.Context(), parseListParams(params), &params.DepartureDate.Time)
	if err != nil {
		sendErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	resp := &api.SearchFlightResponse{
		Data:       make([]api.Flight, len(result.Data)),
		Page:       result.Page,
		PageSize:   result.PageSize,
		TotalCount: result.TotalCount,
	}
	for i, Flight := range result.Data {
		converted := ConvertToFlightResponse(&Flight)
		resp.Data[i] = *converted
	}

	c.JSON(http.StatusOK, resp)
}

func ConvertToFlightResponse(flight *model.Flight) *api.Flight {
	return &api.Flight{
		Id:             flight.ID,
		Aircraft:       flight.Aircraft,
		Airline:        flight.Airline,
		ArrivalCity:    flight.ArrivalCity,
		ArrivalTime:    flight.ArrivalTime,
		AvailableSeats: flight.AvailableSeats,
		BasePrice:      flight.BasePrice,
		DepartureCity:  flight.DepartureCity,
		DepartureTime:  flight.DepartureTime,
		FlightNumber:   flight.FlightNumber,
		Status:         api.FlightStatus(flight.Status),
	}
}

func parseListParams(params api.SearchFlightsParams) *model.ListParams {
	listParams := &model.ListParams{}
	if params.PageSize != nil {
		listParams.PageSize = *params.PageSize
	}
	if params.Page != nil {
		listParams.Page = *params.Page
	}
	if params.SortBy != nil {
		listParams.SortBy = string(*params.SortBy)
	}
	if params.SortOrder != nil {
		listParams.SortOrder = string(*params.SortOrder)
	}
	if params.Filters != nil {
		listParams.Filters = *params.Filters
	}
	return listParams
}

func sendErrorResponse(c *gin.Context, code int, errMsg string) {
	c.JSON(code, api.Error{
		Code:    code,
		Message: errMsg,
	})
}
