package handler

import (
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"docko/internal/database/sqlc"
	"docko/internal/document"
	"docko/internal/processing"
	"docko/templates/pages/admin"
	"docko/templates/partials"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/labstack/echo/v4"
)

// searchParams holds parsed search parameters
type searchParams struct {
	Query           string
	CorrespondentID *uuid.UUID
	TagIDs          []uuid.UUID
	DateFrom        *time.Time
	DateTo          *time.Time
	DateRange       string // Original value: "today", "7d", "30d", "1y"
	Page            int
	PerPage         int
}

// parseSearchParams extracts search parameters from request
func parseSearchParams(c echo.Context) searchParams {
	params := searchParams{
		Query:     strings.TrimSpace(c.QueryParam("q")),
		DateRange: c.QueryParam("date"),
		Page:      1,
		PerPage:   20,
	}

	// Parse page number
	if p := c.QueryParam("page"); p != "" {
		if page, err := strconv.Atoi(p); err == nil && page > 0 {
			params.Page = page
		}
	}

	// Parse correspondent filter
	if corrID := c.QueryParam("correspondent"); corrID != "" {
		if id, err := uuid.Parse(corrID); err == nil {
			params.CorrespondentID = &id
		}
	}

	// Parse tag filters (multiple allowed)
	for _, tagID := range c.QueryParams()["tag"] {
		if id, err := uuid.Parse(tagID); err == nil {
			params.TagIDs = append(params.TagIDs, id)
		}
	}

	// Parse date range
	now := time.Now()
	switch params.DateRange {
	case "today":
		start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		params.DateFrom = &start
	case "7d":
		start := now.AddDate(0, 0, -7)
		params.DateFrom = &start
	case "30d":
		start := now.AddDate(0, 0, -30)
		params.DateFrom = &start
	case "1y":
		start := now.AddDate(-1, 0, 0)
		params.DateFrom = &start
	}

	return params
}

// buildActiveFilters creates filter chip data from params
func buildActiveFilters(params searchParams, correspondentName string, tagNames map[uuid.UUID]string) []partials.ActiveFilter {
	var filters []partials.ActiveFilter
	baseURL := "/documents?"

	// Build base params (for removal URLs)
	buildURL := func(exclude string) string {
		var parts []string
		if params.Query != "" && exclude != "query" {
			parts = append(parts, "q="+url.QueryEscape(params.Query))
		}
		if params.CorrespondentID != nil && exclude != "correspondent" {
			parts = append(parts, "correspondent="+params.CorrespondentID.String())
		}
		if params.DateRange != "" && exclude != "date" {
			parts = append(parts, "date="+params.DateRange)
		}
		for _, tagID := range params.TagIDs {
			if exclude != "tag-"+tagID.String() {
				parts = append(parts, "tag="+tagID.String())
			}
		}
		if len(parts) == 0 {
			return "/documents"
		}
		return baseURL + strings.Join(parts, "&")
	}

	if params.Query != "" {
		filters = append(filters, partials.ActiveFilter{
			Type:      "Search",
			Label:     params.Query,
			Value:     params.Query,
			RemoveURL: buildURL("query"),
		})
	}

	if params.CorrespondentID != nil && correspondentName != "" {
		filters = append(filters, partials.ActiveFilter{
			Type:      "Correspondent",
			Label:     correspondentName,
			Value:     params.CorrespondentID.String(),
			RemoveURL: buildURL("correspondent"),
		})
	}

	if params.DateRange != "" {
		labels := map[string]string{
			"today": "Today",
			"7d":    "Last 7 days",
			"30d":   "Last 30 days",
			"1y":    "Last year",
		}
		filters = append(filters, partials.ActiveFilter{
			Type:      "Date",
			Label:     labels[params.DateRange],
			Value:     params.DateRange,
			RemoveURL: buildURL("date"),
		})
	}

	for _, tagID := range params.TagIDs {
		if name, ok := tagNames[tagID]; ok {
			filters = append(filters, partials.ActiveFilter{
				Type:      "Tag",
				Label:     name,
				Value:     tagID.String(),
				RemoveURL: buildURL("tag-" + tagID.String()),
			})
		}
	}

	return filters
}

// DocumentsPage renders the document list page with search support
// GET /documents
func (h *Handler) DocumentsPage(c echo.Context) error {
	ctx := c.Request().Context()
	params := parseSearchParams(c)

	// Build query parameters for SearchDocuments
	var query *string
	if params.Query != "" {
		query = &params.Query
	}

	var correspondentID uuid.UUID
	hasCorrespondent := params.CorrespondentID != nil
	if hasCorrespondent {
		correspondentID = *params.CorrespondentID
	}

	var dateFrom, dateTo time.Time
	hasDateFrom := params.DateFrom != nil
	hasDateTo := params.DateTo != nil
	if hasDateFrom {
		dateFrom = *params.DateFrom
	}
	if hasDateTo {
		dateTo = *params.DateTo
	}

	hasTags := len(params.TagIDs) > 0
	tagCount := int32(len(params.TagIDs))

	// Execute search
	rows, err := h.db.Queries.SearchDocuments(ctx, sqlc.SearchDocumentsParams{
		Query:            query,
		HasCorrespondent: hasCorrespondent,
		CorrespondentID:  correspondentID,
		HasDateFrom:      hasDateFrom,
		DateFrom:         dateFrom,
		HasDateTo:        hasDateTo,
		DateTo:           dateTo,
		HasTags:          hasTags,
		TagIds:           params.TagIDs,
		TagCount:         tagCount,
		LimitCount:       int64(params.PerPage),
		OffsetCount:      int64((params.Page - 1) * params.PerPage),
	})
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "search failed")
	}

	// Get total count for pagination
	total, err := h.db.Queries.CountSearchDocuments(ctx, sqlc.CountSearchDocumentsParams{
		Query:            query,
		HasCorrespondent: hasCorrespondent,
		CorrespondentID:  correspondentID,
		HasDateFrom:      hasDateFrom,
		DateFrom:         dateFrom,
		HasDateTo:        hasDateTo,
		DateTo:           dateTo,
		HasTags:          hasTags,
		TagIds:           params.TagIDs,
		TagCount:         tagCount,
	})
	if err != nil {
		total = 0 // Non-fatal, just show results
	}

	// Convert rows to SearchResult
	results := make([]partials.SearchResult, len(rows))
	docIDs := make([]uuid.UUID, len(rows))
	for i, row := range rows {
		results[i] = partials.SearchResult{SearchDocumentsRow: row}
		docIDs[i] = row.ID
	}

	// Fetch tags for results
	docTags := make(partials.DocumentTagsMap)
	if len(docIDs) > 0 {
		tagRows, err := h.db.Queries.GetTagsForDocuments(ctx, docIDs)
		if err == nil {
			for _, row := range tagRows {
				tag := sqlc.Tag{ID: row.ID, Name: row.Name, Color: row.Color, CreatedAt: row.CreatedAt}
				docTags[row.DocumentID] = append(docTags[row.DocumentID], tag)
			}
		}
	}

	// Build correspondents map
	docCorrespondents := make(partials.DocumentCorrespondentMap)
	for _, result := range results {
		if result.CorrespondentName != nil {
			docCorrespondents[result.ID] = *result.CorrespondentName
		}
	}

	// Get correspondent name for filter chip
	var correspondentName string
	if params.CorrespondentID != nil {
		if corr, err := h.db.Queries.GetCorrespondent(ctx, *params.CorrespondentID); err == nil {
			correspondentName = corr.Name
		}
	}

	// Get tag names for filter chips
	tagNames := make(map[uuid.UUID]string)
	for _, tagID := range params.TagIDs {
		if tag, err := h.db.Queries.GetTag(ctx, tagID); err == nil {
			tagNames[tagID] = tag.Name
		}
	}

	activeFilters := buildActiveFilters(params, correspondentName, tagNames)

	// Convert params for template
	templateParams := partials.SearchParams{
		Query:     params.Query,
		DateRange: params.DateRange,
		Page:      params.Page,
		PerPage:   params.PerPage,
	}
	if params.CorrespondentID != nil {
		templateParams.CorrespondentID = params.CorrespondentID.String()
	}
	for _, tagID := range params.TagIDs {
		templateParams.TagIDs = append(templateParams.TagIDs, tagID.String())
	}

	// Check if HTMX request (return partial) or full page
	if c.Request().Header.Get("HX-Request") == "true" {
		return partials.SearchResults(results, docTags, docCorrespondents, templateParams, int(total), activeFilters).
			Render(ctx, c.Response().Writer)
	}

	// Fetch all tags for filter dropdown (full page only)
	allTagRows, err := h.db.Queries.ListTagsWithCounts(ctx)
	if err != nil {
		allTagRows = []sqlc.ListTagsWithCountsRow{} // Non-fatal
	}

	// Convert to Tag slice (ListTagsWithCounts returns rows with extra count field)
	allTags := make([]sqlc.Tag, len(allTagRows))
	for i, t := range allTagRows {
		allTags[i] = sqlc.Tag{
			ID:        t.ID,
			Name:      t.Name,
			Color:     t.Color,
			CreatedAt: t.CreatedAt,
		}
	}

	// Fetch all correspondents for filter dropdown (full page only)
	allCorrRows, err := h.db.Queries.ListCorrespondentsWithCounts(ctx)
	if err != nil {
		allCorrRows = []sqlc.ListCorrespondentsWithCountsRow{} // Non-fatal
	}

	// Convert to Correspondent slice
	allCorrespondents := make([]sqlc.Correspondent, len(allCorrRows))
	for i, c := range allCorrRows {
		allCorrespondents[i] = sqlc.Correspondent{
			ID:        c.ID,
			Name:      c.Name,
			Notes:     c.Notes,
			CreatedAt: c.CreatedAt,
		}
	}

	// Full page - render Documents template with search results
	return admin.DocumentsWithSearch(results, docTags, docCorrespondents, templateParams, int(total), activeFilters, allTags, allCorrespondents).
		Render(ctx, c.Response().Writer)
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

	// Get AI suggestions for this document
	aiSuggestions, err := h.db.Queries.ListPendingSuggestionsForDocument(ctx, docID)
	if err != nil {
		aiSuggestions = []sqlc.AiSuggestion{}
	}

	// Check if AI is enabled (has available providers)
	aiEnabled := len(h.aiSvc.AvailableProviders()) > 0

	return admin.DocumentDetail(doc, tags, correspondent, aiSuggestions, aiEnabled).Render(ctx, c.Response().Writer)
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
