package storage

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/google/uuid"
)

// Storage handles file system operations for documents
type Storage struct {
	basePath string
}

// Categories for different file types
const (
	CategoryOriginals  = "originals"
	CategoryThumbnails = "thumbnails"
	CategoryText       = "text"
)

// New creates a Storage instance with the given base path
func New(basePath string) (*Storage, error) {
	if basePath == "" {
		return nil, fmt.Errorf("storage path cannot be empty")
	}

	s := &Storage{basePath: basePath}

	// Ensure base directories exist
	if err := s.EnsureDirectories(); err != nil {
		return nil, fmt.Errorf("failed to create directories: %w", err)
	}

	return s, nil
}

// EnsureDirectories creates the required directory structure
func (s *Storage) EnsureDirectories() error {
	categories := []string{CategoryOriginals, CategoryThumbnails, CategoryText}
	for _, cat := range categories {
		path := filepath.Join(s.basePath, cat)
		if err := os.MkdirAll(path, 0755); err != nil {
			return fmt.Errorf("failed to create %s directory: %w", cat, err)
		}
	}
	return nil
}

// PathForUUID returns the full file path for a UUID in a category
// Uses 2-level sharding: ab/c1/abc12345-...
func (s *Storage) PathForUUID(category string, id uuid.UUID, ext string) string {
	str := id.String()
	return filepath.Join(s.basePath, category, str[0:2], str[2:4], str+ext)
}

// DirForUUID returns the directory path for a UUID in a category
func (s *Storage) DirForUUID(category string, id uuid.UUID) string {
	str := id.String()
	return filepath.Join(s.basePath, category, str[0:2], str[2:4])
}

// CopyAndHash copies a file to destination while computing SHA256 hash
// Returns the hex-encoded hash, file size, and any error
func (s *Storage) CopyAndHash(dst, src string) (string, int64, error) {
	in, err := os.Open(src)
	if err != nil {
		return "", 0, fmt.Errorf("open source: %w", err)
	}
	defer func() { _ = in.Close() }()

	// Ensure destination directory exists
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return "", 0, fmt.Errorf("create directory: %w", err)
	}

	out, err := os.Create(dst)
	if err != nil {
		return "", 0, fmt.Errorf("create destination: %w", err)
	}
	defer func() { _ = out.Close() }()

	hash := sha256.New()
	tee := io.TeeReader(in, hash)

	size, err := io.Copy(out, tee)
	if err != nil {
		// Clean up partial file
		_ = os.Remove(dst)
		return "", 0, fmt.Errorf("copy file: %w", err)
	}

	if err := out.Sync(); err != nil {
		return "", 0, fmt.Errorf("sync file: %w", err)
	}

	return fmt.Sprintf("%x", hash.Sum(nil)), size, nil
}

// HashFile computes SHA256 hash of a file without copying
func (s *Storage) HashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("open file: %w", err)
	}
	defer func() { _ = f.Close() }()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", fmt.Errorf("hash file: %w", err)
	}

	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

// FileExists checks if a file exists at the given path
func (s *Storage) FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// Delete removes a file at the given path
func (s *Storage) Delete(path string) error {
	return os.Remove(path)
}

// BasePath returns the storage base path
func (s *Storage) BasePath() string {
	return s.basePath
}
