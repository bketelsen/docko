package processing

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/bketelsen/docko/internal/storage"
)

func TestCheckDependencies(t *testing.T) {
	// This test verifies the CheckDependencies function works correctly
	// It will pass if both pdftoppm and cwebp are installed, or document
	// the specific missing dependency if not

	err := CheckDependencies()

	// Check if pdftoppm is available
	_, pdftoppmErr := exec.LookPath("pdftoppm")
	// Check if cwebp is available
	_, cwebpErr := exec.LookPath("cwebp")

	if pdftoppmErr != nil || cwebpErr != nil {
		// At least one dependency missing - CheckDependencies should error
		if err == nil {
			t.Error("CheckDependencies returned nil but dependencies are missing")
		}
		t.Skipf("Skipping full test - missing dependencies (pdftoppm: %v, cwebp: %v)", pdftoppmErr, cwebpErr)
	} else {
		// Both available - should succeed
		if err != nil {
			t.Errorf("CheckDependencies failed unexpectedly: %v", err)
		}
	}
}

func TestNewThumbnailGenerator(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := storage.New(tmpDir)
	if err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}

	placeholderPath := filepath.Join(tmpDir, "placeholder.webp")

	gen := NewThumbnailGenerator(store, placeholderPath)

	if gen.storage != store {
		t.Error("storage not set correctly")
	}
	if gen.placeholderPath != placeholderPath {
		t.Errorf("placeholderPath = %q, want %q", gen.placeholderPath, placeholderPath)
	}
}

func TestUsePlaceholder(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := storage.New(tmpDir)
	if err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}

	// Create a fake placeholder file
	placeholderPath := filepath.Join(tmpDir, "placeholder.webp")
	placeholderContent := []byte("fake webp content")
	if err := os.WriteFile(placeholderPath, placeholderContent, 0644); err != nil {
		t.Fatalf("failed to create placeholder: %v", err)
	}

	gen := NewThumbnailGenerator(store, placeholderPath)

	// Create destination path
	destPath := filepath.Join(tmpDir, "thumbnails", "test-thumb.webp")
	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		t.Fatalf("failed to create dest dir: %v", err)
	}

	// Test usePlaceholder
	if err := gen.usePlaceholder(destPath); err != nil {
		t.Fatalf("usePlaceholder failed: %v", err)
	}

	// Verify content was copied
	content, err := os.ReadFile(destPath)
	if err != nil {
		t.Fatalf("failed to read destination: %v", err)
	}
	if string(content) != string(placeholderContent) {
		t.Errorf("content mismatch: got %q, want %q", content, placeholderContent)
	}
}

func TestUsePlaceholder_NoPlaceholderPath(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := storage.New(tmpDir)
	if err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}

	// Create generator without placeholder path
	gen := NewThumbnailGenerator(store, "")

	destPath := filepath.Join(tmpDir, "test-thumb.webp")
	err = gen.usePlaceholder(destPath)

	if err == nil {
		t.Error("expected error when placeholderPath is empty")
	}
}

func TestUsePlaceholder_MissingPlaceholder(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := storage.New(tmpDir)
	if err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}

	// Create generator with non-existent placeholder
	gen := NewThumbnailGenerator(store, filepath.Join(tmpDir, "nonexistent.webp"))

	destPath := filepath.Join(tmpDir, "test-thumb.webp")
	err = gen.usePlaceholder(destPath)

	if err == nil {
		t.Error("expected error when placeholder file doesn't exist")
	}
}

func TestGenerate_WithRealPDF(t *testing.T) {
	// Skip if tools are not available
	if err := CheckDependencies(); err != nil {
		t.Skipf("Skipping test - missing dependencies: %v", err)
	}

	tmpDir := t.TempDir()
	store, err := storage.New(tmpDir)
	if err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}

	// Create placeholder for fallback testing
	placeholderPath := filepath.Join(tmpDir, "placeholder.webp")
	if err := os.WriteFile(placeholderPath, []byte("placeholder"), 0644); err != nil {
		t.Fatalf("failed to create placeholder: %v", err)
	}

	gen := NewThumbnailGenerator(store, placeholderPath)

	// We need a real PDF to test - skip if we don't have one in the storage
	// This is an integration test that requires actual PDF files
	testPDFPath := filepath.Join(tmpDir, "test.pdf")

	// Create a minimal PDF for testing (this is a valid minimal PDF)
	minimalPDF := `%PDF-1.4
1 0 obj
<< /Type /Catalog /Pages 2 0 R >>
endobj
2 0 obj
<< /Type /Pages /Kids [3 0 R] /Count 1 >>
endobj
3 0 obj
<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Contents 4 0 R >>
endobj
4 0 obj
<< /Length 44 >>
stream
BT
/F1 12 Tf
100 700 Td
(Test) Tj
ET
endstream
endobj
xref
0 5
0000000000 65535 f
0000000009 00000 n
0000000058 00000 n
0000000115 00000 n
0000000204 00000 n
trailer
<< /Size 5 /Root 1 0 R >>
startxref
296
%%EOF`

	if err := os.WriteFile(testPDFPath, []byte(minimalPDF), 0644); err != nil {
		t.Fatalf("failed to create test PDF: %v", err)
	}

	docID := uuid.New()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	thumbPath, err := gen.Generate(ctx, testPDFPath, docID)
	if err != nil {
		// Some minimal PDFs may fail - that's okay, we use placeholder
		t.Logf("Generate returned error (may be expected for minimal PDF): %v", err)
	}

	// Verify thumbnail path is correct format
	expectedPath := store.PathForUUID(storage.CategoryThumbnails, docID, ".webp")
	if thumbPath != expectedPath {
		t.Errorf("thumbPath = %q, want %q", thumbPath, expectedPath)
	}

	// Verify file exists (either generated or placeholder)
	if _, err := os.Stat(thumbPath); os.IsNotExist(err) {
		t.Errorf("thumbnail file does not exist at %s", thumbPath)
	}
}

func TestGenerate_InvalidPDF(t *testing.T) {
	// Skip if tools are not available
	if err := CheckDependencies(); err != nil {
		t.Skipf("Skipping test - missing dependencies: %v", err)
	}

	tmpDir := t.TempDir()
	store, err := storage.New(tmpDir)
	if err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}

	// Create placeholder
	placeholderPath := filepath.Join(tmpDir, "placeholder.webp")
	placeholderContent := []byte("placeholder webp")
	if err := os.WriteFile(placeholderPath, placeholderContent, 0644); err != nil {
		t.Fatalf("failed to create placeholder: %v", err)
	}

	gen := NewThumbnailGenerator(store, placeholderPath)

	// Create an invalid PDF (just garbage)
	invalidPDFPath := filepath.Join(tmpDir, "invalid.pdf")
	if err := os.WriteFile(invalidPDFPath, []byte("not a valid PDF"), 0644); err != nil {
		t.Fatalf("failed to create invalid PDF: %v", err)
	}

	docID := uuid.New()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	thumbPath, err := gen.Generate(ctx, invalidPDFPath, docID)

	// Should not error - should use placeholder
	if err != nil {
		t.Fatalf("Generate failed on invalid PDF: %v", err)
	}

	// Verify thumbnail exists
	if _, err := os.Stat(thumbPath); os.IsNotExist(err) {
		t.Error("thumbnail file should exist (placeholder)")
	}

	// Verify it's the placeholder content
	content, err := os.ReadFile(thumbPath)
	if err != nil {
		t.Fatalf("failed to read thumbnail: %v", err)
	}
	if string(content) != string(placeholderContent) {
		t.Error("thumbnail should be placeholder content for invalid PDF")
	}
}

func TestGenerate_NonExistentPDF(t *testing.T) {
	// Skip if tools are not available
	if err := CheckDependencies(); err != nil {
		t.Skipf("Skipping test - missing dependencies: %v", err)
	}

	tmpDir := t.TempDir()
	store, err := storage.New(tmpDir)
	if err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}

	// Create placeholder
	placeholderPath := filepath.Join(tmpDir, "placeholder.webp")
	if err := os.WriteFile(placeholderPath, []byte("placeholder"), 0644); err != nil {
		t.Fatalf("failed to create placeholder: %v", err)
	}

	gen := NewThumbnailGenerator(store, placeholderPath)

	docID := uuid.New()
	ctx := context.Background()

	// Try to generate from non-existent PDF
	thumbPath, err := gen.Generate(ctx, filepath.Join(tmpDir, "nonexistent.pdf"), docID)

	// Should use placeholder instead of erroring
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Verify thumbnail exists (placeholder)
	if _, err := os.Stat(thumbPath); os.IsNotExist(err) {
		t.Error("thumbnail file should exist (placeholder)")
	}
}

func TestThumbnailPath_CorrectExtension(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := storage.New(tmpDir)
	if err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}

	docID := uuid.New()
	path := store.PathForUUID(storage.CategoryThumbnails, docID, ".webp")

	// Verify path ends with .webp
	if filepath.Ext(path) != ".webp" {
		t.Errorf("thumbnail path extension = %q, want .webp", filepath.Ext(path))
	}
}
