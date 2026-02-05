package processing

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewTextExtractor(t *testing.T) {
	e := NewTextExtractor("/input", "/output")

	if e.minTextLength != 100 {
		t.Errorf("minTextLength = %d, want 100", e.minTextLength)
	}
	if e.ocrInputPath != "/input" {
		t.Errorf("ocrInputPath = %s, want /input", e.ocrInputPath)
	}
	if e.ocrOutputPath != "/output" {
		t.Errorf("ocrOutputPath = %s, want /output", e.ocrOutputPath)
	}
	if e.ocrTimeout != 5*time.Minute {
		t.Errorf("ocrTimeout = %v, want 5m", e.ocrTimeout)
	}
}

func TestExtractEmbedded_InvalidPath(t *testing.T) {
	e := NewTextExtractor("/input", "/output")

	_, _, err := e.extractEmbedded("/nonexistent/file.pdf")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestCopyFile(t *testing.T) {
	// Create temp directories
	srcDir := t.TempDir()
	dstDir := t.TempDir()

	// Create source file
	srcPath := filepath.Join(srcDir, "test.txt")
	content := []byte("test content")
	if err := os.WriteFile(srcPath, content, 0644); err != nil {
		t.Fatalf("failed to create source file: %v", err)
	}

	// Copy to nested destination
	dstPath := filepath.Join(dstDir, "nested", "dir", "test.txt")
	if err := copyFile(srcPath, dstPath); err != nil {
		t.Fatalf("copyFile failed: %v", err)
	}

	// Verify content
	result, err := os.ReadFile(dstPath)
	if err != nil {
		t.Fatalf("failed to read destination: %v", err)
	}
	if string(result) != string(content) {
		t.Errorf("content = %q, want %q", result, content)
	}
}

func TestCopyFile_SourceNotExists(t *testing.T) {
	dstDir := t.TempDir()
	dstPath := filepath.Join(dstDir, "test.txt")

	err := copyFile("/nonexistent/source.txt", dstPath)
	if err == nil {
		t.Error("expected error for nonexistent source")
	}
}

func TestOcrViaService_Timeout(t *testing.T) {
	// Create temp directories simulating OCR service
	inputDir := t.TempDir()
	outputDir := t.TempDir()

	// Create a test PDF (minimal valid PDF)
	pdfDir := t.TempDir()
	pdfPath := filepath.Join(pdfDir, "test.pdf")
	// Minimal PDF content
	minimalPDF := []byte("%PDF-1.0\n1 0 obj<</Type/Catalog/Pages 2 0 R>>endobj\n2 0 obj<</Type/Pages/Count 0/Kids[]>>endobj\nxref\n0 3\n0000000000 65535 f \n0000000009 00000 n \n0000000052 00000 n \ntrailer<</Size 3/Root 1 0 R>>\nstartxref\n101\n%%EOF")
	if err := os.WriteFile(pdfPath, minimalPDF, 0644); err != nil {
		t.Fatalf("failed to create test PDF: %v", err)
	}

	e := NewTextExtractor(inputDir, outputDir)
	// Set very short timeout for testing
	e.ocrTimeout = 100 * time.Millisecond

	ctx := context.Background()
	_, err := e.ocrViaService(ctx, pdfPath)

	// Should timeout since no OCR service is writing output
	if err == nil {
		t.Error("expected timeout error")
	}
}

func TestOcrViaService_ContextCancellation(t *testing.T) {
	inputDir := t.TempDir()
	outputDir := t.TempDir()

	// Create a test PDF
	pdfDir := t.TempDir()
	pdfPath := filepath.Join(pdfDir, "test.pdf")
	minimalPDF := []byte("%PDF-1.0\n1 0 obj<</Type/Catalog/Pages 2 0 R>>endobj\n2 0 obj<</Type/Pages/Count 0/Kids[]>>endobj\nxref\n0 3\n0000000000 65535 f \n0000000009 00000 n \n0000000052 00000 n \ntrailer<</Size 3/Root 1 0 R>>\nstartxref\n101\n%%EOF")
	if err := os.WriteFile(pdfPath, minimalPDF, 0644); err != nil {
		t.Fatalf("failed to create test PDF: %v", err)
	}

	e := NewTextExtractor(inputDir, outputDir)
	e.ocrTimeout = 10 * time.Second // Long timeout

	ctx, cancel := context.WithCancel(context.Background())
	// Cancel immediately
	cancel()

	_, err := e.ocrViaService(ctx, pdfPath)

	if err == nil {
		t.Error("expected context cancellation error")
	}
	if err != context.Canceled {
		t.Errorf("err = %v, want context.Canceled", err)
	}
}

func TestOcrViaService_Success(t *testing.T) {
	inputDir := t.TempDir()
	outputDir := t.TempDir()

	// Create a test PDF
	pdfDir := t.TempDir()
	pdfPath := filepath.Join(pdfDir, "test.pdf")
	minimalPDF := []byte("%PDF-1.0\n1 0 obj<</Type/Catalog/Pages 2 0 R>>endobj\n2 0 obj<</Type/Pages/Count 0/Kids[]>>endobj\nxref\n0 3\n0000000000 65535 f \n0000000009 00000 n \n0000000052 00000 n \ntrailer<</Size 3/Root 1 0 R>>\nstartxref\n101\n%%EOF")
	if err := os.WriteFile(pdfPath, minimalPDF, 0644); err != nil {
		t.Fatalf("failed to create test PDF: %v", err)
	}

	e := NewTextExtractor(inputDir, outputDir)
	e.ocrTimeout = 5 * time.Second

	ctx := context.Background()

	// Simulate OCR service in background by writing output after a short delay
	go func() {
		time.Sleep(200 * time.Millisecond)

		// Find the job ID from input directory
		entries, err := os.ReadDir(inputDir)
		if err != nil || len(entries) == 0 {
			return
		}

		// Get job ID from filename (without .pdf extension)
		filename := entries[0].Name()
		jobID := filename[:len(filename)-4] // Remove .pdf

		// Write output text file
		outputTextPath := filepath.Join(outputDir, jobID+".txt")
		_ = os.WriteFile(outputTextPath, []byte("OCR extracted text content"), 0644)

		// Write output PDF file (simulating OCRmyPDF output)
		outputPDFPath := filepath.Join(outputDir, jobID+".pdf")
		_ = os.WriteFile(outputPDFPath, minimalPDF, 0644)
	}()

	text, err := e.ocrViaService(ctx, pdfPath)
	if err != nil {
		t.Fatalf("ocrViaService failed: %v", err)
	}

	expected := "OCR extracted text content"
	if text != expected {
		t.Errorf("text = %q, want %q", text, expected)
	}

	// Verify cleanup - output files should be removed
	entries, _ := os.ReadDir(outputDir)
	if len(entries) != 0 {
		t.Errorf("output directory not cleaned up, has %d files", len(entries))
	}
}

func TestExtract_ReturnsEmbeddedMethod(t *testing.T) {
	// This test requires a real PDF with embedded text
	// Skip if we don't have a test fixture
	testPDF := os.Getenv("TEST_PDF_WITH_TEXT")
	if testPDF == "" {
		t.Skip("TEST_PDF_WITH_TEXT not set, skipping embedded text test")
	}

	inputDir := t.TempDir()
	outputDir := t.TempDir()
	e := NewTextExtractor(inputDir, outputDir)

	ctx := context.Background()
	text, method, err := e.Extract(ctx, testPDF)

	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	if method != "embedded" {
		t.Errorf("method = %q, want embedded", method)
	}

	if len(text) < e.minTextLength {
		t.Errorf("text length = %d, want >= %d", len(text), e.minTextLength)
	}
}

func TestExtract_FallsBackToOCR(t *testing.T) {
	// This test simulates a PDF with insufficient embedded text
	// where the OCR fallback is triggered

	inputDir := t.TempDir()
	outputDir := t.TempDir()

	// Create a minimal PDF (will have no text content)
	pdfDir := t.TempDir()
	pdfPath := filepath.Join(pdfDir, "empty.pdf")
	minimalPDF := []byte("%PDF-1.0\n1 0 obj<</Type/Catalog/Pages 2 0 R>>endobj\n2 0 obj<</Type/Pages/Count 0/Kids[]>>endobj\nxref\n0 3\n0000000000 65535 f \n0000000009 00000 n \n0000000052 00000 n \ntrailer<</Size 3/Root 1 0 R>>\nstartxref\n101\n%%EOF")
	if err := os.WriteFile(pdfPath, minimalPDF, 0644); err != nil {
		t.Fatalf("failed to create test PDF: %v", err)
	}

	e := NewTextExtractor(inputDir, outputDir)
	e.ocrTimeout = 5 * time.Second

	ctx := context.Background()

	// Simulate OCR service
	go func() {
		time.Sleep(200 * time.Millisecond)

		entries, err := os.ReadDir(inputDir)
		if err != nil || len(entries) == 0 {
			return
		}

		filename := entries[0].Name()
		jobID := filename[:len(filename)-4]

		outputTextPath := filepath.Join(outputDir, jobID+".txt")
		_ = os.WriteFile(outputTextPath, []byte("OCR result from scanned document"), 0644)

		outputPDFPath := filepath.Join(outputDir, jobID+".pdf")
		_ = os.WriteFile(outputPDFPath, minimalPDF, 0644)
	}()

	text, method, err := e.Extract(ctx, pdfPath)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	if method != "ocr" {
		t.Errorf("method = %q, want ocr", method)
	}

	expected := "OCR result from scanned document"
	if text != expected {
		t.Errorf("text = %q, want %q", text, expected)
	}
}
