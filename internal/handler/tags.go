package handler

import (
	"log/slog"
	"net/http"
	"strings"

	"docko/internal/database/sqlc"
	"docko/templates/pages/admin"
	"docko/templates/partials"

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

// SearchTagsForDocument searches for tags not already assigned to a document
// GET /documents/:id/tags/search?q=query
func (h *Handler) SearchTagsForDocument(c echo.Context) error {
	ctx := c.Request().Context()

	docID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid document ID")
	}

	query := strings.TrimSpace(c.QueryParam("q"))

	// Search for tags not already assigned to this document
	searchPattern := "%" + query + "%"
	tags, err := h.db.Queries.SearchTagsExcludingDocument(ctx, sqlc.SearchTagsExcludingDocumentParams{
		Name:       searchPattern,
		DocumentID: docID,
	})
	if err != nil {
		slog.Error("failed to search tags", "error", err)
		return c.String(http.StatusInternalServerError, "Failed to search tags")
	}

	// Check if we should offer to create a new tag
	showCreate := query != "" && !hasExactMatch(tags, query)

	return partials.TagSearchResults(docID.String(), tags, query, showCreate).Render(ctx, c.Response().Writer)
}

// hasExactMatch checks if any tag has the exact name (case-insensitive)
func hasExactMatch(tags []sqlc.Tag, query string) bool {
	queryLower := strings.ToLower(query)
	for _, tag := range tags {
		if strings.ToLower(tag.Name) == queryLower {
			return true
		}
	}
	return false
}

// AddDocumentTag assigns a tag to a document
// POST /documents/:id/tags
func (h *Handler) AddDocumentTag(c echo.Context) error {
	ctx := c.Request().Context()

	docID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid document ID")
	}

	tagIDStr := c.FormValue("tag_id")
	tagName := strings.TrimSpace(c.FormValue("name"))

	var tagID uuid.UUID

	// If tag_id is "new", create the tag first
	if tagIDStr == "new" {
		if tagName == "" {
			return c.String(http.StatusBadRequest, "Tag name is required")
		}
		defaultColor := "blue"
		tag, err := h.db.Queries.CreateTag(ctx, sqlc.CreateTagParams{
			Name:  tagName,
			Color: &defaultColor,
		})
		if err != nil {
			// If duplicate, try to find existing tag
			if err.Error() == "no rows in result set" {
				existingTags, err := h.db.Queries.SearchTags(ctx, tagName)
				if err != nil || len(existingTags) == 0 {
					return c.String(http.StatusConflict, "Tag already exists but couldn't be found")
				}
				// Find exact match
				for _, t := range existingTags {
					if strings.EqualFold(t.Name, tagName) {
						tagID = t.ID
						break
					}
				}
				if tagID == uuid.Nil {
					tagID = existingTags[0].ID
				}
			} else {
				slog.Error("failed to create tag", "error", err)
				return c.String(http.StatusInternalServerError, "Failed to create tag")
			}
		} else {
			tagID = tag.ID
		}
	} else {
		tagID, err = uuid.Parse(tagIDStr)
		if err != nil {
			return c.String(http.StatusBadRequest, "Invalid tag ID")
		}
	}

	// Add tag to document
	err = h.db.Queries.AddDocumentTag(ctx, sqlc.AddDocumentTagParams{
		DocumentID: docID,
		TagID:      tagID,
	})
	if err != nil {
		slog.Error("failed to add document tag", "error", err)
		return c.String(http.StatusInternalServerError, "Failed to add tag")
	}

	// Return updated tags list
	tags, err := h.db.Queries.GetDocumentTags(ctx, docID)
	if err != nil {
		slog.Error("failed to get document tags", "error", err)
		return c.String(http.StatusInternalServerError, "Failed to get tags")
	}

	return partials.DocumentTagsList(docID.String(), tags).Render(ctx, c.Response().Writer)
}

// RemoveDocumentTag removes a tag from a document
// DELETE /documents/:id/tags/:tag_id
func (h *Handler) RemoveDocumentTag(c echo.Context) error {
	ctx := c.Request().Context()

	docID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid document ID")
	}

	tagID, err := uuid.Parse(c.Param("tag_id"))
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid tag ID")
	}

	// Remove tag from document
	err = h.db.Queries.RemoveDocumentTag(ctx, sqlc.RemoveDocumentTagParams{
		DocumentID: docID,
		TagID:      tagID,
	})
	if err != nil {
		slog.Error("failed to remove document tag", "error", err)
		return c.String(http.StatusInternalServerError, "Failed to remove tag")
	}

	// Return updated tags list
	tags, err := h.db.Queries.GetDocumentTags(ctx, docID)
	if err != nil {
		slog.Error("failed to get document tags", "error", err)
		return c.String(http.StatusInternalServerError, "Failed to get tags")
	}

	return partials.DocumentTagsList(docID.String(), tags).Render(ctx, c.Response().Writer)
}
