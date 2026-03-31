package service

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/apk471/go-boilerplate/internal/errs"
	"github.com/apk471/go-boilerplate/internal/model"
	"github.com/apk471/go-boilerplate/internal/repository"
)

type BookingService struct {
	movieService *MovieService
	bookingRepo  *repository.BookingRepository
}

func NewBookingService(movieService *MovieService, bookingRepo *repository.BookingRepository) *BookingService {
	return &BookingService{
		movieService: movieService,
		bookingRepo:  bookingRepo,
	}
}

func (s *BookingService) ListSeatStatuses(ctx context.Context, movieID string) ([]model.SeatStatus, error) {
	_, err := s.movieService.GetMovieByID(movieID)
	if err != nil {
		return nil, err
	}

	bookings, err := s.bookingRepo.ListBookings(ctx, movieID)
	if err != nil {
		return nil, err
	}

	statuses := make([]model.SeatStatus, 0, len(bookings))
	for _, booking := range bookings {
		statuses = append(statuses, model.SeatStatus{
			SeatID:    booking.SeatID,
			Booked:    booking.Status == model.BookingStatusHeld,
			Confirmed: booking.Status == model.BookingStatusConfirmed,
			UserID:    booking.UserID,
		})
	}

	return statuses, nil
}

func (s *BookingService) HoldSeat(ctx context.Context, movieID string, seatID string, userID string) (model.HoldSessionResponse, error) {
	movie, err := s.movieService.GetMovieByID(movieID)
	if err != nil {
		return model.HoldSessionResponse{}, err
	}

	if !isValidSeatID(movie, seatID) {
		return model.HoldSessionResponse{}, errs.NewBadRequestError("Seat does not exist for this movie", true, nil, nil, nil)
	}

	booking, err := s.bookingRepo.Hold(ctx, model.Booking{
		MovieID: movieID,
		SeatID:  seatID,
		UserID:  userID,
	})
	if err != nil {
		if errors.Is(err, repository.ErrSeatAlreadyBooked) {
			return model.HoldSessionResponse{}, errs.NewBadRequestError("Seat is already held or confirmed", true, nil, nil, nil)
		}

		return model.HoldSessionResponse{}, err
	}

	return model.HoldSessionResponse{
		SessionID: booking.ID,
		MovieID:   booking.MovieID,
		SeatID:    booking.SeatID,
		ExpiresAt: booking.ExpiresAt,
	}, nil
}

func (s *BookingService) ConfirmSession(ctx context.Context, sessionID string, userID string) (model.Booking, error) {
	booking, err := s.bookingRepo.Confirm(ctx, sessionID, userID)
	if err != nil {
		return model.Booking{}, mapBookingError(err)
	}

	return booking, nil
}

func (s *BookingService) ReleaseSession(ctx context.Context, sessionID string, userID string) error {
	if err := s.bookingRepo.Release(ctx, sessionID, userID); err != nil {
		return mapBookingError(err)
	}

	return nil
}

func mapBookingError(err error) error {
	switch {
	case errors.Is(err, repository.ErrSessionNotFound):
		return errs.NewNotFoundError("Session not found", true, nil)
	case errors.Is(err, repository.ErrSessionForbidden):
		return errs.NewForbiddenError("Session does not belong to this user", true)
	case errors.Is(err, repository.ErrSeatAlreadyBooked):
		return errs.NewBadRequestError("Seat is already held or confirmed", true, nil, nil, nil)
	default:
		return err
	}
}

func isValidSeatID(movie Movie, seatID string) bool {
	normalizedSeatID := strings.ToUpper(strings.TrimSpace(seatID))
	validSeatIDs := make([]string, 0, movie.Rows*movie.SeatsPerRow)
	rowLabels := "ABCDEFGHIJKLMNOPQRSTUVWXYZ"

	for rowIndex := 0; rowIndex < movie.Rows; rowIndex++ {
		rowLabel := string(rowLabels[rowIndex])
		for seatIndex := 1; seatIndex <= movie.SeatsPerRow; seatIndex++ {
			validSeatIDs = append(validSeatIDs, fmt.Sprintf("%s%d", rowLabel, seatIndex))
		}
	}

	return slices.Contains(validSeatIDs, normalizedSeatID)
}
