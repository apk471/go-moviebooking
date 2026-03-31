package service

import "github.com/apk471/go-boilerplate/internal/errs"

type Movie struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Rows        int    `json:"rows"`
	SeatsPerRow int    `json:"seats_per_row"`
}

type MovieService struct {
	movies []Movie
}

func NewMovieService() *MovieService {
	return &MovieService{
		movies: []Movie{
			{ID: "inception", Title: "Inception", Rows: 5, SeatsPerRow: 8},
			{ID: "dune", Title: "Dune: Part Two", Rows: 4, SeatsPerRow: 6},
		},
	}
}

func (s *MovieService) ListMovies() []Movie {
	movies := make([]Movie, len(s.movies))
	copy(movies, s.movies)

	return movies
}

func (s *MovieService) GetMovieByID(movieID string) (Movie, error) {
	for _, movie := range s.movies {
		if movie.ID == movieID {
			return movie, nil
		}
	}

	return Movie{}, errs.NewNotFoundError("Movie not found", true, nil)
}
