package handler

import (
	"context"
	"net/http"

	"docko/internal/database/sqlc"
	"docko/templates/pages/admin"

	"github.com/labstack/echo/v4"
)

// DashboardData holds all stats for the main dashboard
type DashboardData struct {
	// Documents section
	Documents struct {
		Total     int32
		Processed int32
		Pending   int32
		Failed    int32
		Today     int32
	}
	TagCount           int32
	CorrespondentCount int32

	// Processing section
	Processing struct {
		Pending    int32
		Processing int32
		Completed  int32
		Failed     int32
		Health     string // "healthy", "warning", "issues"
	}
	PendingSuggestions int32
	RecentJobs         []sqlc.Job
	ActiveProvider     string
	JobsToday          int32

	// Sources section
	Inboxes struct {
		Total   int32
		Enabled int32
	}
	NetworkSources struct {
		Total   int32
		Enabled int32
	}
}

// calculateQueueHealth returns health status based on queue state
func calculateQueueHealth(pending, failed int32) string {
	if failed > 0 {
		return "issues"
	}
	if pending >= 10 {
		return "warning"
	}
	return "healthy"
}

// getActiveProvider returns the configured AI provider name
func (h *Handler) getActiveProvider(ctx context.Context) string {
	settings, err := h.db.Queries.GetAISettings(ctx)
	if err != nil {
		return "None"
	}
	if settings.PreferredProvider != nil && *settings.PreferredProvider != "" {
		return *settings.PreferredProvider
	}
	return "Auto"
}

func (h *Handler) AdminDashboard(c echo.Context) error {
	ctx := c.Request().Context()
	data := DashboardData{}

	// Documents section
	if docStats, err := h.db.Queries.GetDashboardDocumentStats(ctx); err == nil {
		data.Documents.Total = docStats.Total
		data.Documents.Processed = docStats.Processed
		data.Documents.Pending = docStats.Pending
		data.Documents.Failed = docStats.Failed
		data.Documents.Today = docStats.Today
	}

	if tagCount, err := h.db.Queries.CountTags(ctx); err == nil {
		data.TagCount = tagCount
	}

	if corrCount, err := h.db.Queries.CountCorrespondents(ctx); err == nil {
		data.CorrespondentCount = corrCount
	}

	// Processing section
	if queueStats, err := h.db.Queries.GetDashboardQueueStats(ctx); err == nil {
		data.Processing.Pending = queueStats.Pending
		data.Processing.Processing = queueStats.Processing
		data.Processing.Completed = queueStats.Completed
		data.Processing.Failed = queueStats.Failed
		data.Processing.Health = calculateQueueHealth(queueStats.Pending, queueStats.Failed)
	}

	if pendingSugg, err := h.db.Queries.CountPendingSuggestions(ctx); err == nil {
		data.PendingSuggestions = int32(pendingSugg)
	}

	if recentJobs, err := h.db.Queries.GetRecentJobs(ctx, 5); err == nil {
		data.RecentJobs = recentJobs
	}

	data.ActiveProvider = h.getActiveProvider(ctx)

	if jobsToday, err := h.db.Queries.GetDashboardJobsToday(ctx); err == nil {
		data.JobsToday = jobsToday
	}

	// Sources section
	if sourceStats, err := h.db.Queries.GetDashboardSourceStats(ctx); err == nil {
		data.Inboxes.Total = sourceStats.InboxTotal
		data.Inboxes.Enabled = sourceStats.InboxEnabled
		data.NetworkSources.Total = sourceStats.NetworkTotal
		data.NetworkSources.Enabled = sourceStats.NetworkEnabled
	}

	// TODO: Plan 03 will update template to accept DashboardData
	_ = data // data ready for Plan 03
	return admin.Dashboard().Render(ctx, c.Response().Writer)
}

func (h *Handler) Health(c echo.Context) error {
	return c.String(http.StatusOK, "OK")
}
