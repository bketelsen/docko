package handler

import (
	"bytes"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"docko/templates/partials"

	"github.com/labstack/echo/v4"
)

// ProcessingStatus handles SSE connections for processing status updates
// IMPORTANT: Sends rendered HTML partials, not JSON - HTMX SSE extension expects HTML
func (h *Handler) ProcessingStatus(c echo.Context) error {
	// Check if broadcaster is available
	if h.broadcaster == nil {
		slog.Error("status broadcaster not initialized")
		return echo.NewHTTPError(http.StatusServiceUnavailable, "SSE not available")
	}

	resp := c.Response()
	resp.Header().Set("Content-Type", "text/event-stream")
	resp.Header().Set("Cache-Control", "no-cache")
	resp.Header().Set("Connection", "keep-alive")
	resp.Header().Set("X-Accel-Buffering", "no") // Disable nginx buffering

	ctx := c.Request().Context()

	updates := h.broadcaster.Subscribe(ctx)
	if updates == nil {
		return echo.NewHTTPError(http.StatusServiceUnavailable, "too many SSE connections")
	}
	// Note: Unsubscribe is handled automatically by the broadcaster when context is cancelled

	// Get the underlying ResponseWriter for flushing
	w := resp.Writer
	flusher, ok := w.(http.Flusher)
	if !ok {
		return echo.NewHTTPError(http.StatusInternalServerError, "SSE not supported")
	}

	// Send initial connection established event
	fmt.Fprintf(w, "event: connected\ndata: ok\n\n")
	flusher.Flush()

	heartbeat := time.NewTicker(30 * time.Second)
	defer heartbeat.Stop()

	slog.Debug("SSE connection established", "remote", c.RealIP())

	for {
		select {
		case <-ctx.Done():
			slog.Debug("SSE connection closed", "remote", c.RealIP())
			return nil

		case <-heartbeat.C:
			// Send heartbeat to keep connection alive
			fmt.Fprintf(w, "event: heartbeat\ndata: ping\n\n")
			flusher.Flush()

		case update, ok := <-updates:
			if !ok {
				// Channel closed
				return nil
			}

			// CRITICAL: Render HTML partial, not JSON
			// HTMX SSE extension expects HTML content for swap
			var buf bytes.Buffer
			if err := partials.DocumentStatus(
				update.DocumentID.String(),
				update.Status,
				update.Error,
			).Render(ctx, &buf); err != nil {
				slog.Error("failed to render status partial",
					"doc_id", update.DocumentID,
					"error", err)
				continue
			}

			// Event name matches sse-swap target: doc-{id}
			// The HTML content goes on multiple data: lines if needed
			htmlContent := buf.String()
			fmt.Fprintf(w, "event: doc-%s\ndata: %s\n\n",
				update.DocumentID.String(),
				htmlContent,
			)
			flusher.Flush()

			slog.Debug("SSE status update sent",
				"doc_id", update.DocumentID,
				"status", update.Status)
		}
	}
}
