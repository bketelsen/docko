package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"os"
	"strings"

	"github.com/bketelsen/docko/templates/pages/admin"

	"github.com/google/uuid"
	"github.com/h2non/filetype"
	"github.com/labstack/echo/v4"
)

// UploadResult represents the response for a file upload
type UploadResult struct {
	Success     bool      `json:"success"`
	DocumentID  uuid.UUID `json:"document_id,omitempty"`
	Filename    string    `json:"filename"`
	IsDuplicate bool      `json:"is_duplicate"`
	Error       string    `json:"error,omitempty"`
}

// UploadPage renders the upload page
func (h *Handler) UploadPage(c echo.Context) error {
	return admin.Upload().Render(c.Request().Context(), c.Response().Writer)
}

// UploadSingle handles a single file upload via API
func (h *Handler) UploadSingle(c echo.Context) error {
	file, err := c.FormFile("file")
	if err != nil {
		return h.respondUpload(c, UploadResult{
			Success: false,
			Error:   "No file provided",
		}, http.StatusBadRequest)
	}

	result := h.processUpload(c, file)

	status := http.StatusCreated
	if !result.Success {
		status = http.StatusBadRequest
	} else if result.IsDuplicate {
		status = http.StatusOK
	}

	return h.respondUpload(c, result, status)
}

// UploadMultiple handles multiple file uploads via multipart form
func (h *Handler) UploadMultiple(c echo.Context) error {
	form, err := c.MultipartForm()
	if err != nil {
		return h.respondUploads(c, []UploadResult{{
			Success: false,
			Error:   "Invalid form data",
		}}, http.StatusBadRequest)
	}

	files := form.File["files"]
	if len(files) == 0 {
		// Try single file field
		files = form.File["file"]
	}

	if len(files) == 0 {
		return h.respondUploads(c, []UploadResult{{
			Success: false,
			Error:   "No files provided",
		}}, http.StatusBadRequest)
	}

	results := make([]UploadResult, 0, len(files))
	for _, file := range files {
		result := h.processUpload(c, file)
		results = append(results, result)
	}

	// Determine overall status
	status := http.StatusCreated
	hasErrors := false
	allDuplicates := true
	for _, r := range results {
		if !r.Success {
			hasErrors = true
		}
		if !r.IsDuplicate {
			allDuplicates = false
		}
	}
	if hasErrors {
		status = http.StatusMultiStatus
	} else if allDuplicates {
		status = http.StatusOK
	}

	return h.respondUploads(c, results, status)
}

// processUpload processes a single uploaded file
func (h *Handler) processUpload(c echo.Context, file *multipart.FileHeader) UploadResult {
	ctx := c.Request().Context()

	// Open the uploaded file
	src, err := file.Open()
	if err != nil {
		slog.Error("failed to open uploaded file", "error", err, "filename", file.Filename)
		return UploadResult{
			Success:  false,
			Filename: file.Filename,
			Error:    "Failed to read uploaded file",
		}
	}
	defer func() { _ = src.Close() }()

	// Create temp file
	tmpFile, err := os.CreateTemp("", "upload-*.pdf")
	if err != nil {
		slog.Error("failed to create temp file", "error", err)
		return UploadResult{
			Success:  false,
			Filename: file.Filename,
			Error:    "Failed to process upload",
		}
	}
	tmpPath := tmpFile.Name()
	defer func() { _ = os.Remove(tmpPath) }()

	// Copy to temp file
	if _, err := io.Copy(tmpFile, src); err != nil {
		_ = tmpFile.Close()
		slog.Error("failed to write temp file", "error", err)
		return UploadResult{
			Success:  false,
			Filename: file.Filename,
			Error:    "Failed to process upload",
		}
	}
	_ = tmpFile.Close()

	// Validate PDF using magic bytes
	if !isPDF(tmpPath) {
		return UploadResult{
			Success:  false,
			Filename: file.Filename,
			Error:    "Only PDF files are allowed",
		}
	}

	// Ingest the document
	doc, isDuplicate, err := h.docSvc.Ingest(ctx, tmpPath, file.Filename)
	if err != nil {
		slog.Error("failed to ingest document", "error", err, "filename", file.Filename)
		return UploadResult{
			Success:  false,
			Filename: file.Filename,
			Error:    fmt.Sprintf("Failed to ingest document: %v", err),
		}
	}

	return UploadResult{
		Success:     true,
		DocumentID:  doc.ID,
		Filename:    file.Filename,
		IsDuplicate: isDuplicate,
	}
}

// isPDF checks if the file at the given path is a valid PDF using magic bytes
func isPDF(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer func() { _ = f.Close() }()

	// Read first 262 bytes for magic number detection
	head := make([]byte, 262)
	n, err := f.Read(head)
	if err != nil && err != io.EOF {
		return false
	}

	kind, err := filetype.Match(head[:n])
	if err != nil {
		return false
	}

	return kind.MIME.Value == "application/pdf"
}

// respondUpload sends a single upload result
func (h *Handler) respondUpload(c echo.Context, result UploadResult, status int) error {
	if wantsJSON(c) {
		return c.JSON(status, result)
	}
	return h.renderUploadResult(c, []UploadResult{result}, status)
}

// respondUploads sends multiple upload results
func (h *Handler) respondUploads(c echo.Context, results []UploadResult, status int) error {
	if wantsJSON(c) {
		return c.JSON(status, results)
	}
	return h.renderUploadResult(c, results, status)
}

// wantsJSON checks if the client wants JSON response
func wantsJSON(c echo.Context) bool {
	accept := c.Request().Header.Get("Accept")
	return strings.Contains(accept, "application/json")
}

// renderUploadResult renders HTML partial for HTMX response
func (h *Handler) renderUploadResult(c echo.Context, results []UploadResult, status int) error {
	c.Response().WriteHeader(status)

	// Convert to template-friendly format
	type ResultItem struct {
		Success     bool
		DocumentID  string
		Filename    string
		IsDuplicate bool
		Error       string
	}

	items := make([]ResultItem, 0, len(results))
	for _, r := range results {
		item := ResultItem{
			Success:     r.Success,
			Filename:    r.Filename,
			IsDuplicate: r.IsDuplicate,
			Error:       r.Error,
		}
		if r.DocumentID != uuid.Nil {
			item.DocumentID = r.DocumentID.String()
		}
		items = append(items, item)
	}

	// Build simple HTML response for HTMX
	var html strings.Builder
	for _, item := range items {
		if item.Success {
			if item.IsDuplicate {
				html.WriteString(fmt.Sprintf(`<div class="p-4 bg-yellow-100 dark:bg-yellow-900 border border-yellow-400 dark:border-yellow-600 rounded-lg mb-2">
					<span class="font-medium">%s</span> - Duplicate detected (existing document)
					<a href="/documents/%s" class="text-blue-600 dark:text-blue-400 hover:underline ml-2">View</a>
				</div>`, escapeHTML(item.Filename), item.DocumentID))
			} else {
				html.WriteString(fmt.Sprintf(`<div class="p-4 bg-green-100 dark:bg-green-900 border border-green-400 dark:border-green-600 rounded-lg mb-2">
					<span class="font-medium">%s</span> - Uploaded successfully
					<a href="/documents/%s" class="text-blue-600 dark:text-blue-400 hover:underline ml-2">View</a>
				</div>`, escapeHTML(item.Filename), item.DocumentID))
			}
		} else {
			html.WriteString(fmt.Sprintf(`<div class="p-4 bg-red-100 dark:bg-red-900 border border-red-400 dark:border-red-600 rounded-lg mb-2">
				<span class="font-medium">%s</span> - %s
			</div>`, escapeHTML(item.Filename), escapeHTML(item.Error)))
		}
	}

	c.Response().Header().Set("Content-Type", "text/html; charset=utf-8")
	_, err := c.Response().Write([]byte(html.String()))
	return err
}

// escapeHTML escapes HTML special characters
func escapeHTML(s string) string {
	data, _ := json.Marshal(s)
	// Remove surrounding quotes and use raw string
	escaped := string(data[1 : len(data)-1])
	// JSON encoding escapes quotes but not < > &, so we need to handle those
	escaped = strings.ReplaceAll(escaped, "&", "&amp;")
	escaped = strings.ReplaceAll(escaped, "<", "&lt;")
	escaped = strings.ReplaceAll(escaped, ">", "&gt;")
	return escaped
}
