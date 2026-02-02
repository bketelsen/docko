# Phase 1: Foundation - Research

**Researched:** 2026-02-02
**Domain:** Document Storage, PostgreSQL Job Queue, Audit Trail
**Confidence:** HIGH

## Summary

This phase establishes the foundational infrastructure for document management: file storage with UUID-based organization, a PostgreSQL-backed job queue with retry logic, and an event-based audit trail.

The research confirms that PostgreSQL's `SELECT FOR UPDATE SKIP LOCKED` pattern is the standard approach for database-backed job queues in Go. This eliminates Redis as a dependency while providing reliable, transactional job processing. The pattern is well-documented and used by production systems.

For file storage, the UUID-prefix directory structure (`ab/c1/abc123...`) is a proven sharding strategy that prevents directory bloat. Combined with SHA256 content hashing for duplicate detection, this provides a robust storage layer.

**Primary recommendation:** Implement a custom job queue using PostgreSQL SKIP LOCKED pattern with sqlc-generated queries. This aligns with the existing codebase patterns (pgx/v5, sqlc, goose migrations) and avoids external dependencies while providing full control over retry/backoff behavior.

## Standard Stack

The established libraries/tools for this domain:

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `github.com/google/uuid` | latest | UUID generation | Already in sqlc config, RFC 4122 compliant, supports v4 (random) and v7 (time-ordered) |
| `crypto/sha256` | stdlib | Content hashing | Standard library, streaming support for large files |
| `github.com/jackc/pgx/v5` | v5.x | PostgreSQL driver | Already in project, native pgx protocol, SKIP LOCKED support |
| `github.com/pressly/goose/v3` | v3.x | Migrations | Already in project, embedded migrations support |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `io` | stdlib | File streaming | Hash computation without loading entire file into memory |
| `os` | stdlib | File operations | Directory creation, file copying |
| `path/filepath` | stdlib | Path manipulation | UUID prefix extraction, safe path joining |
| `encoding/json` | stdlib | Job serialization | Job arguments stored as JSONB |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Custom queue | River (riverqueue.com) | River is excellent but adds dependency; custom gives full control over retry behavior |
| Custom queue | pgq (btubbs/pgq) | Simpler but less flexible retry configuration |
| Custom queue | gue (vgarvardt/gue) | Good option but custom aligns better with sqlc patterns |

**Installation:**
```bash
# Already in go.mod:
# github.com/google/uuid
# github.com/jackc/pgx/v5
# github.com/pressly/goose/v3

# No new dependencies required for Phase 1
```

## Architecture Patterns

### Recommended Project Structure
```
internal/
├── storage/           # File storage operations
│   └── storage.go     # Store, retrieve, hash documents
├── queue/             # Job queue implementation
│   └── queue.go       # Enqueue, dequeue, retry logic
├── document/          # Document service layer
│   └── document.go    # Business logic coordinating storage + database
└── database/
    ├── migrations/
    │   └── 003_documents.sql  # Documents, jobs, events tables
    └── sqlc/
        └── *.go       # Generated queries
```

### Pattern 1: UUID-Prefix Directory Sharding
**What:** Split UUIDs into nested directories using first characters
**When to use:** Storing thousands of files to avoid directory bloat
**Example:**
```go
// For UUID "abc12345-6789-..."
// Creates: STORAGE_PATH/originals/ab/c1/abc12345-6789-....pdf
func (s *Storage) PathForUUID(category string, id uuid.UUID) string {
    str := id.String()
    // Use first 2 chars and next 2 chars as directory levels
    return filepath.Join(s.basePath, category, str[0:2], str[2:4], str)
}
```

### Pattern 2: SKIP LOCKED Job Queue
**What:** Use PostgreSQL row-level locking to claim jobs atomically
**When to use:** Multiple workers processing jobs concurrently
**Example:**
```sql
-- Dequeue jobs atomically
-- name: DequeueJobs :many
WITH next_jobs AS (
    SELECT id FROM jobs
    WHERE status = 'pending'
      AND scheduled_at <= NOW()
      AND attempt < max_attempts
    ORDER BY created_at
    LIMIT $1
    FOR UPDATE SKIP LOCKED
)
UPDATE jobs
SET status = 'processing',
    attempt = attempt + 1,
    started_at = NOW(),
    visible_until = NOW() + INTERVAL '5 minutes'
FROM next_jobs
WHERE jobs.id = next_jobs.id
RETURNING jobs.*;
```

### Pattern 3: Event-Sourced Status
**What:** Derive document status from audit events, not explicit status field
**When to use:** When complete audit trail is required
**Example:**
```sql
-- Document status derived from latest event
-- name: GetDocumentStatus :one
SELECT event_type FROM document_events
WHERE document_id = $1
ORDER BY created_at DESC
LIMIT 1;
```

### Pattern 4: Transactional Job Insertion
**What:** Insert jobs within the same transaction as the triggering operation
**When to use:** Ensuring atomicity between data changes and job creation
**Example:**
```go
func (s *Service) IngestDocument(ctx context.Context, file io.Reader, filename string) error {
    tx, err := s.db.Pool.Begin(ctx)
    if err != nil {
        return err
    }
    defer tx.Rollback(ctx)

    qtx := s.db.Queries.WithTx(tx)

    // Insert document
    doc, err := qtx.CreateDocument(ctx, ...)
    if err != nil {
        return err
    }

    // Insert job in same transaction
    _, err = qtx.EnqueueJob(ctx, ...)
    if err != nil {
        return err
    }

    return tx.Commit(ctx)
}
```

### Anti-Patterns to Avoid
- **Storing full file in database:** Use file system for documents, database for metadata only
- **Polling without index:** Always index (status, scheduled_at, created_at) for job queries
- **Missing visibility timeout:** Jobs stuck in "processing" if worker crashes - use visible_until
- **Synchronous processing:** Always use queue for file operations that may be slow
- **Loading entire file for hashing:** Use io.Copy streaming pattern

## Don't Hand-Roll

Problems that look simple but have existing solutions:

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| UUID generation | Custom random ID | `uuid.New()` or `uuid.NewV7()` | Collisions, RFC compliance, database compatibility |
| File hashing | Manual byte reading | `io.Copy(sha256.New(), file)` | Memory efficiency for large files |
| Backoff calculation | Simple exponential | Full jitter formula | Thundering herd problem without jitter |
| Path construction | String concatenation | `filepath.Join()` | Cross-platform path separators |
| Directory creation | Single mkdir | `os.MkdirAll(..., 0755)` | Creates intermediate directories |

**Key insight:** The "simple" implementations break under scale or edge cases. UUID collisions are rare but catastrophic. Large file hashing without streaming causes OOM. Backoff without jitter causes synchronized retry storms.

## Common Pitfalls

### Pitfall 1: Missing SKIP LOCKED
**What goes wrong:** Multiple workers process the same job
**Why it happens:** Using `FOR UPDATE` without `SKIP LOCKED` causes workers to wait instead of moving to next job
**How to avoid:** Always use `FOR UPDATE SKIP LOCKED` in dequeue queries
**Warning signs:** Duplicate processing, workers blocking each other

### Pitfall 2: No Visibility Timeout
**What goes wrong:** Jobs stuck in "processing" forever after worker crash
**Why it happens:** No mechanism to reclaim abandoned jobs
**How to avoid:** Add `visible_until` column, include in WHERE clause:
```sql
WHERE status = 'pending'
   OR (status = 'processing' AND visible_until < NOW())
```
**Warning signs:** Jobs with status="processing" but no recent updates

### Pitfall 3: Backoff Without Jitter
**What goes wrong:** All failed jobs retry at exactly the same time
**Why it happens:** Pure exponential backoff: 2^n causes synchronization
**How to avoid:** Use full jitter: `random(0, min(cap, base * 2^attempt))`
**Warning signs:** Periodic load spikes on retry intervals

### Pitfall 4: File Hash After Copy
**What goes wrong:** Wasted I/O reading file twice (copy then hash)
**Why it happens:** Treating copy and hash as separate operations
**How to avoid:** Use `io.TeeReader` to hash while copying:
```go
hash := sha256.New()
tee := io.TeeReader(src, hash)
io.Copy(dst, tee)
checksum := hash.Sum(nil)
```
**Warning signs:** Double the expected I/O for large files

### Pitfall 5: Audit Events Without Index
**What goes wrong:** Slow queries when deriving document status
**Why it happens:** "Latest event" query needs to scan all events per document
**How to avoid:** Index on (document_id, created_at DESC)
**Warning signs:** Slow document list pages as event count grows

## Code Examples

Verified patterns from official sources and established projects:

### UUID Generation and Path Sharding
```go
// Source: google/uuid package + common sharding pattern
import "github.com/google/uuid"

func NewDocumentID() uuid.UUID {
    return uuid.New() // v4 random UUID
}

func StoragePath(basePath, category string, id uuid.UUID) string {
    s := id.String() // "ab12c345-6789-..."
    return filepath.Join(basePath, category, s[0:2], s[2:4], s+".pdf")
}
// Result: /storage/originals/ab/12/ab12c345-6789-....pdf
```

### Streaming File Hash
```go
// Source: crypto/sha256 package docs + io.Copy pattern
import (
    "crypto/sha256"
    "fmt"
    "io"
    "os"
)

func HashFile(path string) (string, error) {
    f, err := os.Open(path)
    if err != nil {
        return "", fmt.Errorf("open file: %w", err)
    }
    defer f.Close()

    h := sha256.New()
    if _, err := io.Copy(h, f); err != nil {
        return "", fmt.Errorf("hash file: %w", err)
    }

    return fmt.Sprintf("%x", h.Sum(nil)), nil
}
```

### Hash While Copying
```go
// Source: io.TeeReader pattern
func CopyAndHash(dst, src string) (string, error) {
    in, err := os.Open(src)
    if err != nil {
        return "", err
    }
    defer in.Close()

    out, err := os.Create(dst)
    if err != nil {
        return "", err
    }
    defer out.Close()

    hash := sha256.New()
    tee := io.TeeReader(in, hash)

    if _, err := io.Copy(out, tee); err != nil {
        return "", err
    }

    return fmt.Sprintf("%x", hash.Sum(nil)), nil
}
```

### Exponential Backoff with Full Jitter
```go
// Source: AWS Builders' Library algorithm
import (
    "math"
    "math/rand"
    "time"
)

func NextRetryDelay(attempt int, base, cap time.Duration) time.Duration {
    // Full jitter: random(0, min(cap, base * 2^attempt))
    backoff := float64(base) * math.Pow(2, float64(attempt))
    if backoff > float64(cap) {
        backoff = float64(cap)
    }
    jittered := rand.Float64() * backoff
    return time.Duration(jittered)
}

// Example usage:
// attempt 0: random(0, 1s)   -> avg 500ms
// attempt 1: random(0, 2s)   -> avg 1s
// attempt 2: random(0, 4s)   -> avg 2s
// attempt 3: random(0, 8s)   -> avg 4s (capped at max)
```

### Job Queue Schema (sqlc compatible)
```sql
-- Source: PostgreSQL SKIP LOCKED pattern + JSONB best practices
CREATE TYPE job_status AS ENUM ('pending', 'processing', 'completed', 'failed');

CREATE TABLE jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    queue_name TEXT NOT NULL DEFAULT 'default',
    job_type TEXT NOT NULL,
    payload JSONB NOT NULL,
    status job_status NOT NULL DEFAULT 'pending',
    attempt INT NOT NULL DEFAULT 0,
    max_attempts INT NOT NULL DEFAULT 3,
    scheduled_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    visible_until TIMESTAMPTZ,
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    last_error TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Critical index for dequeue performance
CREATE INDEX idx_jobs_dequeue ON jobs (queue_name, status, scheduled_at, created_at)
WHERE status IN ('pending', 'processing');
```

### Document Events Schema
```sql
-- Source: Event sourcing pattern for audit trail
CREATE TABLE document_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    document_id UUID NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
    event_type TEXT NOT NULL,  -- 'ingested', 'hashed', 'text_extracted', etc.
    payload JSONB,             -- Event-specific data
    error_message TEXT,        -- For failed events
    duration_ms INT,           -- Processing time
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_document_events_lookup ON document_events (document_id, created_at DESC);
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| UUID v1 (MAC-based) | UUID v4 or v7 | 2023+ | v7 is time-ordered, better for database indexes |
| Advisory locks | SKIP LOCKED | PostgreSQL 9.5+ | Simpler, no lock management code |
| Redis for queues | PostgreSQL SKIP LOCKED | 2020+ | Fewer dependencies, transactional guarantees |
| Separate status field | Event-sourced status | N/A | Complete audit trail, no sync issues |
| Load file then hash | Stream hash with io.Copy | Always | Memory efficiency |

**Deprecated/outdated:**
- `github.com/pborman/uuid`: Superseded by `google/uuid`
- `github.com/jackc/pgxjob`: Archived August 2025, no longer maintained
- `lib/pq` driver: pgx is now standard for Go PostgreSQL

## Open Questions

Things that couldn't be fully resolved:

1. **Optimal visibility timeout duration**
   - What we know: Should be longer than max expected job processing time
   - What's unclear: Exact duration for PDF processing (depends on file size)
   - Recommendation: Start with 5 minutes, make configurable, add heartbeat if needed

2. **Queue polling interval**
   - What we know: Trade-off between latency and database load
   - What's unclear: Optimal interval for this workload
   - Recommendation: Start with 1 second, make configurable, consider LISTEN/NOTIFY later

3. **Worker count**
   - What we know: Should be based on CPU/memory, not fixed
   - What's unclear: Optimal count for PDF processing workloads
   - Recommendation: Default to `runtime.NumCPU()`, make configurable

## Sources

### Primary (HIGH confidence)
- [google/uuid package](https://pkg.go.dev/github.com/google/uuid) - UUID generation API, version methods
- [crypto/sha256 package](https://pkg.go.dev/crypto/sha256) - Streaming hash interface
- [pressly/goose package](https://pkg.go.dev/github.com/pressly/goose/v3) - Migration syntax and patterns
- [PostgreSQL JSON docs](https://www.postgresql.org/docs/current/datatype-json.html) - JSONB best practices

### Secondary (MEDIUM confidence)
- [AWS Builders' Library](https://aws.amazon.com/builders-library/timeouts-retries-and-backoff-with-jitter/) - Exponential backoff with jitter formulas
- [River Queue docs](https://riverqueue.com/docs) - Job queue patterns for Go + PostgreSQL
- [PostgreSQL SKIP LOCKED article](https://www.inferable.ai/blog/posts/postgres-skip-locked) - SKIP LOCKED pattern explanation

### Tertiary (LOW confidence)
- [Blog: Implementing Postgres job queue](https://aminediro.com/posts/pg_job_queue/) - Complete schema example
- [Blog: Queueing with PostgreSQL and Go](https://robinverton.de/blog/queueing-with-postgresql-and-go/) - Implementation patterns

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - Using stdlib and already-installed packages
- Architecture: HIGH - Well-documented PostgreSQL patterns
- Pitfalls: HIGH - Verified against multiple sources

**Research date:** 2026-02-02
**Valid until:** 2026-03-02 (30 days - stable domain, established patterns)
