package service

import (
	job "github.com/apk471/go-boilerplate/internal/lib/jobs"
	"github.com/apk471/go-boilerplate/internal/repository"
	"github.com/apk471/go-boilerplate/internal/server"
)

type Services struct {
	Auth    *AuthService
	Job     *job.JobService
	Movie   *MovieService
	Booking *BookingService
}

func NewServices(s *server.Server, repos *repository.Repositories) (*Services, error) {
	authService := NewAuthService(s)
	movieService := NewMovieService()
	bookingService := NewBookingService(movieService, repos.Booking)

	return &Services{
		Job:     s.Job,
		Auth:    authService,
		Movie:   movieService,
		Booking: bookingService,
	}, nil
}
