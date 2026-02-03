package handler

import (
	"log/slog"
	"net/http"

	"docko/internal/database/sqlc"
	"docko/templates/pages/admin"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// Valid tag colors (Tailwind color names)
var validColors = map[string]bool{
	"red":     true,
	"orange":  true,
	"amber":   true,
	"yellow":  true,
	"green":   true,
	"emerald": true,
	"teal":    true,
	"blue":    true,
	"indigo":  true,
	"purple":  true,
	"pink":    true,
	"gray":    true,
}

// validateColor returns the color if valid, otherwise "blue" as default
func validateColor(color string) string {
	if validColors[color] {
		return color
	}
	return "blue"
}

// TagsPage renders the tag management page
func (h *Handler) TagsPage(c echo.Context) error {
	ctx := c.Request().Context()

	tags, err := h.db.Queries.ListTagsWithCounts(ctx)
	if err != nil {
		slog.Error("failed to list tags", "error", err)
		return c.String(http.StatusInternalServerError, "Failed to load tags")
	}

	return admin.Tags(tags).Render(ctx, c.Response().Writer)
}

// CreateTag creates a new tag
func (h *Handler) CreateTag(c echo.Context) error {
	ctx := c.Request().Context()

	name := c.FormValue("name")
	color := validateColor(c.FormValue("color"))

	// Validate required fields
	if name == "" {
		return c.String(http.StatusBadRequest, "Name is required")
	}

	// Create tag in database
	colorPtr := &color
	tag, err := h.db.Queries.CreateTag(ctx, sqlc.CreateTagParams{
		Name:  name,
		Color: colorPtr,
	})
	if err != nil {
		// Check for "no rows" error which means duplicate (ON CONFLICT DO NOTHING)
		if err.Error() == "no rows in result set" {
			return c.String(http.StatusConflict, "A tag with this name already exists")
		}
		slog.Error("failed to create tag", "error", err)
		return c.String(http.StatusInternalServerError, "Failed to create tag")
	}

	// Build row with count for template
	row := sqlc.ListTagsWithCountsRow{
		ID:            tag.ID,
		Name:          tag.Name,
		Color:         tag.Color,
		CreatedAt:     tag.CreatedAt,
		DocumentCount: 0, // New tag has no documents
	}

	// Set HX-Trigger to close modal
	c.Response().Header().Set("HX-Trigger", "closeModal")

	// Return the new tag card for HTMX
	return admin.TagCard(row).Render(ctx, c.Response().Writer)
}

// UpdateTag updates an existing tag
func (h *Handler) UpdateTag(c echo.Context) error {
	ctx := c.Request().Context()

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid tag ID")
	}

	name := c.FormValue("name")
	color := validateColor(c.FormValue("color"))

	// Validate required fields
	if name == "" {
		return c.String(http.StatusBadRequest, "Name is required")
	}

	// Update tag in database
	colorPtr := &color
	tag, err := h.db.Queries.UpdateTag(ctx, sqlc.UpdateTagParams{
		ID:    id,
		Name:  name,
		Color: colorPtr,
	})
	if err != nil {
		slog.Error("failed to update tag", "error", err)
		return c.String(http.StatusInternalServerError, "Failed to update tag")
	}

	// Get document count for this tag
	tags, err := h.db.Queries.ListTagsWithCounts(ctx)
	if err != nil {
		slog.Error("failed to get tag counts", "error", err)
		// Fall back to showing 0
		row := sqlc.ListTagsWithCountsRow{
			ID:            tag.ID,
			Name:          tag.Name,
			Color:         tag.Color,
			CreatedAt:     tag.CreatedAt,
			DocumentCount: 0,
		}
		c.Response().Header().Set("HX-Trigger", "closeModal")
		return admin.TagCard(row).Render(ctx, c.Response().Writer)
	}

	// Find the updated tag in the list to get accurate count
	var row sqlc.ListTagsWithCountsRow
	for _, t := range tags {
		if t.ID == id {
			row = t
			break
		}
	}

	// Set HX-Trigger to close modal
	c.Response().Header().Set("HX-Trigger", "closeModal")

	// Return updated tag card
	return admin.TagCard(row).Render(ctx, c.Response().Writer)
}

// DeleteTag removes a tag
func (h *Handler) DeleteTag(c echo.Context) error {
	ctx := c.Request().Context()

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid tag ID")
	}

	// Delete from database (cascade handles document_tags)
	if err := h.db.Queries.DeleteTag(ctx, id); err != nil {
		slog.Error("failed to delete tag", "error", err)
		return c.String(http.StatusInternalServerError, "Failed to delete tag")
	}

	// Return empty response for HTMX to remove the element
	return c.String(http.StatusOK, "")
}
