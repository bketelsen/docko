package network

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"net"
	"path/filepath"
	"strings"
	"time"

	"github.com/hirochachacha/go-smb2"
)

const (
	smbPort           = 445
	smbConnectTimeout = 30 * time.Second
)

// SMBSource implements NetworkSource for SMB2/3 shares.
type SMBSource struct {
	host     string
	share    string
	username string
	password string

	// Connection state (created per operation, not persistent)
	conn    net.Conn
	session *smb2.Session
	fs      *smb2.Share
}

// NewSMBSource creates a new SMB source.
// Password should already be decrypted before passing here.
func NewSMBSource(host, share, username, password string) *SMBSource {
	return &SMBSource{
		host:     host,
		share:    share,
		username: username,
		password: password,
	}
}

// connect establishes an SMB connection.
func (s *SMBSource) connect(ctx context.Context) error {
	// Set up connection with timeout
	dialer := &net.Dialer{Timeout: smbConnectTimeout}
	conn, err := dialer.DialContext(ctx, "tcp", fmt.Sprintf("%s:%d", s.host, smbPort))
	if err != nil {
		return fmt.Errorf("dial: %w", err)
	}

	// SMB2 dialer with NTLM auth
	d := &smb2.Dialer{
		Initiator: &smb2.NTLMInitiator{
			User:     s.username,
			Password: s.password,
		},
	}

	session, err := d.DialContext(ctx, conn)
	if err != nil {
		_ = conn.Close()
		return fmt.Errorf("smb dial: %w", err)
	}

	// Mount the share
	share, err := session.Mount(s.share)
	if err != nil {
		_ = session.Logoff()
		_ = conn.Close()
		return fmt.Errorf("mount %s: %w", s.share, err)
	}

	s.conn = conn
	s.session = session
	s.fs = share
	return nil
}

// disconnect closes the SMB connection.
func (s *SMBSource) disconnect() {
	if s.fs != nil {
		_ = s.fs.Umount()
		s.fs = nil
	}
	if s.session != nil {
		_ = s.session.Logoff()
		s.session = nil
	}
	if s.conn != nil {
		_ = s.conn.Close()
		s.conn = nil
	}
}

// Test validates the SMB connection.
func (s *SMBSource) Test(ctx context.Context) error {
	if err := s.connect(ctx); err != nil {
		return err
	}
	defer s.disconnect()

	// Verify we can read the root directory
	_, err := s.fs.ReadDir(".")
	if err != nil {
		return fmt.Errorf("read share root: %w", err)
	}
	return nil
}

// ListPDFs returns all PDF files in the share (recursive).
func (s *SMBSource) ListPDFs(ctx context.Context) ([]RemoteFile, error) {
	if err := s.connect(ctx); err != nil {
		return nil, err
	}
	defer s.disconnect()

	var files []RemoteFile

	// Walk the directory tree using fs.WalkDir
	fsys := s.fs.DirFS(".")
	err := fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, err error) error {
		// Check context for cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if err != nil {
			// Log and continue on errors (e.g., permission denied)
			return nil
		}
		if d.IsDir() {
			return nil
		}

		// Only include PDF files
		if !strings.EqualFold(filepath.Ext(path), ".pdf") {
			return nil
		}

		info, err := d.Info()
		if err != nil {
			return nil // Skip files we can't stat
		}

		files = append(files, RemoteFile{
			Path:    path,
			Name:    d.Name(),
			Size:    info.Size(),
			ModTime: info.ModTime(),
		})
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("walk directory: %w", err)
	}
	return files, nil
}

// ReadFile copies file content to the provided writer.
func (s *SMBSource) ReadFile(ctx context.Context, remotePath string, w io.Writer) error {
	if err := s.connect(ctx); err != nil {
		return err
	}
	defer s.disconnect()

	f, err := s.fs.Open(remotePath)
	if err != nil {
		return fmt.Errorf("open %s: %w", remotePath, err)
	}
	defer func() { _ = f.Close() }()

	_, err = io.Copy(w, f)
	if err != nil {
		return fmt.Errorf("copy: %w", err)
	}
	return nil
}

// DeleteFile removes a file from the share.
func (s *SMBSource) DeleteFile(ctx context.Context, remotePath string) error {
	if err := s.connect(ctx); err != nil {
		return err
	}
	defer s.disconnect()

	if err := s.fs.Remove(remotePath); err != nil {
		return fmt.Errorf("remove %s: %w", remotePath, err)
	}
	return nil
}

// MoveFile moves a file to a different path within the share.
func (s *SMBSource) MoveFile(ctx context.Context, remotePath, destPath string) error {
	if err := s.connect(ctx); err != nil {
		return err
	}
	defer s.disconnect()

	// Ensure destination directory exists
	destDir := filepath.Dir(destPath)
	if err := s.fs.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("mkdir %s: %w", destDir, err)
	}

	if err := s.fs.Rename(remotePath, destPath); err != nil {
		return fmt.Errorf("rename %s to %s: %w", remotePath, destPath, err)
	}
	return nil
}

// Close releases resources. For SMB, connections are per-operation,
// so this is a no-op but satisfies the interface.
func (s *SMBSource) Close() error {
	s.disconnect()
	return nil
}
