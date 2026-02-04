package middleware

import (
	"context"
	"net/http"

	"github.com/bketelsen/docko/internal/auth"
	"github.com/bketelsen/docko/internal/ctxkeys"

	"github.com/labstack/echo/v4"
)

const SessionCookieName = "admin_session"

// RequireAuth middleware protects routes that require authentication
func RequireAuth(authService *auth.Service) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			cookie, err := c.Cookie(SessionCookieName)
			if err != nil || cookie.Value == "" {
				return c.Redirect(http.StatusSeeOther, "/login")
			}

			session, err := authService.ValidateSession(c.Request().Context(), cookie.Value)
			if err != nil {
				// Invalid/expired session, clear cookie and redirect
				clearSessionCookie(c)
				return c.Redirect(http.StatusSeeOther, "/login")
			}

			// Add user info to context
			ctx := context.WithValue(c.Request().Context(), ctxkeys.AdminUser, session.Username)
			c.SetRequest(c.Request().WithContext(ctx))

			return next(c)
		}
	}
}

func clearSessionCookie(c echo.Context) {
	c.SetCookie(&http.Cookie{
		Name:     SessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   c.Request().TLS != nil,
		SameSite: http.SameSiteLaxMode,
	})
}
