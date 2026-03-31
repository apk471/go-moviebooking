package handler

import (
	"net/http"
	"strings"

	"github.com/apk471/go-boilerplate/internal/server"
	"github.com/apk471/go-boilerplate/internal/service"
	"github.com/apk471/go-boilerplate/internal/validation"
	"github.com/labstack/echo/v4"
)

type listSeatsRequest struct {
	MovieID string `param:"movieID"`
}

func (r listSeatsRequest) Validate() error {
	if strings.TrimSpace(r.MovieID) == "" {
		return validation.CustomValidationErrors{
			{Field: "movie_id", Message: "is required"},
		}
	}

	return nil
}

type holdSeatRequest struct {
	MovieID string `param:"movieID"`
	SeatID  string `param:"seatID"`
	UserID  string `json:"user_id"`
}

func (r holdSeatRequest) Validate() error {
	validationErrors := validation.CustomValidationErrors{}
	if strings.TrimSpace(r.MovieID) == "" {
		validationErrors = append(validationErrors, validation.CustomValidationError{Field: "movie_id", Message: "is required"})
	}
	if strings.TrimSpace(r.SeatID) == "" {
		validationErrors = append(validationErrors, validation.CustomValidationError{Field: "seat_id", Message: "is required"})
	}
	if strings.TrimSpace(r.UserID) == "" {
		validationErrors = append(validationErrors, validation.CustomValidationError{Field: "user_id", Message: "is required"})
	}
	if len(validationErrors) > 0 {
		return validationErrors
	}

	return nil
}

type sessionRequest struct {
	SessionID string `param:"sessionID"`
	UserID    string `json:"user_id"`
}

func (r sessionRequest) Validate() error {
	validationErrors := validation.CustomValidationErrors{}
	if strings.TrimSpace(r.SessionID) == "" {
		validationErrors = append(validationErrors, validation.CustomValidationError{Field: "session_id", Message: "is required"})
	}
	if strings.TrimSpace(r.UserID) == "" {
		validationErrors = append(validationErrors, validation.CustomValidationError{Field: "user_id", Message: "is required"})
	}
	if len(validationErrors) > 0 {
		return validationErrors
	}

	return nil
}

type BookingHandler struct {
	Handler
	bookingService *service.BookingService
}

func NewBookingHandler(s *server.Server, bookingService *service.BookingService) *BookingHandler {
	return &BookingHandler{
		Handler:        NewHandler(s),
		bookingService: bookingService,
	}
}

func (h *BookingHandler) ListSeats(c echo.Context, req *listSeatsRequest) (interface{}, error) {
	return h.bookingService.ListSeatStatuses(c.Request().Context(), req.MovieID)
}

func (h *BookingHandler) HoldSeat(c echo.Context, req *holdSeatRequest) (interface{}, error) {
	return h.bookingService.HoldSeat(c.Request().Context(), req.MovieID, strings.ToUpper(req.SeatID), req.UserID)
}

func (h *BookingHandler) ConfirmSession(c echo.Context, req *sessionRequest) (interface{}, error) {
	return h.bookingService.ConfirmSession(c.Request().Context(), req.SessionID, req.UserID)
}

func (h *BookingHandler) ReleaseSession(c echo.Context, req *sessionRequest) error {
	return h.bookingService.ReleaseSession(c.Request().Context(), req.SessionID, req.UserID)
}

func (h *BookingHandler) RegisterListSeats() echo.HandlerFunc {
	return Handle(h.Handler, h.ListSeats, http.StatusOK, &listSeatsRequest{})
}

func (h *BookingHandler) RegisterHoldSeat() echo.HandlerFunc {
	return Handle(h.Handler, h.HoldSeat, http.StatusCreated, &holdSeatRequest{})
}

func (h *BookingHandler) RegisterConfirmSession() echo.HandlerFunc {
	return Handle(h.Handler, h.ConfirmSession, http.StatusOK, &sessionRequest{})
}

func (h *BookingHandler) RegisterReleaseSession() echo.HandlerFunc {
	return HandleNoContent(h.Handler, h.ReleaseSession, http.StatusNoContent, &sessionRequest{})
}
