package network

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/vmware/go-nfs-client/nfs"
	"github.com/vmware/go-nfs-client/nfs/rpc"
)

const (
	nfsConnectTimeout = 30 * time.Second
)

// NFSSource implements NetworkSource for NFSv3 shares.
type NFSSource struct {
	host       string
	exportPath string

	// Connection state
	mount  *nfs.Mount
	target *nfs.Target
}

// NewNFSSource creates a new NFS source.
// NFS typically uses AUTH_UNIX (no password), so only host and export path are needed.
func NewNFSSource(host, exportPath string) *NFSSource {
	return &NFSSource{
		host:       host,
		exportPath: exportPath,
	}
}

// connect establishes an NFS connection.
func (s *NFSSource) connect(ctx context.Context) error {
	// NFS dial with timeout via context
	mount, err := nfs.DialMount(s.host)
	if err != nil {
		return fmt.Errorf("dial mount: %w", err)
	}

	// AUTH_UNIX is standard for most NFS servers
	// Using uid/gid 0 (root) - server may remap based on exports config
	auth := rpc.NewAuthUnix("docko", 0, 0)

	target, err := mount.Mount(s.exportPath, auth.Auth())
	if err != nil {
		mount.Close()
		return fmt.Errorf("mount %s: %w", s.exportPath, err)
	}

	s.mount = mount
	s.target = target
	return nil
}

// disconnect closes the NFS connection.
func (s *NFSSource) disconnect() {
	if s.mount != nil {
		s.mount.Unmount()
		s.mount.Close()
		s.mount = nil
	}
	s.target = nil
}

// Test validates the NFS connection.
func (s *NFSSource) Test(ctx context.Context) error {
	if err := s.connect(ctx); err != nil {
		return err
	}
	defer s.disconnect()

	// Verify we can read the root directory
	_, err := s.target.ReadDirPlus("/")
	if err != nil {
		return fmt.Errorf("read export root: %w", err)
	}
	return nil
}

// ListPDFs returns all PDF files in the export (recursive).
func (s *NFSSource) ListPDFs(ctx context.Context) ([]RemoteFile, error) {
	if err := s.connect(ctx); err != nil {
		return nil, err
	}
	defer s.disconnect()

	var files []RemoteFile
	err := s.walkDir(ctx, "/", &files)
	if err != nil {
		return nil, err
	}
	return files, nil
}

// walkDir recursively walks the directory tree.
func (s *NFSSource) walkDir(ctx context.Context, dir string, files *[]RemoteFile) error {
	// Check context for cancellation
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	entries, err := s.target.ReadDirPlus(dir)
	if err != nil {
		// Log and continue on errors (permission denied, etc)
		return nil
	}

	for _, entry := range entries {
		// Skip . and ..
		if entry.Name() == "." || entry.Name() == ".." {
			continue
		}

		path := filepath.Join(dir, entry.Name())

		if entry.IsDir() {
			// Recurse into subdirectories
			if err := s.walkDir(ctx, path, files); err != nil {
				return err
			}
			continue
		}

		// Only include PDF files
		if !strings.EqualFold(filepath.Ext(entry.Name()), ".pdf") {
			continue
		}

		// Get file info from entry (EntryPlus implements Size() and ModTime())
		*files = append(*files, RemoteFile{
			Path:    path,
			Name:    entry.Name(),
			Size:    entry.Size(),
			ModTime: entry.ModTime(),
		})
	}

	return nil
}

// ReadFile copies file content to the provided writer.
func (s *NFSSource) ReadFile(ctx context.Context, remotePath string, w io.Writer) error {
	if err := s.connect(ctx); err != nil {
		return err
	}
	defer s.disconnect()

	f, err := s.target.Open(remotePath)
	if err != nil {
		return fmt.Errorf("open %s: %w", remotePath, err)
	}
	defer f.Close()

	_, err = io.Copy(w, f)
	if err != nil {
		return fmt.Errorf("copy: %w", err)
	}
	return nil
}

// DeleteFile removes a file from the export.
func (s *NFSSource) DeleteFile(ctx context.Context, remotePath string) error {
	if err := s.connect(ctx); err != nil {
		return err
	}
	defer s.disconnect()

	if err := s.target.Remove(remotePath); err != nil {
		return fmt.Errorf("remove %s: %w", remotePath, err)
	}
	return nil
}

// MoveFile moves a file to a different path within the export.
// Since go-nfs-client doesn't have a Rename operation, this copies then deletes.
func (s *NFSSource) MoveFile(ctx context.Context, remotePath, destPath string) error {
	if err := s.connect(ctx); err != nil {
		return err
	}
	defer s.disconnect()

	// Ensure destination directory exists
	destDir := filepath.Dir(destPath)
	if destDir != "/" && destDir != "." {
		// NFS doesn't have MkdirAll, create directories one by one
		if err := s.ensureDir(destDir); err != nil {
			return fmt.Errorf("mkdir %s: %w", destDir, err)
		}
	}

	// Open source file for reading
	src, err := s.target.Open(remotePath)
	if err != nil {
		return fmt.Errorf("open source %s: %w", remotePath, err)
	}
	defer src.Close()

	// Create destination file
	dst, err := s.target.OpenFile(destPath, 0644)
	if err != nil {
		return fmt.Errorf("create destination %s: %w", destPath, err)
	}
	defer dst.Close()

	// Copy content
	if _, err := io.Copy(dst, src); err != nil {
		return fmt.Errorf("copy %s to %s: %w", remotePath, destPath, err)
	}

	// Delete source file
	if err := s.target.Remove(remotePath); err != nil {
		return fmt.Errorf("remove source %s after copy: %w", remotePath, err)
	}

	return nil
}

// ensureDir creates directory and parents if needed.
func (s *NFSSource) ensureDir(dir string) error {
	// Split path into components
	parts := strings.Split(strings.TrimPrefix(dir, "/"), "/")
	current := ""

	for _, part := range parts {
		current = current + "/" + part

		// Try to create, ignore "exists" error
		_, err := s.target.Mkdir(current, os.FileMode(0755))
		if err != nil {
			// Check if it's an "exists" error - if so, continue
			// NFS errors are not well-typed, so we check the error message
			if !strings.Contains(err.Error(), "exist") {
				return err
			}
		}
	}
	return nil
}

// Close releases resources.
func (s *NFSSource) Close() error {
	s.disconnect()
	return nil
}
