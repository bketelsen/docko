# Phase 7: Network Sources - Research

**Researched:** 2026-02-03
**Domain:** SMB/NFS network file access in Go
**Confidence:** MEDIUM

## Summary

This phase implements auto-import of documents from SMB and NFS network shares. The research identifies `hirochachacha/go-smb2` as the standard Go SMB2/3 client library, which is actively maintained and used by rclone. For NFS, the ecosystem is more fragmented: `vmware-archive/go-nfs-client` provides NFSv3 support (but is archived), while `Cyberax/go-nfs-client` supports NFSv4 (designed for AWS EFS). The recommendation is to support NFSv3 via the vmware library as it covers the majority of NFS deployments.

The existing inbox system provides a solid foundation with patterns for polling, event logging, duplicate handling, and error management. Network sources will extend this model with additional fields for protocol type, credentials (encrypted at rest), and network-specific status tracking.

**Primary recommendation:** Use `hirochachacha/go-smb2` for SMB and `vmware/go-nfs-client` for NFS. Implement a polling-based sync approach (not filesystem watching) with 5-minute intervals. Store credentials encrypted using AES-256-GCM with a key derived from SESSION_SECRET.

## Standard Stack

The established libraries/tools for this domain:

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| [hirochachacha/go-smb2](https://github.com/hirochachacha/go-smb2) | v1.1.0 | SMB2/3 client | Used by rclone, comprehensive API, actively maintained |
| [vmware/go-nfs-client](https://github.com/vmware/go-nfs-client) | latest | NFSv3 client | Minimal but functional, widely referenced |
| crypto/aes + crypto/cipher | stdlib | Credential encryption | Standard library, AES-256-GCM |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| golang.org/x/crypto/bcrypt | existing | Password hashing | Already in project for auth |
| context | stdlib | Timeout/cancellation | Connection timeouts |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| vmware/go-nfs-client (NFSv3) | Cyberax/go-nfs-client (NFSv4) | NFSv4 is more complex, less universal; NFSv3 covers most deployments |
| AES-256-GCM encryption | Store credentials unencrypted | Security vs simplicity; encryption required for credentials at rest |

**Installation:**
```bash
go get github.com/hirochachacha/go-smb2
go get github.com/vmware/go-nfs-client/nfs
```

## Architecture Patterns

### Recommended Project Structure
```
internal/
├── network/             # New package for network source handling
│   ├── service.go       # NetworkService coordinates all network sources
│   ├── smb.go           # SMB-specific connection and operations
│   ├── nfs.go           # NFS-specific connection and operations
│   ├── poller.go        # Polling scheduler for continuous sync
│   └── crypto.go        # Credential encryption/decryption
├── inbox/               # Existing - reference for patterns
│   ├── service.go
│   └── watcher.go
```

### Pattern 1: Unified Network Source Interface
**What:** Abstract SMB and NFS behind a common interface for file operations
**When to use:** When processing files from either protocol type
**Example:**
```go
// Source: Project pattern following inbox service design
type NetworkSource interface {
    // Test validates connection can be established
    Test(ctx context.Context) error

    // ListFiles returns PDF files in the source (recursive)
    ListFiles(ctx context.Context) ([]RemoteFile, error)

    // ReadFile copies file content to local path
    ReadFile(ctx context.Context, remotePath string, localPath string) error

    // DeleteFile removes a file from the source
    DeleteFile(ctx context.Context, remotePath string) error

    // MoveFile moves file to subfolder (for post-import action)
    MoveFile(ctx context.Context, remotePath, destPath string) error

    // Close releases resources
    Close() error
}

type RemoteFile struct {
    Path    string
    Name    string
    Size    int64
    ModTime time.Time
}
```

### Pattern 2: Polling-Based Sync (Not Filesystem Watching)
**What:** Network sources use scheduled polling, not real-time watching
**When to use:** Always for network sources (SMB/NFS don't support inotify)
**Example:**
```go
// Source: Project pattern from inbox service
type Poller struct {
    interval time.Duration // Default: 5 minutes
    sources  map[uuid.UUID]*NetworkSourceConfig
    mu       sync.RWMutex
    ctx      context.Context
    cancel   context.CancelFunc
}

func (p *Poller) Run(ctx context.Context) error {
    ticker := time.NewTicker(p.interval)
    defer ticker.Stop()

    for {
        select {
        case <-ticker.C:
            p.syncAllSources()
        case <-ctx.Done():
            return ctx.Err()
        }
    }
}
```

### Pattern 3: SMB Connection Pattern
**What:** Connect to SMB share with credentials
**When to use:** SMB source access
**Example:**
```go
// Source: https://pkg.go.dev/github.com/hirochachacha/go-smb2
func connectSMB(ctx context.Context, host string, share string, user, password string) (*smb2.Share, func(), error) {
    conn, err := net.DialTimeout("tcp", host+":445", 30*time.Second)
    if err != nil {
        return nil, nil, fmt.Errorf("dial: %w", err)
    }

    d := &smb2.Dialer{
        Initiator: &smb2.NTLMInitiator{
            User:     user,
            Password: password,
        },
    }

    session, err := d.DialContext(ctx, conn)
    if err != nil {
        conn.Close()
        return nil, nil, fmt.Errorf("smb dial: %w", err)
    }

    fs, err := session.Mount(share)
    if err != nil {
        session.Logoff()
        conn.Close()
        return nil, nil, fmt.Errorf("mount: %w", err)
    }

    cleanup := func() {
        fs.Umount()
        session.Logoff()
        conn.Close()
    }

    return fs, cleanup, nil
}
```

### Pattern 4: NFS Connection Pattern
**What:** Connect to NFS share
**When to use:** NFS source access
**Example:**
```go
// Source: https://github.com/vmware/go-nfs-client
func connectNFS(ctx context.Context, host, exportPath string) (*nfs.Target, func(), error) {
    mount, err := nfs.DialMount(host)
    if err != nil {
        return nil, nil, fmt.Errorf("dial mount: %w", err)
    }

    // AUTH_UNIX for standard NFS, AUTH_NULL for some servers
    auth := rpc.NewAuthUnix("docko", 0, 0)

    target, err := mount.Mount(exportPath, auth)
    if err != nil {
        mount.Close()
        return nil, nil, fmt.Errorf("mount: %w", err)
    }

    cleanup := func() {
        mount.Unmount()
        mount.Close()
    }

    return target, cleanup, nil
}
```

### Pattern 5: Credential Encryption at Rest
**What:** Encrypt passwords stored in database using AES-256-GCM
**When to use:** Always for credential storage
**Example:**
```go
// Source: https://gist.github.com/kkirsche/e28da6754c39d5e7ea10
type CredentialCrypto struct {
    key []byte // 32 bytes derived from SESSION_SECRET
}

func NewCredentialCrypto(secret string) *CredentialCrypto {
    // Derive 32-byte key using SHA-256
    hash := sha256.Sum256([]byte(secret))
    return &CredentialCrypto{key: hash[:]}
}

func (c *CredentialCrypto) Encrypt(plaintext string) (string, error) {
    block, err := aes.NewCipher(c.key)
    if err != nil {
        return "", err
    }

    gcm, err := cipher.NewGCM(block)
    if err != nil {
        return "", err
    }

    nonce := make([]byte, gcm.NonceSize())
    if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
        return "", err
    }

    ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
    return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func (c *CredentialCrypto) Decrypt(encrypted string) (string, error) {
    data, err := base64.StdEncoding.DecodeString(encrypted)
    if err != nil {
        return "", err
    }

    block, err := aes.NewCipher(c.key)
    if err != nil {
        return "", err
    }

    gcm, err := cipher.NewGCM(block)
    if err != nil {
        return "", err
    }

    nonceSize := gcm.NonceSize()
    if len(data) < nonceSize {
        return "", errors.New("ciphertext too short")
    }

    nonce, ciphertext := data[:nonceSize], data[nonceSize:]
    plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
    if err != nil {
        return "", err
    }

    return string(plaintext), nil
}
```

### Anti-Patterns to Avoid
- **Storing credentials in plain text:** Always encrypt passwords at rest using AES-256-GCM
- **Using filesystem watchers for network shares:** SMB/NFS don't support inotify; use polling
- **Single long-lived connections:** Network connections can become stale; reconnect per sync cycle
- **Blocking on slow network sources:** Use context timeouts; don't let one source block others

## Don't Hand-Roll

Problems that look simple but have existing solutions:

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| SMB protocol | Custom socket protocol | hirochachacha/go-smb2 | Complex protocol with authentication, signing |
| NFS protocol | Custom RPC/XDR handling | vmware/go-nfs-client | ONC RPC and XDR encoding are complex |
| File type detection | Extension-only check | h2non/filetype (existing) | Magic bytes more reliable than extensions |
| Duplicate detection | Path-based comparison | Existing content_hash mechanism | Content hash handles renamed files |
| Credential encryption | Simple XOR or base64 | AES-256-GCM | Base64 is not encryption; XOR is trivially broken |

**Key insight:** Network protocols are deceptively complex. SMB involves multiple protocol versions, authentication methods (NTLM, Kerberos), and signing requirements. NFS involves RPC portmapper, mount protocol, and multiple NFS versions. Use established libraries.

## Common Pitfalls

### Pitfall 1: Connection Timeout on Idle
**What goes wrong:** SMB connections reset after 10-15 minutes of idle time
**Why it happens:** Network equipment and SMB servers close idle connections
**How to avoid:** Reconnect fresh for each sync cycle rather than maintaining persistent connections
**Warning signs:** "connection reset by peer" errors after idle periods

### Pitfall 2: NFS Privileged Port Requirement
**What goes wrong:** NFS mount fails with permission errors when not running as root
**Why it happens:** Traditional NFS requires clients to use ports < 1024 (privileged)
**How to avoid:** Either run as root, or configure NFS server with `insecure` option in exports
**Warning signs:** Mount failures with "permission denied" from non-root processes

### Pitfall 3: Large Directory Listings
**What goes wrong:** Memory exhaustion or timeouts on directories with thousands of files
**Why it happens:** Loading all file metadata at once
**How to avoid:** Process files in batches (configurable batch_size); paginate directory listings
**Warning signs:** Slow sync times, OOM errors on large sources

### Pitfall 4: Unencrypted Credential Storage
**What goes wrong:** Database compromise exposes all network credentials
**Why it happens:** Treating database as secure; forgetting credentials are sensitive
**How to avoid:** Always encrypt credentials at rest; use key derived from SESSION_SECRET
**Warning signs:** Passwords visible in database dumps

### Pitfall 5: Missing Error Visibility
**What goes wrong:** Users don't know why sync is failing
**Why it happens:** Errors logged but not surfaced in UI
**How to avoid:** Store per-source connection_state, last_error, consecutive_failures; show in UI
**Warning signs:** Users report "nothing is syncing" with no visible errors

### Pitfall 6: No Retry Backoff
**What goes wrong:** Rapid retries overwhelm server or network
**Why it happens:** Fixed polling without backoff on errors
**How to avoid:** Auto-disable after N consecutive failures (e.g., 5); require manual re-enable
**Warning signs:** Log flooded with connection errors

## Code Examples

Verified patterns from official sources:

### SMB List Directory Recursively
```go
// Source: https://pkg.go.dev/github.com/hirochachacha/go-smb2
func listPDFs(ctx context.Context, fs *smb2.Share, dir string) ([]RemoteFile, error) {
    var files []RemoteFile

    err := iofs.WalkDir(fs.WithContext(ctx).DirFS(dir), ".", func(path string, d iofs.DirEntry, err error) error {
        if err != nil {
            return err
        }
        if d.IsDir() {
            return nil
        }
        if !strings.EqualFold(filepath.Ext(path), ".pdf") {
            return nil
        }

        info, err := d.Info()
        if err != nil {
            return err
        }

        files = append(files, RemoteFile{
            Path:    filepath.Join(dir, path),
            Name:    d.Name(),
            Size:    info.Size(),
            ModTime: info.ModTime(),
        })
        return nil
    })

    return files, err
}
```

### SMB Read File
```go
// Source: https://pkg.go.dev/github.com/hirochachacha/go-smb2
func downloadFile(ctx context.Context, fs *smb2.Share, remotePath, localPath string) error {
    src, err := fs.WithContext(ctx).Open(remotePath)
    if err != nil {
        return fmt.Errorf("open remote: %w", err)
    }
    defer src.Close()

    dst, err := os.Create(localPath)
    if err != nil {
        return fmt.Errorf("create local: %w", err)
    }
    defer dst.Close()

    _, err = io.Copy(dst, src)
    if err != nil {
        os.Remove(localPath)
        return fmt.Errorf("copy: %w", err)
    }

    return nil
}
```

### NFS List Directory
```go
// Source: https://github.com/vmware/go-nfs-client
func listNFSDirectory(target *nfs.Target, dir string) ([]*nfs.EntryPlus, error) {
    entries, err := target.ReadDirPlus(dir)
    if err != nil {
        return nil, fmt.Errorf("readdir: %w", err)
    }
    return entries, nil
}
```

### Test Connection Pattern
```go
// Source: Project pattern for "Test Connection" button
func testSMBConnection(ctx context.Context, host, share, user, password string) error {
    ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
    defer cancel()

    fs, cleanup, err := connectSMB(ctx, host, share, user, password)
    if err != nil {
        return fmt.Errorf("connection failed: %w", err)
    }
    defer cleanup()

    // Verify we can list the root directory
    _, err = fs.WithContext(ctx).ReadDir(".")
    if err != nil {
        return fmt.Errorf("cannot read share: %w", err)
    }

    return nil
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| SMB1/CIFS | SMB2/3 | ~2012 | SMB1 deprecated for security; use SMB2 minimum |
| System mount + inotify | User-space client + polling | N/A | Enables cross-platform, non-root operation |
| Plain text credentials | Encrypted at rest | Best practice | Required for security compliance |

**Deprecated/outdated:**
- SMB1/CIFS: Security vulnerabilities, disabled by default on modern Windows
- stacktitan/smb: Minimal functionality, not maintained for production use
- libsmbclient-go: CGO dependency, not thread-safe

## Open Questions

Things that couldn't be fully resolved:

1. **NFS authentication methods**
   - What we know: AUTH_NULL works with Linux kernel NFS; AUTH_UNIX is standard
   - What's unclear: Whether Kerberos (AUTH_GSS) is needed for enterprise deployments
   - Recommendation: Support AUTH_UNIX only for initial implementation; document as limitation

2. **vmware/go-nfs-client maintenance status**
   - What we know: Repository is archived as of May 2024
   - What's unclear: Whether forks are actively maintained
   - Recommendation: Use as-is for NFSv3; minimal library with few dependencies; can vendor if needed

3. **SMB signing requirements**
   - What we know: Some domains require SMB signing
   - What's unclear: Whether go-smb2 supports all signing modes
   - Recommendation: Test with target environments; may need fallback to system mount for strict requirements

## Claude's Discretion Recommendations

Based on research, recommended values for discretionary items:

| Item | Recommendation | Rationale |
|------|----------------|-----------|
| Polling interval | 5 minutes | Balance between responsiveness and server load; matches rclone default |
| Failures before auto-disable | 5 consecutive | Prevents log flooding while allowing for transient issues |
| Error log retention | 7 days / 1000 events per source | Sufficient for debugging; prevents unbounded growth |
| Connection timeout | 30 seconds | Matches AWS EFS recommendation; reasonable for WAN |
| Batch size default | 100 files per sync cycle | Prevents memory issues; allows progress visibility |

## Sources

### Primary (HIGH confidence)
- [hirochachacha/go-smb2 pkg.go.dev](https://pkg.go.dev/github.com/hirochachacha/go-smb2) - Complete API documentation
- [hirochachacha/go-smb2 GitHub](https://github.com/hirochachacha/go-smb2) - Usage examples, issues

### Secondary (MEDIUM confidence)
- [vmware/go-nfs-client GitHub](https://github.com/vmware/go-nfs-client) - NFS client source
- [go-smb2 idle timeout issue](https://github.com/hirochachacha/go-smb2/issues/68) - Connection management
- [AWS EFS mount settings](https://docs.aws.amazon.com/efs/latest/ug/mounting-fs-nfs-mount-settings.html) - Timeout recommendations
- [AES-256-GCM Gist](https://gist.github.com/kkirsche/e28da6754c39d5e7ea10) - Encryption pattern

### Tertiary (LOW confidence)
- [Cyberax/go-nfs-client](https://github.com/Cyberax/go-nfs-client) - NFSv4 alternative (not recommended for initial implementation)
- [NetApp NFS idle timeout](https://kb.netapp.com/on-prem/ontap/da/NAS/NAS-KBs/What_is_idle_timeout_value_for_CIFS_and_NFS_protocols) - Default timeout values

## Metadata

**Confidence breakdown:**
- Standard stack: MEDIUM - go-smb2 well-documented; vmware/go-nfs-client archived but functional
- Architecture: HIGH - Follows existing inbox patterns; polling approach is proven
- Pitfalls: MEDIUM - Based on issues/community reports; some may be environment-specific

**Research date:** 2026-02-03
**Valid until:** 2026-03-03 (30 days - stable domain, library versions unlikely to change)
