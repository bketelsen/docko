package handler

import (
	"docko/internal/ai"
	"docko/internal/auth"
	"docko/internal/config"
	"docko/internal/database"
	"docko/internal/document"
	"docko/internal/inbox"
	"docko/internal/middleware"
	"docko/internal/network"
	"docko/internal/processing"
	"docko/internal/queue"

	"github.com/labstack/echo/v4"
)

type Handler struct {
	cfg         *config.Config
	db          *database.DB
	auth        *auth.Service
	docSvc      *document.Service
	inboxSvc    *inbox.Service
	networkSvc  *network.Service
	aiSvc       *ai.Service
	queue       *queue.Queue
	broadcaster *processing.StatusBroadcaster
}

func New(cfg *config.Config, db *database.DB, authService *auth.Service, docSvc *document.Service, inboxSvc *inbox.Service, networkSvc *network.Service, aiSvc *ai.Service, q *queue.Queue, broadcaster *processing.StatusBroadcaster) *Handler {
	return &Handler{
		cfg:         cfg,
		db:          db,
		auth:        authService,
		docSvc:      docSvc,
		inboxSvc:    inboxSvc,
		networkSvc:  networkSvc,
		aiSvc:       aiSvc,
		queue:       q,
		broadcaster: broadcaster,
	}
}

func (h *Handler) RegisterRoutes(e *echo.Echo) {
	// Static files
	e.Static("/static", "static")
	e.Static("/assets", "assets")

	// Health check
	e.GET("/health", h.Health)

	// Auth routes (no middleware)
	e.GET("/login", h.LoginPage)
	e.POST("/login", h.Login)
	e.POST("/logout", h.Logout)

	// Protected routes (dashboard at root)
	e.GET("/", h.AdminDashboard, middleware.RequireAuth(h.auth))

	// Upload routes (protected)
	e.GET("/upload", h.UploadPage, middleware.RequireAuth(h.auth))
	e.POST("/upload", h.UploadMultiple, middleware.RequireAuth(h.auth))
	e.POST("/api/upload", h.UploadSingle, middleware.RequireAuth(h.auth))

	// Inbox management routes (protected)
	e.GET("/inboxes", h.InboxesPage, middleware.RequireAuth(h.auth))
	e.POST("/inboxes", h.CreateInbox, middleware.RequireAuth(h.auth))
	e.PUT("/inboxes/:id", h.UpdateInbox, middleware.RequireAuth(h.auth))
	e.DELETE("/inboxes/:id", h.DeleteInbox, middleware.RequireAuth(h.auth))
	e.POST("/inboxes/:id/toggle", h.ToggleInbox, middleware.RequireAuth(h.auth))
	e.GET("/inboxes/:id/events", h.InboxEvents, middleware.RequireAuth(h.auth))

	// Network sources management routes (protected)
	e.GET("/network-sources", h.NetworkSourcesPage, middleware.RequireAuth(h.auth))
	e.POST("/network-sources", h.CreateNetworkSource, middleware.RequireAuth(h.auth))
	e.DELETE("/network-sources/:id", h.DeleteNetworkSource, middleware.RequireAuth(h.auth))
	e.POST("/network-sources/:id/toggle", h.ToggleNetworkSource, middleware.RequireAuth(h.auth))
	e.POST("/network-sources/:id/test", h.TestNetworkSourceConnection, middleware.RequireAuth(h.auth))
	e.POST("/network-sources/:id/sync", h.SyncNetworkSource, middleware.RequireAuth(h.auth))
	e.POST("/network-sources/sync-all", h.SyncAllNetworkSources, middleware.RequireAuth(h.auth))
	e.GET("/network-sources/:id/events", h.NetworkSourceEvents, middleware.RequireAuth(h.auth))

	// Tag management routes (protected)
	e.GET("/tags", h.TagsPage, middleware.RequireAuth(h.auth))
	e.POST("/tags", h.CreateTag, middleware.RequireAuth(h.auth))
	e.POST("/tags/:id", h.UpdateTag, middleware.RequireAuth(h.auth))
	e.DELETE("/tags/:id", h.DeleteTag, middleware.RequireAuth(h.auth))

	// Correspondent management routes (protected)
	e.GET("/correspondents", h.CorrespondentsPage, middleware.RequireAuth(h.auth))
	e.GET("/correspondents/search", h.SearchCorrespondentsForDocument, middleware.RequireAuth(h.auth))
	e.POST("/correspondents", h.CreateCorrespondent, middleware.RequireAuth(h.auth))
	e.POST("/correspondents/merge", h.MergeCorrespondents, middleware.RequireAuth(h.auth))
	e.POST("/correspondents/:id", h.UpdateCorrespondent, middleware.RequireAuth(h.auth))
	e.DELETE("/correspondents/:id", h.DeleteCorrespondent, middleware.RequireAuth(h.auth))

	// Document routes (protected)
	e.GET("/documents", h.DocumentsPage, middleware.RequireAuth(h.auth))
	e.GET("/documents/:id", h.DocumentDetail, middleware.RequireAuth(h.auth))
	e.GET("/documents/:id/view", h.ViewPDF, middleware.RequireAuth(h.auth))
	e.GET("/documents/:id/download", h.DownloadPDF, middleware.RequireAuth(h.auth))
	e.GET("/documents/:id/thumbnail", h.ServeThumbnail, middleware.RequireAuth(h.auth))
	e.GET("/documents/:id/viewer", h.ViewerModal, middleware.RequireAuth(h.auth))
	e.POST("/api/documents/:id/retry", h.RetryDocument, middleware.RequireAuth(h.auth))

	// Document tag assignment routes (protected)
	e.GET("/documents/:id/tags/search", h.SearchTagsForDocument, middleware.RequireAuth(h.auth))
	e.GET("/documents/:id/tags/picker", h.GetDocumentTagsPicker, middleware.RequireAuth(h.auth))
	e.POST("/documents/:id/tags", h.AddDocumentTag, middleware.RequireAuth(h.auth))
	e.DELETE("/documents/:id/tags/:tag_id", h.RemoveDocumentTag, middleware.RequireAuth(h.auth))

	// Document correspondent assignment routes (protected)
	e.GET("/documents/:id/correspondent", h.GetDocumentCorrespondent, middleware.RequireAuth(h.auth))
	e.POST("/documents/:id/correspondent", h.SetDocumentCorrespondent, middleware.RequireAuth(h.auth))
	e.DELETE("/documents/:id/correspondent", h.RemoveDocumentCorrespondent, middleware.RequireAuth(h.auth))

	// SSE endpoint for processing status (protected)
	e.GET("/api/processing/status", h.ProcessingStatus, middleware.RequireAuth(h.auth))
}
