package handler

import (
	"errors"
	"net/http"

	"docko/internal/database/sqlc"
	"docko/internal/document"
	"docko/internal/processing"
	"docko/templates/pages/admin"
	"docko/templates/partials"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/labstack/echo/v4"
)

// DocumentsPage renders the document list page with processing status
func (h *Handler) DocumentsPage(c echo.Context) error {
	ctx := c.Request().Context()

	// Get documents with correspondent info (default limit 50)
	rows, err := h.db.Queries.ListDocumentsWithCorrespondent(ctx, sqlc.ListDocumentsWithCorrespondentParams{
		Limit:  50,
		Offset: 0,
	})
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to list documents")
	}

	// Extract documents and build correspondent map
	docs := make([]sqlc.Document, len(rows))
	docCorrespondents := make(admin.DocumentCorrespondentMap)
	docIDs := make([]uuid.UUID, len(rows))

	for i, row := range rows {
		docs[i] = sqlc.Document{
			ID:                 row.ID,
			OriginalFilename:   row.OriginalFilename,
			ContentHash:        row.ContentHash,
			FileSize:           row.FileSize,
			PageCount:          row.PageCount,
			PdfTitle:           row.PdfTitle,
			PdfAuthor:          row.PdfAuthor,
			PdfCreatedAt:       row.PdfCreatedAt,
			DocumentDate:       row.DocumentDate,
			CreatedAt:          row.CreatedAt,
			UpdatedAt:          row.UpdatedAt,
			ProcessingStatus:   row.ProcessingStatus,
			TextContent:        row.TextContent,
			ThumbnailGenerated: row.ThumbnailGenerated,
			ProcessingError:    row.ProcessingError,
			ProcessedAt:        row.ProcessedAt,
		}
		docIDs[i] = row.ID

		// Add correspondent to map if present
		if row.CorrespondentName != nil {
			docCorrespondents[row.ID] = *row.CorrespondentName
		}
	}

	// Build tags map for all documents
	docTags := make(admin.DocumentTagsMap)
	if len(docs) > 0 {
		// Fetch tags for all documents in one query
		tagRows, err := h.db.Queries.GetTagsForDocuments(ctx, docIDs)
		if err == nil {
			// Build map from rows
			for _, row := range tagRows {
				tag := sqlc.Tag{
					ID:        row.ID,
					Name:      row.Name,
					Color:     row.Color,
					CreatedAt: row.CreatedAt,
				}
				docTags[row.DocumentID] = append(docTags[row.DocumentID], tag)
			}
		}
		// If error, just continue with empty tags - non-fatal
	}

	return admin.Documents(docs, docTags, docCorrespondents).Render(ctx, c.Response().Writer)
}

// DocumentDetail renders the document detail page
// GET /documents/:id
func (h *Handler) DocumentDetail(c echo.Context) error {
	ctx := c.Request().Context()

	docID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid document ID")
	}

	doc, err := h.db.Queries.GetDocument(ctx, docID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return echo.NewHTTPError(http.StatusNotFound, "document not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to fetch document")
	}

	// Fetch tags for this document
	tags, err := h.db.Queries.GetDocumentTags(ctx, docID)
	if err != nil {
		// Non-fatal: render without tags
		tags = []sqlc.Tag{}
	}

	// Fetch correspondent for this document (may not exist)
	var correspondent *sqlc.Correspondent
	c2, err := h.db.Queries.GetDocumentCorrespondent(ctx, docID)
	if err == nil {
		correspondent = &c2
	}
	// If error (including no rows), correspondent stays nil - that's fine

	return admin.DocumentDetail(doc, tags, correspondent).Render(ctx, c.Response().Writer)
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

// ViewerModal returns the PDF viewer modal HTML for HTMX
// GET /documents/:id/viewer
func (h *Handler) ViewerModal(c echo.Context) error {
	ctx := c.Request().Context()

	docID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid document ID")
	}

	doc, err := h.db.Queries.GetDocument(ctx, docID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return echo.NewHTTPError(http.StatusNotFound, "document not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to fetch document")
	}

	return partials.PDFViewerModal(doc).Render(ctx, c.Response().Writer)
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
