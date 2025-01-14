package service

import (
	"context"
	"time"

	"github.com/joremysh/tonx/internal/model"
	"github.com/joremysh/tonx/internal/repository"
	"github.com/joremysh/tonx/pkg/cache"
)

type Flight interface {
	ListFlights(ctx context.Context, params *model.ListParams, departureDate *time.Time) (*PaginatedResult[model.Flight], error)
}

type PaginatedResult[T any] struct {
	Data       []T
	TotalCount int64
	Page       int
	PageSize   int
}

func NewFlightService(repo repository.Flight, redisClient *cache.RedisClient) Flight {
	return &flightService{
		repo:        repo,
		redisClient: redisClient,
	}
}

type flightService struct {
	repo        repository.Flight
	redisClient *cache.RedisClient
}

func (f *flightService) ListFlights(ctx context.Context, params *model.ListParams, departureDate *time.Time) (*PaginatedResult[model.Flight], error) {
	results, totalCount, err := f.repo.List(params, departureDate)
	if err != nil {
		return nil, err
	}
	return &PaginatedResult[model.Flight]{
		Data:       results,
		TotalCount: totalCount,
		Page:       params.Page,
		PageSize:   params.PageSize,
	}, nil
}
