package handler

import (
	"docko/templates/pages/admin"

	"github.com/labstack/echo/v4"
)

func (h *Handler) AdminDashboard(c echo.Context) error {
	return admin.Dashboard().Render(c.Request().Context(), c.Response().Writer)
}
