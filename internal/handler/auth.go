package handler

import (
	"log/slog"
	"net/http"

	"docko/internal/middleware"
	"docko/templates/pages/admin"

	"github.com/labstack/echo/v4"
)

// LoginPage renders the login form
func (h *Handler) LoginPage(c echo.Context) error {
	// If already logged in, redirect to dashboard
	if cookie, err := c.Cookie(middleware.SessionCookieName); err == nil && cookie.Value != "" {
		if _, err := h.auth.ValidateSession(c.Request().Context(), cookie.Value); err == nil {
			return c.Redirect(http.StatusSeeOther, "/")
		}
	}

	errorMsg := c.QueryParam("error")
	return admin.Login(errorMsg).Render(c.Request().Context(), c.Response().Writer)
}

// Login handles the login form submission
func (h *Handler) Login(c echo.Context) error {
	username := c.FormValue("username")
	password := c.FormValue("password")

	if username == "" || password == "" {
		return c.Redirect(http.StatusSeeOther, "/login?error=Please+enter+username+and+password")
	}

	user, err := h.auth.ValidateCredentials(c.Request().Context(), username, password)
	if err != nil {
		slog.Warn("failed login attempt", "username", username, "ip", c.RealIP())
		return c.Redirect(http.StatusSeeOther, "/login?error=Invalid+username+or+password")
	}

	token, err := h.auth.CreateSession(c.Request().Context(), user.ID)
	if err != nil {
		slog.Error("failed to create session", "error", err)
		return c.Redirect(http.StatusSeeOther, "/login?error=Login+failed.+Please+try+again.")
	}

	// Set session cookie
	c.SetCookie(&http.Cookie{
		Name:     middleware.SessionCookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   h.cfg.Auth.SessionMaxAge * 3600, // Convert hours to seconds
		HttpOnly: true,
		Secure:   h.cfg.IsProduction(),
		SameSite: http.SameSiteLaxMode,
	})

	slog.Info("admin login successful", "username", username, "ip", c.RealIP())
	return c.Redirect(http.StatusSeeOther, "/")
}

// Logout handles logout
func (h *Handler) Logout(c echo.Context) error {
	cookie, err := c.Cookie(middleware.SessionCookieName)
	if err == nil && cookie.Value != "" {
		_ = h.auth.DeleteSession(c.Request().Context(), cookie.Value)
	}

	// Clear cookie
	c.SetCookie(&http.Cookie{
		Name:     middleware.SessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   h.cfg.IsProduction(),
		SameSite: http.SameSiteLaxMode,
	})

	return c.Redirect(http.StatusSeeOther, "/login")
}
