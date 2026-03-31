package handler

import (
	"github.com/apk471/go-boilerplate/internal/server"
	"github.com/apk471/go-boilerplate/internal/service"
)

type Handlers struct {
	Health  *HealthHandler
	Movie   *MovieHandler
	Booking *BookingHandler
	OpenAPI *OpenAPIHandler
}

func NewHandlers(s *server.Server, services *service.Services) *Handlers {
	return &Handlers{
		Health:  NewHealthHandler(s),
		Movie:   NewMovieHandler(s, services.Movie),
		Booking: NewBookingHandler(s, services.Booking),
		OpenAPI: NewOpenAPIHandler(s),
	}
}
