package handler

import (
	"docko/internal/auth"
	"docko/internal/config"
	"docko/internal/database"
	"docko/internal/middleware"

	"github.com/labstack/echo/v4"
)

type Handler struct {
	cfg  *config.Config
	db   *database.DB
	auth *auth.Service
}

func New(cfg *config.Config, db *database.DB, authService *auth.Service) *Handler {
	return &Handler{
		cfg:  cfg,
		db:   db,
		auth: authService,
	}
}

func (h *Handler) RegisterRoutes(e *echo.Echo) {
	// Static files
	e.Static("/static", "static")
	e.Static("/assets", "assets")

	// Health check
	e.GET("/health", h.Health)

	// Auth routes (no middleware)
	e.GET("/login", h.LoginPage)
	e.POST("/login", h.Login)
	e.POST("/logout", h.Logout)

	// Protected routes (dashboard at root)
	e.GET("/", h.AdminDashboard, middleware.RequireAuth(h.auth))
}
