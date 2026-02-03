package handler

import (
	"log/slog"
	"net/http"
	"strings"

	"docko/internal/database/sqlc"
	"docko/templates/pages/admin"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// CorrespondentsPage renders the correspondent management page
func (h *Handler) CorrespondentsPage(c echo.Context) error {
	ctx := c.Request().Context()

	correspondents, err := h.db.Queries.ListCorrespondentsWithCounts(ctx)
	if err != nil {
		slog.Error("failed to list correspondents", "error", err)
		return c.String(http.StatusInternalServerError, "Failed to load correspondents")
	}

	return admin.Correspondents(correspondents).Render(ctx, c.Response().Writer)
}

// CreateCorrespondent creates a new correspondent
func (h *Handler) CreateCorrespondent(c echo.Context) error {
	ctx := c.Request().Context()

	name := strings.TrimSpace(c.FormValue("name"))
	notes := strings.TrimSpace(c.FormValue("notes"))

	// Validate required fields
	if name == "" {
		return c.String(http.StatusBadRequest, "Name is required")
	}

	// Prepare notes (nil if empty)
	var notesPtr *string
	if notes != "" {
		notesPtr = &notes
	}

	correspondent, err := h.db.Queries.CreateCorrespondent(ctx, sqlc.CreateCorrespondentParams{
		Name:  name,
		Notes: notesPtr,
	})
	if err != nil {
		slog.Error("failed to create correspondent", "error", err)
		return c.String(http.StatusInternalServerError, "Failed to create correspondent")
	}

	// Return the new correspondent card for HTMX
	return admin.CorrespondentCard(sqlc.ListCorrespondentsWithCountsRow{
		ID:            correspondent.ID,
		Name:          correspondent.Name,
		Notes:         correspondent.Notes,
		CreatedAt:     correspondent.CreatedAt,
		DocumentCount: 0,
	}).Render(ctx, c.Response().Writer)
}

// UpdateCorrespondent updates an existing correspondent
func (h *Handler) UpdateCorrespondent(c echo.Context) error {
	ctx := c.Request().Context()

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid correspondent ID")
	}

	name := strings.TrimSpace(c.FormValue("name"))
	notes := strings.TrimSpace(c.FormValue("notes"))

	// Validate required fields
	if name == "" {
		return c.String(http.StatusBadRequest, "Name is required")
	}

	// Prepare notes (nil if empty)
	var notesPtr *string
	if notes != "" {
		notesPtr = &notes
	}

	correspondent, err := h.db.Queries.UpdateCorrespondent(ctx, sqlc.UpdateCorrespondentParams{
		ID:    id,
		Name:  name,
		Notes: notesPtr,
	})
	if err != nil {
		slog.Error("failed to update correspondent", "error", err)
		return c.String(http.StatusInternalServerError, "Failed to update correspondent")
	}

	// Get document count for the updated correspondent
	correspondents, err := h.db.Queries.ListCorrespondentsWithCounts(ctx)
	if err != nil {
		slog.Error("failed to get correspondent counts", "error", err)
		return c.String(http.StatusInternalServerError, "Failed to get correspondent data")
	}

	// Find the document count for this correspondent
	var documentCount int32 = 0
	for _, c := range correspondents {
		if c.ID == correspondent.ID {
			documentCount = c.DocumentCount
			break
		}
	}

	// Return updated correspondent card
	return admin.CorrespondentCard(sqlc.ListCorrespondentsWithCountsRow{
		ID:            correspondent.ID,
		Name:          correspondent.Name,
		Notes:         correspondent.Notes,
		CreatedAt:     correspondent.CreatedAt,
		DocumentCount: documentCount,
	}).Render(ctx, c.Response().Writer)
}

// DeleteCorrespondent removes a correspondent
func (h *Handler) DeleteCorrespondent(c echo.Context) error {
	ctx := c.Request().Context()

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid correspondent ID")
	}

	// Delete from database (document_correspondents cascade handled by FK)
	if err := h.db.Queries.DeleteCorrespondent(ctx, id); err != nil {
		slog.Error("failed to delete correspondent", "error", err)
		return c.String(http.StatusInternalServerError, "Failed to delete correspondent")
	}

	// Return empty response for HTMX to remove the element
	return c.String(http.StatusOK, "")
}
