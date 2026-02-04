package handler

import (
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/bketelsen/docko/internal/database/sqlc"
	"github.com/bketelsen/docko/templates/pages/admin"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// NetworkSourcesPage renders the network sources management page
func (h *Handler) NetworkSourcesPage(c echo.Context) error {
	ctx := c.Request().Context()

	sources, err := h.db.Queries.ListNetworkSources(ctx)
	if err != nil {
		slog.Error("failed to list network sources", "error", err)
		return c.String(http.StatusInternalServerError, "Failed to load network sources")
	}

	return admin.NetworkSources(sources).Render(ctx, c.Response().Writer)
}

// CreateNetworkSource creates a new network source
func (h *Handler) CreateNetworkSource(c echo.Context) error {
	ctx := c.Request().Context()

	// Parse form values
	name := c.FormValue("name")
	protocol := c.FormValue("protocol")
	host := c.FormValue("host")
	sharePath := c.FormValue("share_path")
	username := c.FormValue("username")
	password := c.FormValue("password")
	continuousSync := c.FormValue("continuous_sync") == "true"
	postImportAction := c.FormValue("post_import_action")
	moveSubfolder := c.FormValue("move_subfolder")
	duplicateAction := c.FormValue("duplicate_action")
	batchSizeStr := c.FormValue("batch_size")

	// Validate required fields
	if name == "" || host == "" || sharePath == "" {
		return c.String(http.StatusBadRequest, "Name, host, and share path are required")
	}

	// Parse protocol
	var proto sqlc.NetworkProtocol
	switch protocol {
	case "smb":
		proto = sqlc.NetworkProtocolSmb
	case "nfs":
		proto = sqlc.NetworkProtocolNfs
	default:
		return c.String(http.StatusBadRequest, "Invalid protocol")
	}

	// Parse post-import action
	var postAction sqlc.PostImportAction
	switch postImportAction {
	case "delete":
		postAction = sqlc.PostImportActionDelete
	case "move":
		postAction = sqlc.PostImportActionMove
	default:
		postAction = sqlc.PostImportActionLeave
	}

	// Parse duplicate action
	var dupAction sqlc.DuplicateAction
	switch duplicateAction {
	case "rename":
		dupAction = sqlc.DuplicateActionRename
	case "skip":
		dupAction = sqlc.DuplicateActionSkip
	default:
		dupAction = sqlc.DuplicateActionDelete
	}

	// Parse batch size
	batchSize := int32(100)
	if batchSizeStr != "" {
		if bs, err := strconv.Atoi(batchSizeStr); err == nil && bs > 0 {
			batchSize = int32(bs)
		}
	}

	// Encrypt password if provided
	var passwordEncrypted *string
	if password != "" {
		encrypted, err := h.networkSvc.GetCrypto().Encrypt(password)
		if err != nil {
			slog.Error("failed to encrypt password", "error", err)
			return c.String(http.StatusInternalServerError, "Failed to encrypt credentials")
		}
		passwordEncrypted = &encrypted
	}

	// Set optional fields
	var usernamePtr *string
	if username != "" {
		usernamePtr = &username
	}
	var moveSubfolderPtr *string
	if moveSubfolder != "" {
		moveSubfolderPtr = &moveSubfolder
	}

	// Create in database (disabled by default until tested)
	source, err := h.db.Queries.CreateNetworkSource(ctx, sqlc.CreateNetworkSourceParams{
		Name:              name,
		Protocol:          proto,
		Host:              host,
		SharePath:         sharePath,
		Username:          usernamePtr,
		PasswordEncrypted: passwordEncrypted,
		Enabled:           false, // Require test connection first
		ContinuousSync:    continuousSync,
		PostImportAction:  postAction,
		MoveSubfolder:     moveSubfolderPtr,
		DuplicateAction:   dupAction,
		BatchSize:         batchSize,
	})
	if err != nil {
		slog.Error("failed to create network source", "error", err)
		return c.String(http.StatusInternalServerError, "Failed to create network source")
	}

	// Return the new source card for HTMX
	return admin.NetworkSourceCard(source).Render(ctx, c.Response().Writer)
}

// TestNetworkSourceConnection tests connectivity to a network source
func (h *Handler) TestNetworkSourceConnection(c echo.Context) error {
	ctx := c.Request().Context()

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid source ID")
	}

	source, err := h.db.Queries.GetNetworkSource(ctx, id)
	if err != nil {
		return c.String(http.StatusNotFound, "Source not found")
	}

	if err := h.networkSvc.TestConnection(ctx, &source); err != nil {
		return c.String(http.StatusBadRequest, fmt.Sprintf("Connection failed: %v", err))
	}

	return c.String(http.StatusOK, "Connection successful")
}

// ToggleNetworkSource enables/disables a network source
func (h *Handler) ToggleNetworkSource(c echo.Context) error {
	ctx := c.Request().Context()

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid source ID")
	}

	source, err := h.db.Queries.GetNetworkSource(ctx, id)
	if err != nil {
		return c.String(http.StatusNotFound, "Source not found")
	}

	// Toggle enabled state
	newEnabled := !source.Enabled

	// Update in database
	updated, err := h.db.Queries.UpdateNetworkSource(ctx, sqlc.UpdateNetworkSourceParams{
		ID:                source.ID,
		Name:              source.Name,
		Protocol:          source.Protocol,
		Host:              source.Host,
		SharePath:         source.SharePath,
		Username:          source.Username,
		PasswordEncrypted: source.PasswordEncrypted,
		Enabled:           newEnabled,
		ContinuousSync:    source.ContinuousSync,
		PostImportAction:  source.PostImportAction,
		MoveSubfolder:     source.MoveSubfolder,
		DuplicateAction:   source.DuplicateAction,
		BatchSize:         source.BatchSize,
	})
	if err != nil {
		slog.Error("failed to toggle network source", "error", err)
		return c.String(http.StatusInternalServerError, "Failed to toggle source")
	}

	// Return updated card
	return admin.NetworkSourceCard(updated).Render(ctx, c.Response().Writer)
}

// SyncNetworkSource triggers a manual sync for a source
func (h *Handler) SyncNetworkSource(c echo.Context) error {
	ctx := c.Request().Context()

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid source ID")
	}

	imported, err := h.networkSvc.SyncSource(ctx, id)
	if err != nil {
		return c.String(http.StatusInternalServerError, fmt.Sprintf("Sync failed: %v", err))
	}

	return c.String(http.StatusOK, fmt.Sprintf("Imported %d files", imported))
}

// SyncAllNetworkSources triggers sync for all enabled sources
func (h *Handler) SyncAllNetworkSources(c echo.Context) error {
	ctx := c.Request().Context()

	if err := h.networkSvc.SyncAll(ctx); err != nil {
		return c.String(http.StatusInternalServerError, fmt.Sprintf("Sync failed: %v", err))
	}

	return c.String(http.StatusOK, "Sync complete")
}

// DeleteNetworkSource removes a network source
func (h *Handler) DeleteNetworkSource(c echo.Context) error {
	ctx := c.Request().Context()

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid source ID")
	}

	if err := h.db.Queries.DeleteNetworkSource(ctx, id); err != nil {
		slog.Error("failed to delete network source", "error", err)
		return c.String(http.StatusInternalServerError, "Failed to delete source")
	}

	// Return empty response for HTMX to remove the element
	return c.String(http.StatusOK, "")
}

// NetworkSourceEvents returns recent events for a source
func (h *Handler) NetworkSourceEvents(c echo.Context) error {
	ctx := c.Request().Context()

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid source ID")
	}

	events, err := h.db.Queries.ListNetworkSourceEvents(ctx, sqlc.ListNetworkSourceEventsParams{
		SourceID: id,
		Limit:    10,
	})
	if err != nil {
		slog.Error("failed to list events", "error", err)
		return c.String(http.StatusInternalServerError, "Failed to load events")
	}

	return admin.NetworkSourceEventsList(events).Render(ctx, c.Response().Writer)
}
