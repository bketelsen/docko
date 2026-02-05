package processing

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/google/uuid"

	"github.com/bketelsen/docko/internal/storage"
)

// ThumbnailGenerator creates thumbnails from PDF files
type ThumbnailGenerator struct {
	placeholderPath string           // Path to placeholder.webp for failures
	storage         *storage.Storage // Storage for path generation
}

// NewThumbnailGenerator creates a ThumbnailGenerator
// placeholderPath should point to a fallback WebP image for corrupt PDFs
func NewThumbnailGenerator(storage *storage.Storage, placeholderPath string) *ThumbnailGenerator {
	return &ThumbnailGenerator{
		storage:         storage,
		placeholderPath: placeholderPath,
	}
}

// Generate creates a 300px WebP thumbnail from the first page of a PDF
// Returns the path to the generated thumbnail
func (g *ThumbnailGenerator) Generate(ctx context.Context, pdfPath string, docID uuid.UUID) (string, error) {
	start := time.Now()

	// Compute thumbnail path using storage
	thumbPath := g.storage.PathForUUID(storage.CategoryThumbnails, docID, ".webp")

	// Ensure thumbnail directory exists
	if err := os.MkdirAll(filepath.Dir(thumbPath), 0755); err != nil {
		return "", fmt.Errorf("create thumbnail dir: %w", err)
	}

	// Create temp directory for intermediate PNG
	tmpDir, err := os.MkdirTemp("", "thumb-*")
	if err != nil {
		return "", fmt.Errorf("create temp dir: %w", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Output prefix for pdftoppm (will create thumb.png)
	pngPrefix := filepath.Join(tmpDir, "thumb")

	// Create context with 2-minute timeout for PDF rendering
	renderCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	// Step 1: PDF to PNG (first page only, 300px width)
	// -f 1: first page
	// -singlefile: output single file (no -000001 suffix)
	// -scale-to 300: scale to 300px width
	cmd := exec.CommandContext(renderCtx, "pdftoppm",
		"-png",
		"-f", "1",
		"-singlefile",
		"-scale-to", "300",
		pdfPath,
		pngPrefix,
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// PDF is corrupt or unrenderable - use placeholder
		slog.Warn("pdftoppm failed, using placeholder",
			"doc_id", docID,
			"error", err,
			"output", string(output))
		if placeholderErr := g.usePlaceholder(thumbPath); placeholderErr != nil {
			return "", fmt.Errorf("pdftoppm failed and placeholder copy failed: %w (original: %v)", placeholderErr, err)
		}
		slog.Info("thumbnail generated (placeholder)",
			"doc_id", docID,
			"path", thumbPath,
			"duration_ms", time.Since(start).Milliseconds())
		return thumbPath, nil
	}

	// pdftoppm creates thumb.png
	pngPath := pngPrefix + ".png"

	// Verify PNG was created
	if _, err := os.Stat(pngPath); os.IsNotExist(err) {
		slog.Warn("pdftoppm produced no output, using placeholder",
			"doc_id", docID,
			"expected_path", pngPath)
		if placeholderErr := g.usePlaceholder(thumbPath); placeholderErr != nil {
			return "", fmt.Errorf("pdftoppm produced no output and placeholder copy failed: %w", placeholderErr)
		}
		return thumbPath, nil
	}

	// Step 2: PNG to WebP
	// -q 80: quality 80%
	cmd = exec.CommandContext(renderCtx, "cwebp",
		"-q", "80",
		pngPath,
		"-o", thumbPath,
	)
	output, err = cmd.CombinedOutput()
	if err != nil {
		// cwebp failure is unexpected if PNG exists - return error
		return "", fmt.Errorf("cwebp failed: %w\noutput: %s", err, output)
	}

	slog.Info("thumbnail generated",
		"doc_id", docID,
		"path", thumbPath,
		"duration_ms", time.Since(start).Milliseconds())

	return thumbPath, nil
}

// usePlaceholder copies the placeholder image to the thumbnail location
func (g *ThumbnailGenerator) usePlaceholder(thumbPath string) error {
	if g.placeholderPath == "" {
		return fmt.Errorf("no placeholder path configured")
	}

	src, err := os.Open(g.placeholderPath)
	if err != nil {
		return fmt.Errorf("open placeholder: %w", err)
	}
	defer func() { _ = src.Close() }()

	dst, err := os.Create(thumbPath)
	if err != nil {
		return fmt.Errorf("create thumbnail: %w", err)
	}
	defer func() { _ = dst.Close() }()

	if _, err := io.Copy(dst, src); err != nil {
		_ = os.Remove(thumbPath)
		return fmt.Errorf("copy placeholder: %w", err)
	}

	return nil
}

// CheckDependencies verifies pdftoppm and cwebp are available
func CheckDependencies() error {
	if _, err := exec.LookPath("pdftoppm"); err != nil {
		return fmt.Errorf("pdftoppm not found: %w", err)
	}
	if _, err := exec.LookPath("cwebp"); err != nil {
		return fmt.Errorf("cwebp not found: %w", err)
	}
	return nil
}
