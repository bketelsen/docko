package handler

import (
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/labstack/echo/v4"

	"docko/internal/ai"
	"docko/internal/database/sqlc"
	"docko/internal/processing"
	"docko/templates/pages/admin"
	"docko/templates/partials"
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
	minWordCountStr := c.FormValue("min_word_count")

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

	minWordCount, err := strconv.Atoi(minWordCountStr)
	if err != nil || minWordCount < 0 || minWordCount > 10000 {
		minWordCount = 0
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
		MinWordCount:       int32(minWordCount),
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

// ReviewQueuePage renders the suggestion review queue
func (h *Handler) ReviewQueuePage(c echo.Context) error {
	ctx := c.Request().Context()

	// Parse pagination
	page, _ := strconv.Atoi(c.QueryParam("page"))
	if page < 1 {
		page = 1
	}
	limit := int64(20)
	offset := int64(page-1) * limit

	// Get pending suggestions
	suggestions, err := h.db.Queries.ListPendingSuggestions(ctx, sqlc.ListPendingSuggestionsParams{
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to list suggestions")
	}

	// Get total count
	totalCount, _ := h.db.Queries.CountPendingSuggestions(ctx)

	return admin.ReviewQueue(suggestions, page, int(totalCount), int(limit)).Render(ctx, c.Response().Writer)
}

// AcceptSuggestion handles accepting a suggestion
func (h *Handler) AcceptSuggestion(c echo.Context) error {
	ctx := c.Request().Context()

	suggestionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid suggestion id")
	}

	// Get suggestion
	suggestion, err := h.db.Queries.GetAISuggestion(ctx, suggestionID)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "suggestion not found")
	}

	// Apply the suggestion
	sug := ai.Suggestion{
		Type:       string(suggestion.SuggestionType),
		Value:      suggestion.Value,
		Confidence: 1.0, // Accepted by user
		IsNew:      suggestion.IsNew,
	}
	if err := h.aiSvc.ApplySuggestionManual(ctx, suggestion.DocumentID, sug); err != nil {
		c.Response().Header().Set("HX-Trigger", `{"showToast": {"message": "Failed to apply suggestion", "type": "error"}}`)
		return c.NoContent(http.StatusInternalServerError)
	}

	// Mark as accepted
	_, err = h.db.Queries.AcceptSuggestion(ctx, suggestionID)
	if err != nil {
		c.Response().Header().Set("HX-Trigger", `{"showToast": {"message": "Failed to update suggestion status", "type": "error"}}`)
		return c.NoContent(http.StatusInternalServerError)
	}

	c.Response().Header().Set("HX-Trigger", `{"showToast": {"message": "Suggestion applied", "type": "success"}}`)
	return c.String(http.StatusOK, "") // Return empty to remove the row
}

// RejectSuggestion handles rejecting a suggestion
func (h *Handler) RejectSuggestion(c echo.Context) error {
	ctx := c.Request().Context()

	suggestionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid suggestion id")
	}

	_, err = h.db.Queries.RejectSuggestion(ctx, suggestionID)
	if err != nil {
		c.Response().Header().Set("HX-Trigger", `{"showToast": {"message": "Failed to reject suggestion", "type": "error"}}`)
		return c.NoContent(http.StatusInternalServerError)
	}

	c.Response().Header().Set("HX-Trigger", `{"showToast": {"message": "Suggestion rejected", "type": "success"}}`)
	return c.String(http.StatusOK, "") // Return empty to remove the row
}

// QueueDashboardPage renders the queue status dashboard
func (h *Handler) QueueDashboardPage(c echo.Context) error {
	ctx := c.Request().Context()

	// Get queue stats
	stats, err := h.db.Queries.GetQueueStats(ctx)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to get queue stats")
	}

	return admin.QueueDashboard(stats).Render(ctx, c.Response().Writer)
}

// RetryJob handles retrying a failed job
func (h *Handler) RetryJob(c echo.Context) error {
	ctx := c.Request().Context()

	jobID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid job id")
	}

	_, err = h.db.Queries.ResetJobForRetry(ctx, jobID)
	if err != nil {
		c.Response().Header().Set("HX-Trigger", `{"showToast": {"message": "Failed to retry job", "type": "error"}}`)
		return c.NoContent(http.StatusInternalServerError)
	}

	c.Response().Header().Set("HX-Trigger", `{"showToast": {"message": "Job queued for retry", "type": "success"}}`)
	return c.Redirect(http.StatusSeeOther, "/queues")
}

// RetryAllFailedJobs handles retrying all failed jobs
func (h *Handler) RetryAllFailedJobs(c echo.Context) error {
	ctx := c.Request().Context()

	err := h.db.Queries.ResetAllFailedJobs(ctx)
	if err != nil {
		c.Response().Header().Set("HX-Trigger", `{"showToast": {"message": "Failed to retry jobs", "type": "error"}}`)
		return c.NoContent(http.StatusInternalServerError)
	}

	c.Response().Header().Set("HX-Trigger", `{"showToast": {"message": "All failed jobs queued for retry", "type": "success"}}`)
	return c.Redirect(http.StatusSeeOther, "/queues")
}

// ReanalyzeDocument triggers AI analysis for a document
func (h *Handler) ReanalyzeDocument(c echo.Context) error {
	ctx := c.Request().Context()

	docID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid document id")
	}

	// Check document exists
	doc, err := h.db.Queries.GetDocument(ctx, docID)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "document not found")
	}

	// Check document has text content
	if doc.TextContent == nil || *doc.TextContent == "" {
		c.Response().Header().Set("HX-Trigger", `{"showToast": {"message": "Document has no text content", "type": "error"}}`)
		return h.returnSuggestionsPartial(c, docID)
	}

	// Delete existing pending suggestions for this document
	err = h.db.Queries.DeleteDocumentSuggestions(ctx, docID)
	if err != nil {
		slog.Warn("failed to delete existing suggestions", "error", err)
		// Continue anyway
	}

	// Enqueue AI analysis job
	payload := processing.AIPayload{DocumentID: docID}
	_, err = h.queue.Enqueue(ctx, processing.QueueAI, processing.JobTypeAI, payload)
	if err != nil {
		c.Response().Header().Set("HX-Trigger", `{"showToast": {"message": "Failed to queue analysis", "type": "error"}}`)
		return h.returnSuggestionsPartial(c, docID)
	}

	c.Response().Header().Set("HX-Trigger", `{"showToast": {"message": "Analysis queued", "type": "success"}}`)
	return h.returnSuggestionsPartial(c, docID)
}

// returnSuggestionsPartial returns the AI suggestions partial for a document
func (h *Handler) returnSuggestionsPartial(c echo.Context, docID uuid.UUID) error {
	ctx := c.Request().Context()

	// Check if AI is enabled (has available providers)
	aiEnabled := len(h.aiSvc.AvailableProviders()) > 0

	// Get pending suggestions
	suggestions, err := h.db.Queries.ListPendingSuggestionsForDocument(ctx, docID)
	if err != nil {
		suggestions = []sqlc.AiSuggestion{}
	}

	return partials.AISuggestions(docID, suggestions, aiEnabled).Render(ctx, c.Response().Writer)
}

// QueueDetails returns the expanded content for a queue section
// Called via HTMX lazy loading when user expands a queue
func (h *Handler) QueueDetails(c echo.Context) error {
	ctx := c.Request().Context()
	queueName := c.Param("name")

	// Get failed jobs with document info (limit 20)
	failedJobs, err := h.db.Queries.GetFailedJobsForQueue(ctx, sqlc.GetFailedJobsForQueueParams{
		QueueName: queueName,
		Limit:     20,
		Offset:    0,
	})
	if err != nil {
		failedJobs = []sqlc.GetFailedJobsForQueueRow{}
	}

	// Get recent completed jobs (last 24h, limit 20)
	recentJobs, err := h.db.Queries.GetRecentCompletedJobsForQueue(ctx, sqlc.GetRecentCompletedJobsForQueueParams{
		QueueName: queueName,
		Limit:     20,
		Offset:    0,
	})
	if err != nil {
		recentJobs = []sqlc.GetRecentCompletedJobsForQueueRow{}
	}

	return admin.QueueDetailContent(queueName, failedJobs, recentJobs).Render(ctx, c.Response().Writer)
}

// DismissJob marks a single failed job as dismissed
func (h *Handler) DismissJob(c echo.Context) error {
	ctx := c.Request().Context()

	jobID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid job id")
	}

	_, err = h.db.Queries.DismissJob(ctx, jobID)
	if err != nil {
		c.Response().Header().Set("HX-Trigger", `{"showToast": {"message": "Failed to dismiss job", "type": "error"}}`)
		return c.NoContent(http.StatusInternalServerError)
	}

	c.Response().Header().Set("HX-Trigger", `{"showToast": {"message": "Job dismissed", "type": "success"}}`)
	// Return empty string to remove the row via outerHTML swap
	return c.String(http.StatusOK, "")
}

// RetryQueueJobs retries all failed jobs in a specific queue
func (h *Handler) RetryQueueJobs(c echo.Context) error {
	ctx := c.Request().Context()
	queueName := c.Param("name")

	count, err := h.db.Queries.ResetFailedJobsForQueue(ctx, queueName)
	if err != nil {
		c.Response().Header().Set("HX-Trigger", `{"showToast": {"message": "Failed to retry jobs", "type": "error"}}`)
		return c.NoContent(http.StatusInternalServerError)
	}

	msg := fmt.Sprintf("%d job(s) queued for retry", count)
	c.Response().Header().Set("HX-Trigger", fmt.Sprintf(`{"showToast": {"message": "%s", "type": "success"}}`, msg))
	return c.Redirect(http.StatusSeeOther, "/queues")
}

// ClearQueueJobs dismisses all failed jobs in a specific queue
func (h *Handler) ClearQueueJobs(c echo.Context) error {
	ctx := c.Request().Context()
	queueName := c.Param("name")

	count, err := h.db.Queries.DismissFailedJobsForQueue(ctx, queueName)
	if err != nil {
		c.Response().Header().Set("HX-Trigger", `{"showToast": {"message": "Failed to clear jobs", "type": "error"}}`)
		return c.NoContent(http.StatusInternalServerError)
	}

	msg := fmt.Sprintf("%d job(s) dismissed", count)
	c.Response().Header().Set("HX-Trigger", fmt.Sprintf(`{"showToast": {"message": "%s", "type": "success"}}`, msg))
	return c.Redirect(http.StatusSeeOther, "/queues")
}
