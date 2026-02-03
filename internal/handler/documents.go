package handler

import (
	"net/http"

	"docko/internal/database/sqlc"
	"docko/internal/document"
	"docko/internal/processing"
	"docko/templates/pages/admin"
	"docko/templates/partials"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// DocumentsPage renders the document list page with processing status
func (h *Handler) DocumentsPage(c echo.Context) error {
	ctx := c.Request().Context()

	// Get documents with pagination (default limit 50)
	docs, err := h.db.Queries.ListDocuments(ctx, sqlc.ListDocumentsParams{
		Limit:  50,
		Offset: 0,
	})
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to list documents")
	}

	return admin.Documents(docs).Render(ctx, c.Response().Writer)
}

// ViewPDF serves a PDF file inline for browser viewing
// GET /documents/:id/view
func (h *Handler) ViewPDF(c echo.Context) error {
	ctx := c.Request().Context()

	docID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid document ID")
	}

	doc, err := h.db.Queries.GetDocument(ctx, docID)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "document not found")
	}

	pdfPath := h.docSvc.OriginalPath(&doc)
	if !h.docSvc.FileExists(pdfPath) {
		return echo.NewHTTPError(http.StatusNotFound, "PDF file not found")
	}

	// Serve with Content-Disposition: inline for browser viewing
	return c.Inline(pdfPath, doc.OriginalFilename)
}

// DownloadPDF serves a PDF file as attachment for download
// GET /documents/:id/download
func (h *Handler) DownloadPDF(c echo.Context) error {
	ctx := c.Request().Context()

	docID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid document ID")
	}

	doc, err := h.db.Queries.GetDocument(ctx, docID)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "document not found")
	}

	pdfPath := h.docSvc.OriginalPath(&doc)
	if !h.docSvc.FileExists(pdfPath) {
		return echo.NewHTTPError(http.StatusNotFound, "PDF file not found")
	}

	// Serve with Content-Disposition: attachment for download
	return c.Attachment(pdfPath, doc.OriginalFilename)
}

// ServeThumbnail serves a document's thumbnail image
// GET /documents/:id/thumbnail
func (h *Handler) ServeThumbnail(c echo.Context) error {
	ctx := c.Request().Context()

	docID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid document ID")
	}

	doc, err := h.db.Queries.GetDocument(ctx, docID)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "document not found")
	}

	// Check if thumbnail was generated
	if !doc.ThumbnailGenerated {
		return echo.NewHTTPError(http.StatusNotFound, "thumbnail not generated")
	}

	thumbnailPath := h.docSvc.ThumbnailPath(&doc)
	if !h.docSvc.FileExists(thumbnailPath) {
		return echo.NewHTTPError(http.StatusNotFound, "thumbnail file not found")
	}

	// Serve thumbnail image
	return c.File(thumbnailPath)
}

// RetryDocument re-queues a failed document for processing
// POST /api/documents/:id/retry
func (h *Handler) RetryDocument(c echo.Context) error {
	ctx := c.Request().Context()

	docID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid document ID")
	}

	// Verify document exists and is in failed state
	doc, err := h.db.Queries.GetDocument(ctx, docID)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "document not found")
	}

	if doc.ProcessingStatus != sqlc.ProcessingStatusFailed {
		return echo.NewHTTPError(http.StatusBadRequest, "document is not in failed state")
	}

	// Reset status to pending
	_, err = h.db.Queries.SetDocumentProcessingStatus(ctx, sqlc.SetDocumentProcessingStatusParams{
		ID:               docID,
		ProcessingStatus: sqlc.ProcessingStatusPending,
	})
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to reset status")
	}

	// Re-enqueue processing job
	payload := document.IngestPayload{DocumentID: docID}
	_, err = h.queue.Enqueue(ctx, document.QueueDefault, document.JobTypeProcess, payload)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to enqueue processing job")
	}

	// Broadcast status update
	if h.broadcaster != nil {
		h.broadcaster.Broadcast(processing.StatusUpdate{
			DocumentID: docID,
			Status:     "pending",
		})
	}

	// Return updated status partial
	return partials.DocumentStatus(docID.String(), "pending", "").
		Render(ctx, c.Response().Writer)
}
