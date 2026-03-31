package router

import (
	"github.com/apk471/go-boilerplate/internal/handler"

	"github.com/labstack/echo/v4"
)

func registerSystemRoutes(r *echo.Echo, h *handler.Handlers) {
	r.GET("/status", h.Health.CheckHealth)
	r.GET("/movies", h.Movie.RegisterListMovies())
	r.GET("/movies/:movieID/seats", h.Booking.RegisterListSeats())
	r.POST("/movies/:movieID/seats/:seatID/hold", h.Booking.RegisterHoldSeat())
	r.PUT("/sessions/:sessionID/confirm", h.Booking.RegisterConfirmSession())
	r.DELETE("/sessions/:sessionID", h.Booking.RegisterReleaseSession())

	r.Static("/static", "static")

	r.GET("/docs", h.OpenAPI.ServeOpenAPIUI)
}
