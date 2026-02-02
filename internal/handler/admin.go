package handler

import (
	"net/http"

	"docko/templates/pages/admin"

	"github.com/labstack/echo/v4"
)

func (h *Handler) AdminDashboard(c echo.Context) error {
	return admin.Dashboard().Render(c.Request().Context(), c.Response().Writer)
}

func (h *Handler) Health(c echo.Context) error {
	return c.String(http.StatusOK, "OK")
}
