package model

import "time"

type BookingStatus string

const (
	BookingStatusHeld      BookingStatus = "held"
	BookingStatusConfirmed BookingStatus = "confirmed"
)

type Booking struct {
	ID        string        `json:"id"`
	MovieID   string        `json:"movie_id"`
	SeatID    string        `json:"seat_id"`
	UserID    string        `json:"user_id"`
	Status    BookingStatus `json:"status"`
	ExpiresAt time.Time     `json:"expires_at,omitempty"`
}

type SeatStatus struct {
	SeatID    string `json:"seat_id"`
	Booked    bool   `json:"booked"`
	Confirmed bool   `json:"confirmed"`
	UserID    string `json:"user_id,omitempty"`
}

type HoldSessionResponse struct {
	SessionID string    `json:"session_id"`
	MovieID   string    `json:"movie_id"`
	SeatID    string    `json:"seat_id"`
	ExpiresAt time.Time `json:"expires_at"`
}
