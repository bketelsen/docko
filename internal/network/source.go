package network

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/bketelsen/docko/internal/database/sqlc"
)

// RemoteFile represents a file found on a network source.
type RemoteFile struct {
	Path    string    // Full path relative to share root
	Name    string    // Filename only
	Size    int64     // File size in bytes
	ModTime time.Time // Last modification time
}

// NetworkSource defines the interface for network file sources (SMB, NFS).
type NetworkSource interface {
	// Test validates that connection can be established.
	// Returns nil on success, error describing the failure otherwise.
	Test(ctx context.Context) error

	// ListPDFs returns all PDF files in the source (recursive).
	// Files are returned in no particular order.
	ListPDFs(ctx context.Context) ([]RemoteFile, error)

	// ReadFile copies file content from remote path to the provided writer.
	ReadFile(ctx context.Context, remotePath string, w io.Writer) error

	// DeleteFile removes a file from the source.
	DeleteFile(ctx context.Context, remotePath string) error

	// MoveFile moves a file to a subfolder within the source.
	// destPath is relative to share root.
	MoveFile(ctx context.Context, remotePath, destPath string) error

	// Close releases any resources (connections, etc).
	Close() error
}

// NewSourceFromConfig creates a NetworkSource from database configuration.
// The crypto parameter is used to decrypt passwords for SMB sources.
func NewSourceFromConfig(cfg *sqlc.NetworkSource, crypto *CredentialCrypto) (NetworkSource, error) {
	switch cfg.Protocol {
	case sqlc.NetworkProtocolSmb:
		// Decrypt password if present
		password := ""
		if cfg.PasswordEncrypted != nil && *cfg.PasswordEncrypted != "" {
			var err error
			password, err = crypto.Decrypt(*cfg.PasswordEncrypted)
			if err != nil {
				return nil, fmt.Errorf("decrypt password: %w", err)
			}
		}

		username := ""
		if cfg.Username != nil {
			username = *cfg.Username
		}

		return NewSMBSource(cfg.Host, cfg.SharePath, username, password), nil

	case sqlc.NetworkProtocolNfs:
		return NewNFSSource(cfg.Host, cfg.SharePath), nil

	default:
		return nil, fmt.Errorf("unsupported protocol: %s", cfg.Protocol)
	}
}
