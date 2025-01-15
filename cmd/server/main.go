package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"

	middleware "github.com/oapi-codegen/gin-middleware"

	"github.com/joremysh/tonx/api"
	"github.com/joremysh/tonx/internal/handler"
	"github.com/joremysh/tonx/internal/repository"
	"github.com/joremysh/tonx/pkg/cache"
	"github.com/joremysh/tonx/pkg/database"
)

func NewServer(bookingSystem *handler.BookingSystem, port string) *http.Server {
	swagger, err := api.GetSwagger()

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading swagger spec\n: %s", err)
		os.Exit(1)
	}

	// Clear out the servers array in the swagger spec, that skips validating
	// that server names match. We don't know how this thing will be run.
	swagger.Servers = nil
	r := gin.Default()

	// Use our validation middleware to check all requests against the
	// OpenAPI schema.
	r.Use(middleware.OapiRequestValidator(swagger))

	api.RegisterHandlers(r, bookingSystem)

	s := &http.Server{
		Handler: r,
		Addr:    net.JoinHostPort("0.0.0.0", port),
	}
	return s
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	dsn := os.Getenv("DSN")
	redisHost := os.Getenv("REDIS_HOST")
	redisPort := os.Getenv("REDIS_PORT")

	gdb, err := database.NewDatabase(dsn)
	if err != nil {
		log.Fatal(err.Error())
	}
	err = repository.Migrate(gdb)
	if err != nil {
		log.Fatal(err.Error())
	}

	redisClient, err := cache.NewRedisClient(net.JoinHostPort(redisHost, redisPort))
	if err != nil {
		log.Fatal(err.Error())
	}

	handler.StartUp = time.Now().Format(time.RFC3339)
	bookingSystem := handler.NewBookingSystem(gdb, redisClient)
	s := NewServer(bookingSystem, port)

	log.Fatal(s.ListenAndServe())
}
