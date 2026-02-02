package handler

import (
	"docko/internal/config"
	"docko/internal/database"

	"github.com/labstack/echo/v4"
)

type Handler struct {
	cfg *config.Config
	db  *database.DB
}

func New(cfg *config.Config, db *database.DB) *Handler {
	return &Handler{
		cfg: cfg,
		db:  db,
	}
}

func (h *Handler) RegisterRoutes(e *echo.Echo) {
	// Static files
	e.Static("/static", "static")

	// Health check
	e.GET("/health", h.Health)

	// Public routes
	e.GET("/", h.Home)

	// Admin routes
	e.GET("/admin", h.AdminDashboard)
}
