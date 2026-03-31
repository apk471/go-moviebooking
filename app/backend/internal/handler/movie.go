package handler

import (
	"net/http"

	"github.com/apk471/go-boilerplate/internal/server"
	"github.com/apk471/go-boilerplate/internal/service"
	"github.com/labstack/echo/v4"
)

type ListMoviesRequest struct{}

func (r ListMoviesRequest) Validate() error {
	return nil
}

type MovieHandler struct {
	Handler
	movieService *service.MovieService
}

func NewMovieHandler(s *server.Server, movieService *service.MovieService) *MovieHandler {
	return &MovieHandler{
		Handler:      NewHandler(s),
		movieService: movieService,
	}
}

func (h *MovieHandler) ListMovies(c echo.Context, req ListMoviesRequest) ([]service.Movie, error) {
	return h.movieService.ListMovies(), nil
}

func (h *MovieHandler) RegisterListMovies() echo.HandlerFunc {
	return Handle(h.Handler, h.ListMovies, http.StatusOK, ListMoviesRequest{})
}
