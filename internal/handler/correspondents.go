package handler

import (
	"context"
	"fmt"
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

// MergeCorrespondents merges multiple correspondents into a target correspondent
func (h *Handler) MergeCorrespondents(c echo.Context) error {
	ctx := c.Request().Context()

	// Parse target ID
	targetIDStr := c.FormValue("target_id")
	targetID, err := uuid.Parse(targetIDStr)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid target correspondent ID")
	}

	// Parse merge IDs (multiple values with same name)
	mergeIDStrs := c.Request().PostForm["merge_ids"]
	if len(mergeIDStrs) == 0 {
		return c.String(http.StatusBadRequest, "No correspondents selected for merge")
	}

	// Convert to UUIDs and validate
	var mergeIDs []uuid.UUID
	for _, idStr := range mergeIDStrs {
		id, err := uuid.Parse(idStr)
		if err != nil {
			return c.String(http.StatusBadRequest, "Invalid correspondent ID in merge list")
		}
		// Ensure target is not in merge list
		if id == targetID {
			return c.String(http.StatusBadRequest, "Cannot merge target into itself")
		}
		mergeIDs = append(mergeIDs, id)
	}

	if len(mergeIDs) == 0 {
		return c.String(http.StatusBadRequest, "No valid correspondents to merge")
	}

	// Execute merge in transaction
	if err := h.executeMerge(ctx, targetID, mergeIDs); err != nil {
		slog.Error("failed to merge correspondents", "error", err, "target", targetID, "merge_count", len(mergeIDs))
		return c.String(http.StatusInternalServerError, "Failed to merge correspondents")
	}

	slog.Info("merged correspondents", "target", targetID, "merged_count", len(mergeIDs))

	// Return updated correspondent list
	correspondents, err := h.db.Queries.ListCorrespondentsWithCounts(ctx)
	if err != nil {
		slog.Error("failed to list correspondents after merge", "error", err)
		return c.String(http.StatusInternalServerError, "Merge succeeded but failed to refresh list")
	}

	return admin.CorrespondentList(correspondents).Render(ctx, c.Response().Writer)
}

// executeMerge performs the merge operation in a transaction
func (h *Handler) executeMerge(ctx context.Context, targetID uuid.UUID, mergeIDs []uuid.UUID) error {
	tx, err := h.db.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	qtx := h.db.Queries.WithTx(tx)

	// Step 1: Update document references to point to target
	if err := qtx.MergeCorrespondentsUpdateDocs(ctx, sqlc.MergeCorrespondentsUpdateDocsParams{
		CorrespondentID: targetID,
		Column2:         mergeIDs,
	}); err != nil {
		return fmt.Errorf("failed to update document references: %w", err)
	}

	// Step 2: Get notes from merged correspondents
	notesToMerge, err := qtx.GetCorrespondentsNotes(ctx, mergeIDs)
	if err != nil {
		return fmt.Errorf("failed to get notes from merged correspondents: %w", err)
	}

	// Step 3: Append notes to target (if any)
	if len(notesToMerge) > 0 {
		var combinedNotes strings.Builder
		for _, n := range notesToMerge {
			if n.Notes != nil && *n.Notes != "" {
				combinedNotes.WriteString(fmt.Sprintf("--- Merged from %s ---\n%s\n", n.Name, *n.Notes))
			}
		}
		if combinedNotes.Len() > 0 {
			notesStr := combinedNotes.String()
			if _, err := qtx.AppendCorrespondentNotes(ctx, sqlc.AppendCorrespondentNotesParams{
				ID:    targetID,
				Notes: &notesStr,
			}); err != nil {
				return fmt.Errorf("failed to append notes to target: %w", err)
			}
		}
	}

	// Step 4: Delete merged correspondents
	if err := qtx.DeleteCorrespondentsByIds(ctx, mergeIDs); err != nil {
		return fmt.Errorf("failed to delete merged correspondents: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}
