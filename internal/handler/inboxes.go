package handler

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"docko/internal/database/sqlc"
	"docko/templates/pages/admin"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// countPDFsInDir counts .pdf files in a directory
func countPDFsInDir(dir string) int {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0 // Directory doesn't exist or can't be read
	}
	count := 0
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(strings.ToLower(e.Name()), ".pdf") {
			count++
		}
	}
	return count
}

// resolveErrorPath returns the error directory path for an inbox
func resolveErrorPath(inbox sqlc.Inbox) string {
	if inbox.ErrorPath != nil && *inbox.ErrorPath != "" {
		return *inbox.ErrorPath
	}
	return filepath.Join(inbox.Path, "errors")
}

// InboxesPage renders the inbox management page
func (h *Handler) InboxesPage(c echo.Context) error {
	ctx := c.Request().Context()

	inboxes, err := h.db.Queries.ListInboxes(ctx)
	if err != nil {
		slog.Error("failed to list inboxes", "error", err)
		return c.String(http.StatusInternalServerError, "Failed to load inboxes")
	}

	// Add error counts for each inbox
	inboxesWithCounts := make([]admin.InboxWithErrorCount, len(inboxes))
	for i, inbox := range inboxes {
		errorPath := resolveErrorPath(inbox)
		inboxesWithCounts[i] = admin.InboxWithErrorCount{
			Inbox:      inbox,
			ErrorCount: countPDFsInDir(errorPath),
		}
	}

	return admin.InboxesWithCounts(inboxesWithCounts).Render(ctx, c.Response().Writer)
}

// CreateInbox creates a new inbox directory
func (h *Handler) CreateInbox(c echo.Context) error {
	ctx := c.Request().Context()

	name := c.FormValue("name")
	path := c.FormValue("path")
	errorPath := c.FormValue("error_path")
	duplicateAction := c.FormValue("duplicate_action")

	// Validate required fields
	if name == "" || path == "" {
		return c.String(http.StatusBadRequest, "Name and path are required")
	}

	// Validate path exists and is a directory
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			// Try to create the directory
			if err := os.MkdirAll(path, 0755); err != nil {
				return c.String(http.StatusBadRequest, fmt.Sprintf("Cannot create directory: %v", err))
			}
		} else {
			return c.String(http.StatusBadRequest, fmt.Sprintf("Cannot access path: %v", err))
		}
	} else if !info.IsDir() {
		return c.String(http.StatusBadRequest, "Path must be a directory")
	}

	// Parse duplicate action
	var action sqlc.DuplicateAction
	switch duplicateAction {
	case "rename":
		action = sqlc.DuplicateActionRename
	case "skip":
		action = sqlc.DuplicateActionSkip
	default:
		action = sqlc.DuplicateActionDelete
	}

	// Create inbox in database
	var errorPathPtr *string
	if errorPath != "" {
		errorPathPtr = &errorPath
	}

	inbox, err := h.db.Queries.CreateInbox(ctx, sqlc.CreateInboxParams{
		Path:            path,
		Name:            name,
		ErrorPath:       errorPathPtr,
		DuplicateAction: action,
		Enabled:         true,
	})
	if err != nil {
		slog.Error("failed to create inbox", "error", err)
		return c.String(http.StatusInternalServerError, "Failed to create inbox")
	}

	// Add to watcher
	if err := h.inboxSvc.AddInbox(&inbox); err != nil {
		slog.Warn("failed to start watching inbox", "error", err)
		// Don't fail - inbox is created, just not watching yet
	}

	// Return the new inbox card for HTMX
	return admin.InboxCard(inbox).Render(ctx, c.Response().Writer)
}

// UpdateInbox updates inbox settings
func (h *Handler) UpdateInbox(c echo.Context) error {
	ctx := c.Request().Context()

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid inbox ID")
	}

	name := c.FormValue("name")
	path := c.FormValue("path")
	errorPath := c.FormValue("error_path")
	duplicateAction := c.FormValue("duplicate_action")
	enabled := c.FormValue("enabled") == "true"

	// Validate required fields
	if name == "" || path == "" {
		return c.String(http.StatusBadRequest, "Name and path are required")
	}

	// Parse duplicate action
	var action sqlc.DuplicateAction
	switch duplicateAction {
	case "rename":
		action = sqlc.DuplicateActionRename
	case "skip":
		action = sqlc.DuplicateActionSkip
	default:
		action = sqlc.DuplicateActionDelete
	}

	var errorPathPtr *string
	if errorPath != "" {
		errorPathPtr = &errorPath
	}

	// Get current inbox to check if path changed
	oldInbox, err := h.db.Queries.GetInbox(ctx, id)
	if err != nil {
		return c.String(http.StatusNotFound, "Inbox not found")
	}

	// Update inbox in database
	inbox, err := h.db.Queries.UpdateInbox(ctx, sqlc.UpdateInboxParams{
		ID:              id,
		Name:            name,
		Path:            path,
		ErrorPath:       errorPathPtr,
		DuplicateAction: action,
		Enabled:         enabled,
	})
	if err != nil {
		slog.Error("failed to update inbox", "error", err)
		return c.String(http.StatusInternalServerError, "Failed to update inbox")
	}

	// Update watcher if path changed or enabled status changed
	if oldInbox.Path != path || oldInbox.Enabled != enabled {
		// Remove old path from watcher
		if err := h.inboxSvc.RemoveInbox(id); err != nil {
			slog.Warn("failed to remove old inbox from watcher", "error", err)
		}
		// Add new path if enabled
		if enabled {
			if err := h.inboxSvc.AddInbox(&inbox); err != nil {
				slog.Warn("failed to add inbox to watcher", "error", err)
			}
		}
	}

	// Return updated inbox card
	return admin.InboxCard(inbox).Render(ctx, c.Response().Writer)
}

// DeleteInbox removes an inbox
func (h *Handler) DeleteInbox(c echo.Context) error {
	ctx := c.Request().Context()

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid inbox ID")
	}

	// Remove from watcher first
	if err := h.inboxSvc.RemoveInbox(id); err != nil {
		slog.Warn("failed to remove inbox from watcher", "error", err)
	}

	// Delete from database
	if err := h.db.Queries.DeleteInbox(ctx, id); err != nil {
		slog.Error("failed to delete inbox", "error", err)
		return c.String(http.StatusInternalServerError, "Failed to delete inbox")
	}

	// Return empty response for HTMX to remove the element
	return c.String(http.StatusOK, "")
}

// ToggleInbox enables/disables an inbox
func (h *Handler) ToggleInbox(c echo.Context) error {
	ctx := c.Request().Context()

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid inbox ID")
	}

	// Get current inbox
	inbox, err := h.db.Queries.GetInbox(ctx, id)
	if err != nil {
		return c.String(http.StatusNotFound, "Inbox not found")
	}

	// Toggle enabled state
	newEnabled := !inbox.Enabled

	// Update in database
	updatedInbox, err := h.db.Queries.UpdateInbox(ctx, sqlc.UpdateInboxParams{
		ID:              id,
		Name:            inbox.Name,
		Path:            inbox.Path,
		ErrorPath:       inbox.ErrorPath,
		DuplicateAction: inbox.DuplicateAction,
		Enabled:         newEnabled,
	})
	if err != nil {
		slog.Error("failed to toggle inbox", "error", err)
		return c.String(http.StatusInternalServerError, "Failed to toggle inbox")
	}

	// Update watcher
	if newEnabled {
		if err := h.inboxSvc.AddInbox(&updatedInbox); err != nil {
			slog.Warn("failed to add inbox to watcher", "error", err)
		}
	} else {
		if err := h.inboxSvc.RemoveInbox(id); err != nil {
			slog.Warn("failed to remove inbox from watcher", "error", err)
		}
	}

	// Return updated inbox card
	return admin.InboxCard(updatedInbox).Render(ctx, c.Response().Writer)
}

// InboxEvents returns recent events for an inbox
func (h *Handler) InboxEvents(c echo.Context) error {
	ctx := c.Request().Context()

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid inbox ID")
	}

	events, err := h.db.Queries.ListInboxEvents(ctx, sqlc.ListInboxEventsParams{
		InboxID: id,
		Limit:   10,
	})
	if err != nil {
		slog.Error("failed to list inbox events", "error", err)
		return c.String(http.StatusInternalServerError, "Failed to load events")
	}

	return admin.InboxEventsList(events).Render(ctx, c.Response().Writer)
}
