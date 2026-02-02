# Pitfalls Research

**Domain:** PDF Document Management System
**Context:** Brownfield Go application, PostgreSQL, tens of thousands of PDFs, network shares
**Confidence:** MEDIUM (based on training knowledge, not current web research)

---

## File Handling

### Critical: Incomplete File Writes Leading to Corruption

**What goes wrong:** Files are partially written when the process crashes, disk fills up, or network connection drops during copy. The system then indexes a corrupted PDF that cannot be opened.

**Why it happens:**
- Using `io.Copy` without atomic write patterns
- Not verifying file integrity after copy
- Not handling disk-full conditions

**Warning signs:**
- PDF viewers fail to open some documents
- File sizes don't match source
- Index contains documents that error on retrieval

**Prevention:**
1. Write to temporary file first, then atomic rename (`os.Rename` is atomic on same filesystem)
2. Verify file size matches expected after copy
3. Optionally verify PDF header bytes (`%PDF-`) after copy
4. Monitor disk space, fail ingestion before disk is full

**Code pattern:**
```go
// Write to temp file in same directory (ensures same filesystem)
tmpFile := filepath.Join(destDir, ".tmp-"+uuid.NewString())
defer os.Remove(tmpFile) // cleanup on failure

// Copy to temp
if err := copyFile(src, tmpFile); err != nil {
    return fmt.Errorf("copy to temp: %w", err)
}

// Verify size
if info, _ := os.Stat(tmpFile); info.Size() != expectedSize {
    return fmt.Errorf("size mismatch: got %d, want %d", info.Size(), expectedSize)
}

// Atomic rename to final destination
if err := os.Rename(tmpFile, finalPath); err != nil {
    return fmt.Errorf("atomic rename: %w", err)
}
```

**Phase:** Ingestion queue (Phase 1)

---

### Critical: Race Conditions on Concurrent File Access

**What goes wrong:** Multiple workers process the same file simultaneously, causing duplicate entries, corrupted metadata, or lost files.

**Why it happens:**
- File watcher triggers twice for same file (common with network shares)
- Duplicate detection runs before ingestion completes
- Two processes scan the same inbox

**Warning signs:**
- Duplicate documents in database with same content hash
- Missing documents that were definitely ingested
- Inconsistent counts between filesystem and database

**Prevention:**
1. Use database-level locking (advisory locks or row-level locks) before processing
2. Implement idempotent ingestion keyed on content hash
3. Use `SELECT ... FOR UPDATE SKIP LOCKED` pattern for job claiming
4. Add filesystem-level locks during file operations

**Code pattern:**
```go
// Claim job with advisory lock
tx, _ := db.Begin(ctx)
defer tx.Rollback()

// Advisory lock based on file hash
lockKey := int64(crc32.ChecksumIEEE([]byte(contentHash)))
var acquired bool
tx.QueryRow("SELECT pg_try_advisory_xact_lock($1)", lockKey).Scan(&acquired)
if !acquired {
    return nil // Another worker has this file
}

// Process file...
tx.Commit()
```

**Phase:** Ingestion queue (Phase 1)

---

### High: Storage Exhaustion Without Warning

**What goes wrong:** Disk fills up during batch ingestion, causing cascading failures. Partial ingestions leave orphaned files.

**Why it happens:**
- No pre-flight check for available space
- Thumbnails generate faster than expected
- Log files consume unexpected space
- Temp files not cleaned up on failure

**Warning signs:**
- Sudden increase in ingestion failures
- "no space left on device" errors
- Database transaction failures

**Prevention:**
1. Check available space before starting ingestion batch
2. Implement configurable storage thresholds (e.g., stop at 90% full)
3. Separate storage for originals, thumbnails, and temp files
4. Clean temp directory on startup
5. Monitor storage with Prometheus/metrics

**Implementation:**
```go
func checkStorage(path string, minFreeBytes uint64) error {
    var stat syscall.Statfs_t
    if err := syscall.Statfs(path, &stat); err != nil {
        return fmt.Errorf("statfs: %w", err)
    }
    freeBytes := stat.Bavail * uint64(stat.Bsize)
    if freeBytes < minFreeBytes {
        return fmt.Errorf("insufficient storage: %d bytes free, need %d", freeBytes, minFreeBytes)
    }
    return nil
}
```

**Phase:** Infrastructure setup (Phase 1), monitoring (later phase)

---

### Medium: File Handle Leaks

**What goes wrong:** Open file handles accumulate, eventually hitting OS limits. System becomes unresponsive.

**Why it happens:**
- Not closing files after error paths
- PDF libraries holding file handles open
- Network share connections not released

**Warning signs:**
- "too many open files" errors
- Increasing memory usage
- Slow file operations

**Prevention:**
1. Always use `defer file.Close()` immediately after open
2. Use connection pooling for network shares
3. Process PDFs by reading into memory, then closing handle
4. Set reasonable file descriptor limits in deployment

**Phase:** All file processing phases

---

## Queue Processing

### Critical: Lost Jobs on Crash

**What goes wrong:** Worker crashes mid-processing, job disappears. Document never gets indexed or tagged.

**Why it happens:**
- Job marked as "in progress" but worker dies
- No heartbeat mechanism to detect stuck jobs
- Transaction not committed before ACK

**Warning signs:**
- Jobs stuck in "processing" state indefinitely
- Count of "pending" never decreases despite workers running
- Documents missing from search despite being in originals folder

**Prevention:**
1. Use "at-least-once" delivery with idempotent handlers
2. Implement job timeout and automatic retry
3. Store job state in PostgreSQL with visibility timeout pattern
4. Use `SELECT ... FOR UPDATE SKIP LOCKED` for job claiming
5. Add heartbeat/progress updates for long-running jobs

**Code pattern:**
```go
// Claim job with visibility timeout
UPDATE document_jobs
SET status = 'processing',
    locked_until = NOW() + interval '5 minutes',
    attempts = attempts + 1
WHERE id = (
    SELECT id FROM document_jobs
    WHERE status = 'pending'
       OR (status = 'processing' AND locked_until < NOW())
    ORDER BY created_at
    LIMIT 1
    FOR UPDATE SKIP LOCKED
)
RETURNING *
```

**Phase:** Queue infrastructure (Phase 1)

---

### Critical: Duplicate Processing

**What goes wrong:** Same document is processed multiple times, wasting resources (especially expensive AI calls) or creating duplicate entries.

**Why it happens:**
- Job visibility timeout expires while still processing (slow network share)
- Retry logic too aggressive
- File watcher fires multiple events for same file

**Warning signs:**
- AI costs higher than expected
- Same document appears multiple times in results
- Processing time much higher than document count suggests

**Prevention:**
1. Use content-hash-based deduplication at ingestion
2. Mark documents as processed before AI tagging (not after)
3. Use longer visibility timeouts for AI jobs (10+ minutes)
4. Debounce file watcher events (wait 2-5 seconds after last event)
5. Store processing receipts for each stage

**Implementation:**
```go
// Debounce file events
func debounce(events <-chan string, wait time.Duration) <-chan string {
    out := make(chan string)
    go func() {
        pending := make(map[string]time.Time)
        ticker := time.NewTicker(wait / 2)
        for {
            select {
            case path := <-events:
                pending[path] = time.Now()
            case <-ticker.C:
                now := time.Now()
                for path, t := range pending {
                    if now.Sub(t) >= wait {
                        out <- path
                        delete(pending, path)
                    }
                }
            }
        }
    }()
    return out
}
```

**Phase:** Queue infrastructure (Phase 1), AI integration (Phase 3)

---

### High: Stuck Queues Due to Poison Messages

**What goes wrong:** One malformed document causes worker to crash or hang repeatedly. All jobs behind it get stuck.

**Why it happens:**
- PDF parsing library crashes on malformed PDF
- Infinite loop in text extraction
- Out of memory on large document

**Warning signs:**
- Same job retrying indefinitely
- Worker restarts in loop
- Queue depth growing despite workers running

**Prevention:**
1. Implement max retry limit (e.g., 3 attempts)
2. Move failed jobs to dead-letter queue after max retries
3. Add processing timeouts with context cancellation
4. Isolate PDF parsing in separate goroutine with recovery
5. Admin UI to view and retry dead-letter jobs

**Code pattern:**
```go
// Process with timeout and panic recovery
func processWithTimeout(ctx context.Context, job Job, timeout time.Duration) error {
    ctx, cancel := context.WithTimeout(ctx, timeout)
    defer cancel()

    errCh := make(chan error, 1)
    go func() {
        defer func() {
            if r := recover(); r != nil {
                errCh <- fmt.Errorf("panic: %v", r)
            }
        }()
        errCh <- processJob(ctx, job)
    }()

    select {
    case err := <-errCh:
        return err
    case <-ctx.Done():
        return fmt.Errorf("timeout after %v", timeout)
    }
}
```

**Phase:** Queue infrastructure (Phase 1)

---

### Medium: Queue Starvation

**What goes wrong:** One queue type consumes all workers, leaving other queues waiting indefinitely.

**Why it happens:**
- Single worker pool for all job types
- AI tagging jobs take 30 seconds each, starving fast jobs
- Burst of ingestion jobs blocks duplicate detection

**Warning signs:**
- Some queues have high latency while others are idle
- Processing order doesn't match priority expectations

**Prevention:**
1. Separate worker pools per queue type
2. Configure concurrency per queue (e.g., ingestion: 4, AI: 2, indexing: 4)
3. Implement priority queues for time-sensitive operations
4. Rate limit expensive operations (AI) separately

**Phase:** Queue infrastructure (Phase 1)

---

## Full-Text Search

### Critical: Unbounded Index Growth

**What goes wrong:** PostgreSQL full-text index grows without bound, queries slow to unusable speed, disk space exhausted.

**Why it happens:**
- Indexing entire PDF content without limit
- No compression or cleanup of old index data
- GIN index bloat from updates

**Warning signs:**
- Search queries taking >1 second
- Index size growing faster than document count
- VACUUM taking very long

**Prevention:**
1. Limit indexed content per document (first N pages or N KB)
2. Use `to_tsvector` with appropriate text configuration
3. Regular VACUUM and REINDEX maintenance
4. Monitor index size vs document count ratio

**Implementation:**
```sql
-- Index configuration
CREATE INDEX idx_documents_fts ON documents
USING GIN (to_tsvector('english',
    COALESCE(title, '') || ' ' ||
    LEFT(content, 100000)  -- Limit to 100KB
));

-- Maintenance job
REINDEX INDEX CONCURRENTLY idx_documents_fts;
VACUUM ANALYZE documents;
```

**Phase:** Full-text indexing (Phase 2)

---

### High: Poor Search Relevance

**What goes wrong:** Users search for "invoice" and get irrelevant results, or miss documents that should match.

**Why it happens:**
- Wrong PostgreSQL text configuration (language)
- Not using weights for title vs content
- No stemming or synonym support
- Indexing non-text content (base64, metadata)

**Warning signs:**
- Users complain "I know this document exists but can't find it"
- Search returns too many results
- Exact matches ranked below partial matches

**Prevention:**
1. Use `ts_rank` with appropriate weights (title > content)
2. Configure proper text search dictionary
3. Strip non-text content before indexing
4. Add phrase matching for exact queries
5. Test search quality with real queries

**Implementation:**
```sql
-- Weighted search
SELECT
    id,
    ts_rank(
        setweight(to_tsvector('english', title), 'A') ||
        setweight(to_tsvector('english', content), 'B'),
        plainto_tsquery('english', $1)
    ) AS rank
FROM documents
WHERE to_tsvector('english', title || ' ' || content) @@ plainto_tsquery('english', $1)
ORDER BY rank DESC
LIMIT 50;
```

**Phase:** Full-text indexing (Phase 2), search UI (Phase 2)

---

### Medium: Search Query Injection

**What goes wrong:** User input breaks tsquery syntax, causing errors or unexpected results.

**Why it happens:**
- Using `to_tsquery` with raw user input
- Special characters not escaped

**Warning signs:**
- 500 errors on certain search terms
- Error logs showing "syntax error in tsquery"

**Prevention:**
1. Use `plainto_tsquery` or `websearch_to_tsquery` instead of `to_tsquery`
2. `websearch_to_tsquery` handles quoted phrases and operators naturally

**Code pattern:**
```go
// Safe search - use websearch_to_tsquery
query := `
    SELECT * FROM documents
    WHERE fts_vector @@ websearch_to_tsquery('english', $1)
    ORDER BY ts_rank(fts_vector, websearch_to_tsquery('english', $1)) DESC
`
```

**Phase:** Search UI (Phase 2)

---

### Medium: Memory Bloat During Indexing

**What goes wrong:** Indexing large PDFs consumes excessive memory, OOM kills worker.

**Why it happens:**
- Loading entire PDF into memory
- Storing full extracted text in memory before insert
- No streaming for large documents

**Warning signs:**
- Worker memory usage spikes during certain documents
- OOM kills in logs
- Indexing stops on large documents

**Prevention:**
1. Stream PDF processing page-by-page
2. Limit indexed content per document
3. Process in batches, not all at once
4. Set memory limits and let job retry fail gracefully

**Phase:** Full-text indexing (Phase 2)

---

## AI Integration

### Critical: Runaway API Costs

**What goes wrong:** AI tagging costs $500/month instead of expected $50. Budget exhausted in days.

**Why it happens:**
- No per-document cost tracking
- Processing full PDFs instead of first few pages
- Retries on rate limits without backoff (paying for same doc multiple times)
- Testing against production API

**Warning signs:**
- API costs exceed budget
- Token usage much higher than document count suggests
- Same document processed multiple times

**Prevention:**
1. Track and display cost per document and total spend
2. Configure maximum pages to send to AI (default: 3-5)
3. Implement daily/monthly spend limits
4. Use mock API in development
5. Log estimated cost before API call

**Implementation:**
```go
type AIConfig struct {
    MaxPagesPerDoc    int           // Default: 3
    DailySpendLimit   float64       // Default: $10
    MonthlySpendLimit float64       // Default: $100
    CostPerInputToken float64       // e.g., 0.000003 for GPT-4
}

func (ai *AITagger) estimateCost(pages int, avgCharsPerPage int) float64 {
    tokens := pages * avgCharsPerPage / 4 // rough estimate
    return float64(tokens) * ai.config.CostPerInputToken
}
```

**Phase:** AI tagging (Phase 3)

---

### Critical: Rate Limit Handling

**What goes wrong:** AI provider rate limits cause cascade failures, jobs retry immediately and get rate limited again, creating hot loop.

**Why it happens:**
- No exponential backoff on rate limits
- Treating rate limit as permanent failure
- Multiple workers hitting rate limit simultaneously

**Warning signs:**
- 429 errors in logs
- AI queue processing very slowly
- Increased error rates during batch processing

**Prevention:**
1. Implement exponential backoff with jitter
2. Rate limit errors should NOT increment retry count
3. Use circuit breaker pattern for AI calls
4. Add concurrency limit for AI worker (1-2 max)

**Implementation:**
```go
func callWithBackoff(ctx context.Context, fn func() error) error {
    backoff := 1 * time.Second
    maxBackoff := 5 * time.Minute

    for attempt := 0; attempt < 10; attempt++ {
        err := fn()
        if err == nil {
            return nil
        }

        if isRateLimit(err) {
            // Don't count against retries, just wait
            jitter := time.Duration(rand.Float64() * float64(backoff))
            time.Sleep(backoff + jitter)
            backoff = min(backoff*2, maxBackoff)
            continue
        }

        return err // Permanent failure
    }
    return fmt.Errorf("max retries exceeded")
}
```

**Phase:** AI tagging (Phase 3)

---

### High: Poor Prompt Engineering Leading to Useless Tags

**What goes wrong:** AI returns generic tags like "document", "text", "important" that provide no value.

**Why it happens:**
- Prompt doesn't specify tag format
- No example tags provided
- Asking for too many tags
- Not constraining to useful categories

**Warning signs:**
- Most documents have same tags
- Tags don't help with retrieval
- Users ignore AI-generated tags

**Prevention:**
1. Provide example tags in prompt
2. Specify tag categories (document type, topic, correspondent)
3. Ask for 3-5 specific tags, not unlimited
4. Include rejection option ("no applicable tags")
5. A/B test prompts with real documents

**Prompt pattern:**
```
Analyze this document and extract tags in these categories:

1. Document type (pick ONE): invoice, receipt, contract, letter, statement, report, manual, other
2. Topic tags (pick 1-3): [list relevant to your domain]
3. Correspondent: Company or person name if identifiable

Respond in JSON format:
{"type": "invoice", "topics": ["utilities", "electricity"], "correspondent": "Pacific Gas & Electric"}

If no clear match, use null for that field.
```

**Phase:** AI tagging (Phase 3)

---

### Medium: Inconsistent AI Results

**What goes wrong:** Same document gets different tags on retry, making results unpredictable.

**Why it happens:**
- Using high temperature setting
- Not using deterministic/seed parameters
- Prompt variations between calls

**Warning signs:**
- Tag churn on reprocessing
- User confusion about tag meaning

**Prevention:**
1. Use temperature=0 for consistent results
2. Use seed parameter if available
3. Store and reuse prompt versions
4. Hash document content + prompt to create cache key

**Phase:** AI tagging (Phase 3)

---

## Network Shares

### Critical: Authentication Token Expiry

**What goes wrong:** Long-running batch jobs fail mid-process when SMB/NFS authentication expires.

**Why it happens:**
- Kerberos tickets expire (default 8-10 hours)
- Session timeout on NAS devices
- Network interruption invalidates session

**Warning signs:**
- Jobs that worked for 2 hours suddenly fail
- Authentication errors in logs
- Files accessible manually but not via application

**Prevention:**
1. Re-authenticate before each batch (not once at startup)
2. Implement connection health checks with reconnect
3. Use connection pooling with session refresh
4. Handle auth errors gracefully and retry with fresh connection

**Implementation:**
```go
type ShareConnection struct {
    config     ShareConfig
    conn       *smb.Session
    lastUsed   time.Time
    maxAge     time.Duration
}

func (sc *ShareConnection) GetConnection() (*smb.Session, error) {
    if sc.conn == nil || time.Since(sc.lastUsed) > sc.maxAge {
        if sc.conn != nil {
            sc.conn.Close()
        }
        conn, err := smb.Connect(sc.config)
        if err != nil {
            return nil, fmt.Errorf("reconnect: %w", err)
        }
        sc.conn = conn
    }
    sc.lastUsed = time.Now()
    return sc.conn, nil
}
```

**Phase:** Network share integration (Phase 1)

---

### Critical: Network Timeout Handling

**What goes wrong:** Network share becomes temporarily unreachable, causing jobs to hang indefinitely or fail permanently.

**Why it happens:**
- No timeout on file operations
- Network switch reboot, brief outage
- NAS enters sleep mode

**Warning signs:**
- Workers stuck on network operations
- Intermittent failures that work on retry
- Batch jobs take much longer than expected

**Prevention:**
1. Set explicit timeouts on all network operations
2. Implement retry with exponential backoff for transient errors
3. Use context with timeout for all share operations
4. Add health check endpoint for share connectivity

**Implementation:**
```go
func (s *ShareClient) ReadFile(ctx context.Context, path string) ([]byte, error) {
    ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
    defer cancel()

    // Retry transient errors
    var lastErr error
    for attempt := 0; attempt < 3; attempt++ {
        data, err := s.readFileOnce(ctx, path)
        if err == nil {
            return data, nil
        }
        if isTransientError(err) {
            lastErr = err
            time.Sleep(time.Duration(attempt+1) * time.Second)
            continue
        }
        return nil, err // Permanent error
    }
    return nil, fmt.Errorf("transient error after retries: %w", lastErr)
}
```

**Phase:** Network share integration (Phase 1)

---

### High: File Locking Conflicts

**What goes wrong:** Application tries to read file that user has open, or two processes try to copy same file.

**Why it happens:**
- User editing document while ingestion runs
- Multiple docko instances scanning same share
- NAS-level locks not respected by Go libraries

**Warning signs:**
- "file in use" or "sharing violation" errors
- Corrupted file copies
- Inconsistent file reads

**Prevention:**
1. Copy files to local temp before processing
2. Implement retry on lock errors with backoff
3. Use non-blocking reads where possible
4. Document: users should not edit files in watched folders

**Phase:** Network share integration (Phase 1)

---

### Medium: Path Encoding Issues

**What goes wrong:** Files with non-ASCII characters or special characters fail to process.

**Why it happens:**
- SMB vs NFS path encoding differences
- Windows vs Unix path conventions
- Unicode normalization issues (NFC vs NFD)

**Warning signs:**
- Some files not found despite existing
- Errors on files with accented characters
- Duplicate files with "same" name

**Prevention:**
1. Normalize all paths to UTF-8 NFC
2. Test with files containing Unicode characters
3. Handle Windows-style paths (backslash) if source is Windows share
4. Log original path and normalized path for debugging

**Phase:** Network share integration (Phase 1)

---

### Medium: Large Directory Enumeration

**What goes wrong:** Listing directory with 50,000 files takes 5 minutes and times out.

**Why it happens:**
- SMB protocol overhead per file
- Loading all metadata upfront
- No pagination in directory listing

**Warning signs:**
- Scan jobs take extremely long
- Memory usage spikes during scans
- Timeouts during directory listing

**Prevention:**
1. Use streaming/iterative directory listing
2. Cache directory state, detect changes incrementally
3. Store "last scan" cursor to resume interrupted scans
4. Process files as found, don't wait for full listing

**Phase:** Network share integration (Phase 1)

---

## Duplicate Detection

### Critical: Hash Collisions on Different Documents

**What goes wrong:** Two different PDFs have same hash, one gets silently discarded.

**Why it happens:**
- Using weak hash (MD5, CRC32)
- Hashing only metadata, not content
- Hash computed on different data than stored

**Warning signs:**
- Documents reported as duplicates that are clearly different
- Lost documents after ingestion

**Prevention:**
1. Use SHA-256 for content hashing (collision-resistant)
2. Hash full file content, not just first N bytes
3. On duplicate detection, verify byte-for-byte if unsure
4. Log both files when duplicate detected

**Implementation:**
```go
func hashFile(path string) (string, error) {
    f, err := os.Open(path)
    if err != nil {
        return "", err
    }
    defer f.Close()

    h := sha256.New()
    if _, err := io.Copy(h, f); err != nil {
        return "", err
    }
    return hex.EncodeToString(h.Sum(nil)), nil
}
```

**Phase:** Duplicate detection queue (Phase 1)

---

### High: Near-Duplicate Detection Complexity

**What goes wrong:** System treats slightly different versions as duplicates (wrong), or fails to detect actual duplicates with minor differences (also wrong).

**Why it happens:**
- PDF regeneration changes internal structure
- Metadata differences (creation date)
- Minor content edits create new document

**Warning signs:**
- Version updates treated as duplicates
- Identical content not matched as duplicate
- User confusion about which version is "canonical"

**Prevention:**
1. Separate "exact duplicate" (content hash) from "near duplicate" (fuzzy)
2. For exact: use SHA-256 of full content
3. For near: extract text, compute similarity (not for v1)
4. Let user decide on near-duplicates, don't auto-merge
5. Keep all versions, mark as related

**Recommendation for v1:**
- Implement exact-match only (SHA-256)
- Defer near-duplicate detection to v2
- Allow user to manually mark documents as related

**Phase:** Duplicate detection (Phase 1 for exact, defer fuzzy to v2)

---

### Medium: Performance on Large Archive

**What goes wrong:** Duplicate check requires comparing against 50,000 existing hashes, becomes slow.

**Why it happens:**
- Linear scan of all hashes
- No index on hash column
- Hash computed in application, not database

**Warning signs:**
- Duplicate detection queue backs up
- Increasing latency as archive grows

**Prevention:**
1. Index hash column in PostgreSQL
2. Use single query: `SELECT EXISTS(SELECT 1 FROM documents WHERE content_hash = $1)`
3. Consider bloom filter for fast negative check (optimization)

```sql
CREATE INDEX idx_documents_content_hash ON documents(content_hash);
```

**Phase:** Duplicate detection (Phase 1)

---

### Medium: Correspondent Deduplication Failures

**What goes wrong:** "Pacific Gas & Electric", "PG&E", and "Pacific Gas and Electric Company" treated as three different correspondents.

**Why it happens:**
- String matching on exact names
- Abbreviations not normalized
- Company name variations (Inc, LLC, Co)

**Warning signs:**
- Same company appears multiple times in correspondent list
- Tags/correspondents proliferate without bound
- User manually merging correspondents frequently

**Prevention:**
1. Implement fuzzy matching on correspondent names (Levenshtein, Jaro-Winkler)
2. Normalize company names (remove Inc, LLC, etc.)
3. Build alias table for known variations
4. Present "might be same as X" to user for confirmation
5. Allow manual merge with alias creation

**Implementation approach:**
```go
func normalizeCorrespondent(name string) string {
    // Remove common suffixes
    suffixes := []string{"Inc", "Inc.", "LLC", "Ltd", "Corp", "Co", "Company"}
    normalized := name
    for _, s := range suffixes {
        normalized = strings.TrimSuffix(normalized, " "+s)
    }
    // Remove punctuation, lowercase
    normalized = strings.ToLower(normalized)
    normalized = strings.ReplaceAll(normalized, "&", "and")
    // etc.
    return normalized
}

func findSimilarCorrespondents(db *Database, name string, threshold float64) ([]Correspondent, error) {
    normalized := normalizeCorrespondent(name)
    // Use PostgreSQL similarity() or application-level fuzzy match
}
```

**Phase:** Correspondent detection (Phase 3)

---

## Cross-Cutting Pitfalls

### Critical: No Audit Trail

**What goes wrong:** Document disappears or gets wrong tags, no way to understand what happened.

**Why it happens:**
- Only storing current state, not history
- No logging of processing steps
- Errors not recorded with context

**Prevention:**
1. Create `document_events` table with all processing steps
2. Log: ingested, duplicate_checked, indexed, ai_tagged, with timestamps and results
3. Store errors with full context for debugging
4. Admin UI to view document history

**Phase:** Queue infrastructure (Phase 1)

---

### High: Inconsistent Transaction Boundaries

**What goes wrong:** File is copied but database insert fails, leaving orphan. Or database updated but file copy fails.

**Why it happens:**
- File operations and database operations not coordinated
- No compensation logic on failure

**Prevention:**
1. Database-first: insert record, then copy file
2. On file copy failure, mark record as failed (not delete)
3. Cleanup job for failed records
4. Never delete source until database confirms success

**Phase:** Ingestion (Phase 1)

---

### Medium: Missing Health Checks

**What goes wrong:** System appears healthy but queues are stuck, shares disconnected, or database overwhelmed.

**Prevention:**
1. Health endpoint checking: database, each queue depth, each share connectivity
2. Metrics for: queue depth, processing rate, error rate
3. Alert on: queue depth > threshold, error rate spike, share disconnect

**Phase:** Infrastructure (Phase 1), monitoring (later phase)

---

## Phase Mapping Summary

| Pitfall Category | Phase to Address |
|-----------------|------------------|
| File handling (atomic writes, races) | Phase 1: Ingestion |
| Storage management | Phase 1: Infrastructure |
| Queue fundamentals (lost jobs, retries) | Phase 1: Queue Infrastructure |
| Network share integration | Phase 1: Document Sources |
| Duplicate detection (exact match) | Phase 1: Duplicate Queue |
| Full-text indexing | Phase 2: Search |
| Search relevance | Phase 2: Search UI |
| AI rate limits and costs | Phase 3: AI Integration |
| AI prompt engineering | Phase 3: AI Tagging |
| Correspondent matching | Phase 3: Correspondent Detection |
| Near-duplicate detection | v2 (defer) |
| Monitoring and alerting | Cross-cutting, add incrementally |

---

## Confidence Assessment

| Section | Confidence | Notes |
|---------|------------|-------|
| File Handling | MEDIUM | Based on general systems knowledge, not Go-specific verification |
| Queue Processing | MEDIUM | Standard patterns, PostgreSQL-specific patterns verified via training |
| Full-Text Search | MEDIUM | PostgreSQL FTS patterns from training, should verify current best practices |
| AI Integration | MEDIUM | General API patterns, specific provider behaviors may vary |
| Network Shares | MEDIUM-LOW | SMB/NFS Go library specifics need verification during implementation |
| Duplicate Detection | MEDIUM | Standard patterns, fuzzy matching approaches may need research |

**Note:** Web search was unavailable for this research. Recommendations are based on training knowledge and should be verified against current documentation during implementation phases.

---

*Research date: 2026-02-02*
*Confidence: MEDIUM (training knowledge, no current web verification)*
