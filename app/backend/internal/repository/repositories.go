package repository

import "github.com/apk471/go-boilerplate/internal/server"

type Repositories struct {
	Booking *BookingRepository
}

func NewRepositories(s *server.Server) *Repositories {
	return &Repositories{
		Booking: NewBookingRepository(s.Redis),
	}
}
