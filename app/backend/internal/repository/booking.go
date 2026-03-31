package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/apk471/go-boilerplate/internal/model"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const DefaultHoldTTL = 2 * time.Minute

var (
	ErrSeatAlreadyBooked = errors.New("seat already booked")
	ErrSessionNotFound   = errors.New("session not found")
	ErrSessionForbidden  = errors.New("session does not belong to user")
)

type BookingRepository struct {
	rdb *redis.Client
}

func NewBookingRepository(rdb *redis.Client) *BookingRepository {
	return &BookingRepository{rdb: rdb}
}

func (r *BookingRepository) Hold(ctx context.Context, booking model.Booking) (model.Booking, error) {
	sessionID := uuid.New().String()
	expiresAt := time.Now().Add(DefaultHoldTTL)
	seatKey := buildSeatKey(booking.MovieID, booking.SeatID)

	heldBooking := model.Booking{
		ID:        sessionID,
		MovieID:   booking.MovieID,
		SeatID:    booking.SeatID,
		UserID:    booking.UserID,
		Status:    model.BookingStatusHeld,
		ExpiresAt: expiresAt,
	}

	payload, err := json.Marshal(heldBooking)
	if err != nil {
		return model.Booking{}, err
	}

	result := r.rdb.SetArgs(ctx, seatKey, payload, redis.SetArgs{
		Mode: "NX",
		TTL:  DefaultHoldTTL,
	})
	if err := result.Err(); err != nil {
		return model.Booking{}, err
	}
	if result.Val() != "OK" {
		return model.Booking{}, ErrSeatAlreadyBooked
	}

	if err := r.rdb.Set(ctx, buildSessionKey(sessionID), seatKey, DefaultHoldTTL).Err(); err != nil {
		_ = r.rdb.Del(ctx, seatKey).Err()
		return model.Booking{}, err
	}

	return heldBooking, nil
}

func (r *BookingRepository) ListBookings(ctx context.Context, movieID string) ([]model.Booking, error) {
	pattern := buildSeatPattern(movieID)
	bookings := make([]model.Booking, 0)

	iter := r.rdb.Scan(ctx, 0, pattern, 0).Iterator()
	for iter.Next(ctx) {
		seatKey := iter.Val()

		payload, err := r.rdb.Get(ctx, seatKey).Result()
		if err != nil {
			continue
		}

		booking, err := parseBooking(payload)
		if err != nil {
			continue
		}

		ttl, err := r.rdb.TTL(ctx, seatKey).Result()
		if err == nil && ttl > 0 {
			booking.ExpiresAt = time.Now().Add(ttl)
		}

		bookings = append(bookings, booking)
	}

	if err := iter.Err(); err != nil {
		return nil, err
	}

	return bookings, nil
}

func (r *BookingRepository) Confirm(ctx context.Context, sessionID string, userID string) (model.Booking, error) {
	booking, seatKey, err := r.getSession(ctx, sessionID, userID)
	if err != nil {
		return model.Booking{}, err
	}

	booking.Status = model.BookingStatusConfirmed
	booking.ExpiresAt = time.Time{}

	payload, err := json.Marshal(booking)
	if err != nil {
		return model.Booking{}, err
	}

	if err := r.rdb.Set(ctx, seatKey, payload, 0).Err(); err != nil {
		return model.Booking{}, err
	}
	if err := r.rdb.Persist(ctx, seatKey).Err(); err != nil {
		return model.Booking{}, err
	}
	if err := r.rdb.Persist(ctx, buildSessionKey(sessionID)).Err(); err != nil {
		return model.Booking{}, err
	}

	return booking, nil
}

func (r *BookingRepository) Release(ctx context.Context, sessionID string, userID string) error {
	_, seatKey, err := r.getSession(ctx, sessionID, userID)
	if err != nil {
		return err
	}

	return r.rdb.Del(ctx, seatKey, buildSessionKey(sessionID)).Err()
}

func (r *BookingRepository) getSession(ctx context.Context, sessionID string, userID string) (model.Booking, string, error) {
	seatKey, err := r.rdb.Get(ctx, buildSessionKey(sessionID)).Result()
	if errors.Is(err, redis.Nil) {
		return model.Booking{}, "", ErrSessionNotFound
	}
	if err != nil {
		return model.Booking{}, "", err
	}

	payload, err := r.rdb.Get(ctx, seatKey).Result()
	if errors.Is(err, redis.Nil) {
		return model.Booking{}, "", ErrSessionNotFound
	}
	if err != nil {
		return model.Booking{}, "", err
	}

	booking, err := parseBooking(payload)
	if err != nil {
		return model.Booking{}, "", err
	}
	if booking.UserID != userID {
		return model.Booking{}, "", ErrSessionForbidden
	}

	return booking, seatKey, nil
}

func buildSeatKey(movieID string, seatID string) string {
	return fmt.Sprintf("seat:%s:%s", movieID, seatID)
}

func buildSeatPattern(movieID string) string {
	return fmt.Sprintf("seat:%s:*", movieID)
}

func buildSessionKey(sessionID string) string {
	return fmt.Sprintf("session:%s", sessionID)
}

func parseBooking(payload string) (model.Booking, error) {
	var booking model.Booking
	if err := json.Unmarshal([]byte(payload), &booking); err != nil {
		return model.Booking{}, err
	}

	return booking, nil
}
