package handler

import (
	"net/http"
	"strconv"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/labstack/echo/v4"

	"docko/internal/ai"
	"docko/internal/database/sqlc"
	"docko/templates/pages/admin"
)

// AISettingsPage renders the AI configuration page
func (h *Handler) AISettingsPage(c echo.Context) error {
	ctx := c.Request().Context()

	// Get current settings
	settings, err := h.aiSvc.GetSettings(ctx)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to get ai settings")
	}

	// Get available providers
	providers := h.aiSvc.AvailableProviders()

	// Get usage stats
	stats, err := h.aiSvc.GetUsageStats(ctx)
	if err != nil {
		// Non-fatal - show page with empty stats
		stats = &ai.UsageStats{}
	}

	// Get pending suggestion count
	pendingCount, err := h.db.Queries.CountPendingSuggestions(ctx)
	if err != nil {
		pendingCount = 0
	}

	return admin.AISettings(settings, providers, stats, pendingCount).Render(ctx, c.Response().Writer)
}

// UpdateAISettings handles AI settings form submission
func (h *Handler) UpdateAISettings(c echo.Context) error {
	ctx := c.Request().Context()

	// Parse form values
	preferredProvider := c.FormValue("preferred_provider")
	maxPagesStr := c.FormValue("max_pages")
	autoProcess := c.FormValue("auto_process") == "on"
	autoApplyThresholdStr := c.FormValue("auto_apply_threshold")
	reviewThresholdStr := c.FormValue("review_threshold")

	// Parse numeric values
	maxPages, err := strconv.Atoi(maxPagesStr)
	if err != nil || maxPages < 1 || maxPages > 50 {
		maxPages = 5
	}

	autoApplyThreshold, err := strconv.ParseFloat(autoApplyThresholdStr, 64)
	if err != nil || autoApplyThreshold < 0 || autoApplyThreshold > 1 {
		autoApplyThreshold = 0.85
	}

	reviewThreshold, err := strconv.ParseFloat(reviewThresholdStr, 64)
	if err != nil || reviewThreshold < 0 || reviewThreshold > 1 {
		reviewThreshold = 0.50
	}

	// Prepare preferred provider
	var preferred *string
	if preferredProvider != "" && preferredProvider != "auto" {
		preferred = &preferredProvider
	}

	// Update settings
	_, err = h.aiSvc.UpdateSettings(ctx, sqlc.UpdateAISettingsParams{
		PreferredProvider:  preferred,
		MaxPages:           int32(maxPages),
		AutoProcess:        autoProcess,
		AutoApplyThreshold: numericFromFloat(autoApplyThreshold),
		ReviewThreshold:    numericFromFloat(reviewThreshold),
	})
	if err != nil {
		c.Response().Header().Set("HX-Trigger", `{"showToast": {"message": "Failed to update settings", "type": "error"}}`)
		return c.NoContent(http.StatusInternalServerError)
	}

	c.Response().Header().Set("HX-Trigger", `{"showToast": {"message": "AI settings updated", "type": "success"}}`)
	return c.Redirect(http.StatusSeeOther, "/ai")
}

// numericFromFloat converts float64 to pgtype.Numeric
func numericFromFloat(f float64) pgtype.Numeric {
	str := strconv.FormatFloat(f, 'f', 2, 64)
	var n pgtype.Numeric
	n.Scan(str)
	return n
}
