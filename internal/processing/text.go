package processing

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/ledongthuc/pdf"
)

// TextExtractor extracts text from PDF files using embedded text extraction
// and OCR fallback via the OCRmyPDF Docker service
type TextExtractor struct {
	minTextLength int    // Minimum chars to consider "has text" (default: 100)
	ocrInputPath  string // Path to ocr-input directory (bind mount to OCRmyPDF container)
	ocrOutputPath string // Path to ocr-output directory (bind mount from OCRmyPDF container)
	ocrTimeout    time.Duration
}

// NewTextExtractor creates a TextExtractor
// ocrInputPath and ocrOutputPath are the mount points for the OCRmyPDF service volumes
func NewTextExtractor(ocrInputPath, ocrOutputPath string) *TextExtractor {
	return &TextExtractor{
		minTextLength: 100,
		ocrInputPath:  ocrInputPath,
		ocrOutputPath: ocrOutputPath,
		ocrTimeout:    5 * time.Minute,
	}
}

// Extract extracts text from a PDF file
// Returns: text content, method used ("embedded" or "ocr"), error
func (e *TextExtractor) Extract(ctx context.Context, pdfPath string) (string, string, error) {
	start := time.Now()

	// Try embedded text extraction first
	text, hasText, err := e.extractEmbedded(pdfPath)
	if err != nil {
		slog.Warn("embedded text extraction failed, falling back to OCR",
			"pdf_path", pdfPath,
			"error", err)
	}

	if hasText {
		slog.Info("extracted embedded text",
			"pdf_path", pdfPath,
			"text_length", len(text),
			"duration_ms", time.Since(start).Milliseconds())
		return text, "embedded", nil
	}

	// Fall back to OCR via OCRmyPDF service
	slog.Info("insufficient embedded text, falling back to OCR",
		"pdf_path", pdfPath,
		"embedded_length", len(text))

	ocrText, err := e.ocrViaService(ctx, pdfPath)
	if err != nil {
		return "", "", fmt.Errorf("ocr extraction: %w", err)
	}

	slog.Info("extracted text via OCR",
		"pdf_path", pdfPath,
		"text_length", len(ocrText),
		"duration_ms", time.Since(start).Milliseconds())

	return ocrText, "ocr", nil
}

// extractEmbedded attempts to extract embedded text using ledongthuc/pdf
// Returns: text, hasText (len > minTextLength), error
func (e *TextExtractor) extractEmbedded(pdfPath string) (string, bool, error) {
	f, r, err := pdf.Open(pdfPath)
	if err != nil {
		return "", false, fmt.Errorf("open pdf: %w", err)
	}
	defer f.Close()

	reader, err := r.GetPlainText()
	if err != nil {
		return "", false, fmt.Errorf("get plain text: %w", err)
	}

	var buf bytes.Buffer
	if _, err := buf.ReadFrom(reader); err != nil {
		return "", false, fmt.Errorf("read text: %w", err)
	}

	text := buf.String()
	trimmed := strings.TrimSpace(text)
	hasText := len(trimmed) >= e.minTextLength

	return text, hasText, nil
}

// ocrViaService sends PDF to OCRmyPDF service and retrieves extracted text
// Uses shared volumes configured in docker-compose.yml
func (e *TextExtractor) ocrViaService(ctx context.Context, pdfPath string) (string, error) {
	// Generate unique job ID
	jobID := uuid.New().String()

	// Copy PDF to ocr-input volume
	inputPath := filepath.Join(e.ocrInputPath, jobID+".pdf")
	if err := copyFile(pdfPath, inputPath); err != nil {
		return "", fmt.Errorf("copy to ocr input: %w", err)
	}

	// Clean up input file on return (output files are cleaned after reading)
	defer os.Remove(inputPath)

	// Wait for output in ocr-output volume
	// The OCRmyPDF service watches /input and writes to /output
	outputTextPath := filepath.Join(e.ocrOutputPath, jobID+".txt")
	outputPDFPath := filepath.Join(e.ocrOutputPath, jobID+".pdf")

	// Poll for output with timeout
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	timeout := time.After(e.ocrTimeout)

	for {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-timeout:
			return "", fmt.Errorf("OCR timeout waiting for output after %v", e.ocrTimeout)
		case <-ticker.C:
			// Check if output text file exists
			if _, err := os.Stat(outputTextPath); err == nil {
				// Output ready, read it
				text, err := os.ReadFile(outputTextPath)
				if err != nil {
					return "", fmt.Errorf("read OCR output: %w", err)
				}

				// Clean up output files
				os.Remove(outputTextPath)
				os.Remove(outputPDFPath)

				return string(text), nil
			}
		}
	}
}

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	// Ensure destination directory exists
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}

	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("open source: %w", err)
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("create destination: %w", err)
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return fmt.Errorf("copy: %w", err)
	}

	return dstFile.Sync()
}
