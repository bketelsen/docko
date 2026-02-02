# Architecture Research

**Domain:** PDF Document Management System
**Researched:** 2026-02-02
**Confidence:** HIGH (well-established patterns)

## Component Overview

A document management system for this scale (tens of thousands of documents) requires six major subsystems that interact through well-defined boundaries.

### 1. Document Store (File System Layer)

**Purpose:** Physical storage of document files with predictable, conflict-free naming.

**Structure:**
```
data/
  inbox/              # Drop zone for new documents
  originals/          # Unmodified source PDFs (UUID-named)
  documents/          # Processed PDFs (optional normalization)
  thumbnails/         # Generated preview images (UUID-named)
  temp/               # Working directory for processing
```

**Rationale:**
- UUID-based naming eliminates filename collisions
- Separating originals from processed allows rollback
- Inbox isolation prevents processing of partially-written files
- Thumbnails separate from documents for cache management

**Key Design Decisions:**
- **Flat vs nested:** Flat directories (all files at same level) until >100K files. PostgreSQL handles the indexing.
- **Original preservation:** Always keep the original untouched. Processing creates derived artifacts.
- **Configurable root:** Single `DATA_DIR` env var points to the storage root.

### 2. Metadata Database (PostgreSQL)

**Purpose:** Track document metadata, relationships, processing state, and enable search.

**Core Entities:**

```
documents
  id: UUID (PK)
  original_filename: text
  file_hash: text (SHA-256 for dedup)
  file_size: bigint
  page_count: int
  mime_type: text
  source_id: UUID -> sources
  correspondent_id: UUID -> correspondents (nullable)
  status: enum (pending, processing, ready, error)
  created_at: timestamptz
  updated_at: timestamptz
  archived_at: timestamptz (nullable, soft delete)

document_content
  document_id: UUID (PK, FK)
  full_text: text
  search_vector: tsvector (generated)

tags
  id: UUID (PK)
  name: text (unique)
  color: text
  created_at: timestamptz

document_tags
  document_id: UUID (FK)
  tag_id: UUID (FK)
  confidence: float (nullable, for AI-assigned)
  source: enum (manual, ai)
  PRIMARY KEY (document_id, tag_id)

correspondents
  id: UUID (PK)
  name: text (unique)
  created_at: timestamptz

sources
  id: UUID (PK)
  name: text
  type: enum (local, smb, nfs)
  path: text
  duplicate_action: enum (delete, rename, skip)
  enabled: boolean
  last_scan_at: timestamptz
  created_at: timestamptz
```

**Processing State Entities:**

```
queue_jobs
  id: UUID (PK)
  document_id: UUID (FK)
  queue_name: text
  status: enum (pending, running, completed, failed, dead)
  priority: int
  attempts: int
  max_attempts: int
  payload: jsonb
  error_message: text
  scheduled_at: timestamptz
  started_at: timestamptz
  completed_at: timestamptz
  created_at: timestamptz

document_audit_log
  id: UUID (PK)
  document_id: UUID (FK)
  action: text
  details: jsonb
  created_at: timestamptz
```

**Rationale:**
- Separate `document_content` table keeps main documents table lean for listing queries
- `tsvector` generated column auto-updates for full-text search
- `queue_jobs` in same database simplifies transactions (no Redis needed at this scale)
- Audit log provides complete processing history per document

### 3. Processing Pipeline (Queue-Based Workers)

**Purpose:** Asynchronous document processing through defined stages.

**Queue Architecture:**

```
                    +------------------+
                    |   Source Watcher |
                    +--------+---------+
                             |
                             v
                    +--------+---------+
                    |  Ingestion Queue |
                    +--------+---------+
                             |
              +--------------+--------------+
              |                             |
              v                             v
    +---------+---------+         +---------+---------+
    | Duplicate Check Q |         |  Thumbnail Queue  |
    +---------+---------+         +---------+---------+
              |                             |
              v                             |
    +---------+---------+                   |
    |  Text Extract Q   |                   |
    +---------+---------+                   |
              |                             |
              v                             |
    +---------+---------+                   |
    |    AI Tagging Q   |                   |
    +---------+---------+                   |
              |                             |
              v                             v
    +---------+---------------------------+-+
    |              Document Ready            |
    +----------------------------------------+
```

**Queue Definitions:**

| Queue | Input | Output | Failure Mode |
|-------|-------|--------|--------------|
| `ingestion` | File path | Document record + UUID copy | Retry 3x, then dead |
| `duplicate_check` | Document ID | Hash comparison | Skip if duplicate |
| `text_extract` | Document ID | Full text in DB | Retry 3x, mark error |
| `thumbnail` | Document ID | Thumbnail file | Non-blocking, retry |
| `ai_tagging` | Document ID | Tag assignments | Optional, retry 2x |
| `correspondent` | Document ID | Correspondent link | Optional, best-effort |

**Worker Model:**

```go
type QueueWorker struct {
    db          *database.DB
    store       *store.DocumentStore
    queueName   string
    concurrency int
    handler     func(ctx context.Context, job *Job) error
}
```

**Rationale:**
- PostgreSQL-backed queue (no Redis) simplifies deployment
- Each queue has independent concurrency (thumbnail generation can run parallel to text extraction)
- Failure isolation: one queue's backlog doesn't block others
- Audit trail built-in via `document_audit_log`

### 4. Search Subsystem (PostgreSQL Full-Text)

**Purpose:** Fast document retrieval by content, metadata, and facets.

**Indexing Strategy:**

```sql
-- Generated tsvector column (auto-updates)
ALTER TABLE document_content ADD COLUMN search_vector tsvector
  GENERATED ALWAYS AS (
    setweight(to_tsvector('english', coalesce(full_text, '')), 'A')
  ) STORED;

-- GIN index for fast full-text search
CREATE INDEX idx_document_content_search ON document_content USING GIN (search_vector);

-- Partial index for active documents only
CREATE INDEX idx_documents_active ON documents (created_at DESC)
  WHERE archived_at IS NULL;

-- Index for tag filtering
CREATE INDEX idx_document_tags_tag_id ON document_tags (tag_id);

-- Index for correspondent filtering
CREATE INDEX idx_documents_correspondent ON documents (correspondent_id)
  WHERE correspondent_id IS NOT NULL;
```

**Query Pattern:**

```sql
SELECT d.id, d.original_filename, ts_rank(dc.search_vector, query) as rank
FROM documents d
JOIN document_content dc ON d.id = dc.document_id
CROSS JOIN websearch_to_tsquery('english', $1) query
WHERE dc.search_vector @@ query
  AND d.archived_at IS NULL
  AND ($2::uuid IS NULL OR d.correspondent_id = $2)
  AND ($3::uuid[] IS NULL OR EXISTS (
    SELECT 1 FROM document_tags dt
    WHERE dt.document_id = d.id AND dt.tag_id = ANY($3)
  ))
ORDER BY rank DESC
LIMIT 50;
```

**Rationale:**
- `websearch_to_tsquery` handles user-friendly queries (AND/OR, quotes)
- GIN index provides O(1) lookup for term matching
- Facet filtering via subqueries keeps query planner happy
- No Elasticsearch needed at tens of thousands of documents

### 5. Source Management (Watchers)

**Purpose:** Monitor local directories and network shares for new documents.

**Architecture:**

```
+-------------------+     +-------------------+     +-------------------+
|   Local Watcher   |     |   SMB Watcher     |     |   NFS Watcher     |
|   (fsnotify)      |     |   (go-smb2)       |     |   (NFS client)    |
+--------+----------+     +--------+----------+     +--------+----------+
         |                         |                         |
         v                         v                         v
+--------+--------------------------+-------------------------+
|                        Ingestion Service                      |
|  - Validates file type                                       |
|  - Checks if inbox file is complete (size stable)            |
|  - Queues for ingestion                                      |
+--------------------------------------------------------------+
```

**Watcher Behaviors:**

| Type | Library | Polling | Event-Based |
|------|---------|---------|-------------|
| Local | `fsnotify` | No | Yes (inotify/kqueue) |
| SMB | `go-smb2` | Yes (configurable) | No |
| NFS | Go NFS client or mount | Configurable | Depends |

**File Stability Check:**
```go
// Wait for file to stop changing (upload complete)
func waitForStable(path string, timeout time.Duration) error {
    var lastSize int64 = -1
    deadline := time.Now().Add(timeout)

    for time.Now().Before(deadline) {
        info, err := os.Stat(path)
        if err != nil { return err }

        if info.Size() == lastSize {
            return nil // Stable
        }
        lastSize = info.Size()
        time.Sleep(1 * time.Second)
    }
    return errors.New("file not stable within timeout")
}
```

**Rationale:**
- Local directories use event-based watching (efficient)
- Network shares poll (SMB/NFS don't support cross-network events)
- Stability check prevents processing partial uploads
- Per-source configuration for duplicate handling

### 6. AI Integration Layer

**Purpose:** Flexible AI provider integration for document tagging.

**Architecture:**

```go
type AIProvider interface {
    // TagDocument returns suggested tags with confidence scores
    TagDocument(ctx context.Context, text string, existingTags []string) ([]TagSuggestion, error)

    // DetectCorrespondent attempts to identify document sender/source
    DetectCorrespondent(ctx context.Context, text string, existingCorrespondents []string) (*string, float64, error)
}

type TagSuggestion struct {
    Tag        string
    Confidence float64
}
```

**Provider Implementations:**

```
internal/ai/
  provider.go         # Interface definition
  openai.go           # OpenAI implementation
  claude.go           # Claude implementation
  ollama.go           # Local Ollama implementation
  mock.go             # Mock for testing
```

**Configuration:**

```go
type AIConfig struct {
    Provider      string  // "openai", "claude", "ollama", "none"
    APIKey        string  // For cloud providers
    Model         string  // e.g., "gpt-4o-mini", "claude-3-haiku"
    MaxPages      int     // Limit text sent to AI (cost control)
    Endpoint      string  // For self-hosted (Ollama)
    TagPrompt     string  // Custom prompt template
}
```

**Rationale:**
- Interface allows swapping providers without code changes
- MaxPages limits API costs (first N pages usually contain key info)
- Custom prompt template allows domain-specific tuning
- "none" provider disables AI features entirely

## Data Flow

### Document Ingestion Flow

```
1. Source detects new file
     |
2. Stability check (wait for upload complete)
     |
3. Create document record (status: pending)
     |
4. Copy file to originals/ with UUID name
     |
5. Compute file hash (SHA-256)
     |
6. Queue: duplicate_check
     |
7. IF duplicate:
     |   - Link to existing document OR
     |   - Apply source's duplicate_action
     |
8. ELSE: Queue parallel jobs:
     |   - text_extract
     |   - thumbnail
     |
9. After text_extract complete:
     |   - Queue: ai_tagging (if enabled)
     |   - Queue: correspondent_detect (if enabled)
     |
10. All queues complete:
      - Set status: ready
      - Log audit entry
```

### Search Flow

```
1. User enters search query
     |
2. Parse query (websearch_to_tsquery)
     |
3. Apply filters (tags, correspondent, date)
     |
4. Execute PostgreSQL query with:
     |   - Full-text matching
     |   - Facet filtering
     |   - Ranking
     |
5. Return paginated results with:
     |   - Document metadata
     |   - Snippet with highlights
     |   - Thumbnail URL
```

### Tag Assignment Flow

```
Manual:
1. User selects tag for document
2. Insert document_tag with source=manual
3. Log audit entry

AI-Suggested:
1. AI returns tag suggestions with confidence
2. Insert document_tags with source=ai, confidence score
3. User can confirm/reject (updates source to manual)
```

## Integration Points

### External Systems

| System | Integration | Library/Method |
|--------|-------------|----------------|
| SMB shares | Document ingestion | `github.com/hirochachacha/go-smb2` |
| NFS shares | Document ingestion | Go NFS client or OS mount |
| OpenAI API | AI tagging | `github.com/sashabaranov/go-openai` |
| Anthropic API | AI tagging | HTTP client |
| Ollama | Local AI | HTTP client to local endpoint |

### Internal Service Boundaries

```
cmd/server/main.go
  |
  +-- internal/config/        # Configuration
  |
  +-- internal/database/      # Database layer
  |
  +-- internal/auth/          # Authentication (existing)
  |
  +-- internal/store/         # Document file storage    [NEW]
  |
  +-- internal/queue/         # Job queue system         [NEW]
  |
  +-- internal/source/        # Source watchers          [NEW]
  |     +-- local.go
  |     +-- smb.go
  |     +-- nfs.go
  |
  +-- internal/processor/     # Processing workers       [NEW]
  |     +-- ingest.go
  |     +-- duplicate.go
  |     +-- text.go
  |     +-- thumbnail.go
  |
  +-- internal/ai/            # AI provider interface    [NEW]
  |     +-- provider.go
  |     +-- openai.go
  |
  +-- internal/search/        # Search service           [NEW]
  |
  +-- internal/handler/       # HTTP handlers (existing)
```

## Build Order

Based on dependencies, recommended implementation order:

### Phase 1: Foundation (No Dependencies)

1. **Document Store (`internal/store/`)**
   - Directory structure management
   - UUID file naming
   - File operations (copy, move, delete)
   - No database dependency

2. **Database Schema**
   - Core tables: documents, tags, correspondents
   - Queue table: queue_jobs
   - Audit table: document_audit_log
   - Migrations in `internal/database/migrations/`

### Phase 2: Queue Infrastructure (Depends on Phase 1)

3. **Queue System (`internal/queue/`)**
   - Job creation, claiming, completion
   - Retry logic with exponential backoff
   - Concurrency control
   - Depends on: Database schema

### Phase 3: Core Processing (Depends on Phase 2)

4. **Ingestion Pipeline (`internal/processor/`)**
   - Ingest handler: file copy, record creation
   - Duplicate detection: hash comparison
   - Depends on: Store, Queue, Database

5. **Text Extraction**
   - PDF text extraction (pdftotext or pure Go)
   - Full-text indexing in PostgreSQL
   - Depends on: Queue, Database

6. **Thumbnail Generation**
   - PDF to image conversion
   - Resize and optimize
   - Depends on: Store, Queue

### Phase 4: Source Management (Depends on Phase 3)

7. **Local Source Watcher (`internal/source/`)**
   - fsnotify-based directory watching
   - File stability checking
   - Depends on: Ingestion pipeline

8. **Network Sources**
   - SMB integration
   - NFS integration
   - Depends on: Source watcher pattern

### Phase 5: Search & UI (Depends on Phases 1-3)

9. **Search Service (`internal/search/`)**
   - Query parsing
   - Faceted filtering
   - Pagination
   - Depends on: Database schema with indexes

10. **Document Management UI**
    - Document list with search
    - Document detail view
    - Tag/correspondent management
    - Depends on: Search service, existing Templ/HTMX patterns

### Phase 6: AI Integration (Optional, Depends on Phase 3)

11. **AI Provider Interface (`internal/ai/`)**
    - Provider interface definition
    - OpenAI/Claude implementation
    - Depends on: Text extraction (needs content to analyze)

12. **AI Processing Workers**
    - Tag suggestion worker
    - Correspondent detection worker
    - Depends on: AI provider, Queue

## Anti-Patterns to Avoid

### 1. Processing in HTTP Request

**Wrong:**
```go
func (h *Handler) UploadDocument(c echo.Context) error {
    file := c.FormFile("document")
    text := extractText(file)           // Blocks request
    tags := aiService.SuggestTags(text) // Blocks request
    // ...
}
```

**Right:** Queue for async processing, return immediately.

### 2. Single Monolithic Queue

**Wrong:** One queue for all processing steps.

**Right:** Separate queues allow independent scaling and failure isolation.

### 3. Storing Files by Original Name

**Wrong:** `/documents/Invoice-2024.pdf` (collisions, special characters)

**Right:** `/originals/{uuid}.pdf` with original name in database.

### 4. Missing Audit Trail

**Wrong:** Direct database updates without logging.

**Right:** Every state change logged with timestamp and details.

### 5. Tight AI Provider Coupling

**Wrong:** OpenAI-specific code throughout codebase.

**Right:** Interface with swappable implementations.

## Scalability Considerations

| Scale | Documents | Approach |
|-------|-----------|----------|
| Small | <10K | Single process, sync processing acceptable |
| Medium | 10K-100K | Queue-based async, single server |
| Large | 100K-1M | Multiple workers, consider file sharding |
| Enterprise | >1M | Beyond scope; needs distributed architecture |

For the target scale (tens of thousands), the single-server PostgreSQL-backed queue architecture is appropriate. No need for Redis, Elasticsearch, or distributed workers.

---

*Research date: 2026-02-02*
*Confidence: HIGH - Document management is a well-established domain with proven patterns*
