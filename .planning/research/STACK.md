# Stack Research: Document Management Extensions

**Project:** Docko - PDF Document Management System
**Existing Stack:** Go 1.25 + Echo 4.15 + Templ + HTMX + Tailwind + PostgreSQL (pgx/v5)
**Research Date:** 2026-02-02
**Research Mode:** Ecosystem

---

## Important: Confidence Disclaimer

Web search and fetch tools were unavailable during this research. All recommendations are based on training data (cutoff ~May 2025). **Verify versions before adding dependencies.**

Command to check latest versions:

```bash
go list -m -versions github.com/[org]/[repo]
```

---

## PDF Processing

### Recommendation: pdfcpu + ledongthuc/pdf (dual approach)

**Confidence:** MEDIUM (based on training data, verify versions)

| Library                     | Purpose                                  | License    | Notes                             |
| --------------------------- | ---------------------------------------- | ---------- | --------------------------------- |
| `github.com/pdfcpu/pdfcpu`  | PDF manipulation, validation, page count | Apache 2.0 | Most mature Go-native PDF library |
| `github.com/ledongthuc/pdf` | Text extraction                          | MIT        | Simple, focused on extraction     |

**Why this combination:**

1. **pdfcpu** is the most actively maintained Go-native PDF library. It handles:
   - PDF validation (reject malformed files early)
   - Page count extraction
   - PDF metadata
   - Splitting, merging (useful for future features)

2. **ledongthuc/pdf** is specifically designed for text extraction and does it well. It's simpler than trying to use pdfcpu's lower-level APIs for text.

**Installation:**

```bash
go get github.com/pdfcpu/pdfcpu/pkg/api
go get github.com/ledongthuc/pdf
```

**Usage pattern:**

```go
// Validate and get page count with pdfcpu
import "github.com/pdfcpu/pdfcpu/pkg/api"

ctx, err := api.ReadContextFile(pdfPath)
if err != nil {
    return fmt.Errorf("invalid PDF: %w", err)
}
pageCount := ctx.PageCount

// Extract text with ledongthuc/pdf
import "github.com/ledongthuc/pdf"

f, r, err := pdf.Open(pdfPath)
if err != nil {
    return fmt.Errorf("failed to open PDF: %w", err)
}
defer f.Close()

var buf bytes.Buffer
for i := 1; i <= r.NumPage(); i++ {
    page := r.Page(i)
    content, _ := page.GetPlainText(nil)
    buf.WriteString(content)
}
text := buf.String()
```

### Alternatives Considered

| Library             | Why Not                                                                                                      |
| ------------------- | ------------------------------------------------------------------------------------------------------------ |
| `unidoc/unipdf`     | Commercial license required for production use. Excellent quality but expensive for personal/small team use. |
| `gen2brain/go-fitz` | Requires CGO and MuPDF C library. Adds deployment complexity.                                                |
| `signintech/gopdf`  | Focused on PDF generation, not extraction.                                                                   |

---

## Full-Text Search (PostgreSQL)

### Recommendation: tsvector + GIN index with ts_rank

**Confidence:** HIGH (PostgreSQL FTS is stable, well-documented)

PostgreSQL's built-in full-text search is sufficient for tens of thousands of documents. No need for Elasticsearch.

**Schema pattern:**

```sql
-- In documents table
CREATE TABLE documents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    original_filename TEXT NOT NULL,
    content_text TEXT,  -- Extracted text (may be large)
    content_tsv TSVECTOR GENERATED ALWAYS AS (to_tsvector('english', coalesce(content_text, ''))) STORED,
    -- ... other fields
    created_at TIMESTAMPTZ DEFAULT now()
);

-- GIN index for fast full-text search
CREATE INDEX idx_documents_content_tsv ON documents USING GIN (content_tsv);

-- Optional: Index on filename for search
CREATE INDEX idx_documents_filename_tsv ON documents
    USING GIN (to_tsvector('english', original_filename));
```

**Search query pattern:**

```sql
-- Basic search with ranking
SELECT
    id,
    original_filename,
    ts_rank(content_tsv, query) AS rank,
    ts_headline('english', content_text, query, 'MaxWords=50, MinWords=25') AS snippet
FROM documents, plainto_tsquery('english', $1) query
WHERE content_tsv @@ query
ORDER BY rank DESC
LIMIT 50;
```

**Configuration recommendations:**

1. **Use `english` configuration** for stemming and stop words
2. **GENERATED ALWAYS AS STORED** keeps tsvector in sync automatically
3. **ts_headline** provides search result snippets with highlighted matches
4. **plainto_tsquery** for user input (handles phrases naturally)
5. **websearch_to_tsquery** if you want Google-like syntax (quotes, minus for exclusion)

**sqlc query example:**

```sql
-- name: SearchDocuments :many
SELECT
    id,
    original_filename,
    ts_rank(content_tsv, plainto_tsquery('english', @query)) AS rank,
    ts_headline('english', content_text, plainto_tsquery('english', @query),
        'MaxWords=50, MinWords=25, StartSel=<mark>, StopSel=</mark>') AS snippet
FROM documents
WHERE content_tsv @@ plainto_tsquery('english', @query)
ORDER BY rank DESC
LIMIT @limit_val;
```

**Performance notes:**

- GIN index handles 100K+ documents easily
- For very large text, consider storing only first N pages for search
- `ts_headline` is expensive; consider computing only for displayed results

---

## Queue System

### Recommendation: River

**Confidence:** MEDIUM-HIGH (River gained significant traction in 2024-2025)

| Library   | Backend    | Why/Why Not                                                                     |
| --------- | ---------- | ------------------------------------------------------------------------------- |
| **River** | PostgreSQL | **Recommended.** Native PostgreSQL, no Redis needed. Transactional job enqueue. |
| Asynq     | Redis      | Mature, but requires Redis infrastructure.                                      |
| Machinery | Redis      | Older, more complex setup.                                                      |
| Go-Queue  | Various    | Less active maintenance.                                                        |

**Why River:**

1. **PostgreSQL-native**: Uses the same database you already have. No Redis required.
2. **Transactional enqueue**: Insert document + enqueue job in same transaction. If insert fails, no orphan job.
3. **Observability**: Job state is in PostgreSQL, queryable with standard SQL.
4. **Well-designed API**: Clean Go interfaces, good documentation.
5. **Active development**: Gained significant adoption in 2024-2025 Go ecosystem.

**Installation:**

```bash
go get github.com/riverqueue/river
go get github.com/riverqueue/river/riverdriver/riverpgxv5
```

**Setup pattern:**

```go
import (
    "github.com/riverqueue/river"
    "github.com/riverqueue/river/riverdriver/riverpgxv5"
)

// Define job types
type IngestArgs struct {
    DocumentID uuid.UUID `json:"document_id"`
    SourcePath string    `json:"source_path"`
}

func (IngestArgs) Kind() string { return "ingest" }

type ExtractTextArgs struct {
    DocumentID uuid.UUID `json:"document_id"`
}

func (ExtractTextArgs) Kind() string { return "extract_text" }

// Worker implementation
type IngestWorker struct {
    river.WorkerDefaults[IngestArgs]
    db *database.DB
}

func (w *IngestWorker) Work(ctx context.Context, job *river.Job[IngestArgs]) error {
    // Process ingestion
    // Enqueue next step (text extraction)
    return nil
}
```

**Migration:**
River provides migration files. Run them with your existing Goose setup:

```bash
# River publishes migration SQL files - add to internal/database/migrations/
```

**Queue per processing step** (as specified in PROJECT.md):

- `ingest` - UUID assignment, copy to originals
- `duplicate_check` - Hash comparison
- `extract_text` - PDF text extraction
- `ai_tag` - AI-powered tagging
- `detect_correspondent` - Correspondent extraction

---

## Network Shares

### SMB: go-smb2

**Confidence:** MEDIUM (most common choice, verify current version)

**Recommendation:** `github.com/hirochachacha/go-smb2`

```bash
go get github.com/hirochachacha/go-smb2
```

**Features:**

- Pure Go implementation (no C dependencies)
- SMB 2.x and 3.x support
- File operations (read, write, stat, list)
- Authentication (NTLM, guest)

**Usage pattern:**

```go
import "github.com/hirochachacha/go-smb2"

conn, err := net.Dial("tcp", "server:445")
if err != nil {
    return err
}
defer conn.Close()

d := &smb2.Dialer{
    Initiator: &smb2.NTLMInitiator{
        User:     "username",
        Password: "password",
        Domain:   "domain",
    },
}

session, err := d.Dial(conn)
if err != nil {
    return err
}
defer session.Logoff()

share, err := session.Mount("\\\\server\\share")
if err != nil {
    return err
}
defer share.Umount()

// Now use share like os filesystem
files, _ := share.ReadDir("inbox")
for _, f := range files {
    if strings.HasSuffix(f.Name(), ".pdf") {
        // Process file
    }
}
```

### NFS: Native OS Mount (Recommended)

**Confidence:** HIGH

**Recommendation:** For NFS, use OS-level mounts rather than Go libraries.

**Rationale:**

- Go NFS client libraries are less mature than SMB options
- NFS is typically mounted at OS level anyway
- Linux NFS mounts are reliable and well-supported
- You can treat mounted NFS as a local directory

**Alternative if pure-Go required:** `github.com/vmware/go-nfs-client`

- Less actively maintained
- Use only if OS mounts are truly impossible

**Configuration pattern:**

```go
type DocumentSource struct {
    ID        uuid.UUID
    Name      string
    Type      string  // "local", "smb", "nfs"
    Path      string  // Local path or SMB URI
    // SMB-specific
    SMBServer   string
    SMBShare    string
    SMBUsername string
    SMBPassword string  // Encrypted at rest
}
```

---

## Thumbnail Generation

### Recommendation: External tool (pdftoppm) via exec

**Confidence:** HIGH (pdftoppm is the standard approach)

Pure Go PDF rendering is limited. The ecosystem standard is to shell out to `pdftoppm` (from poppler-utils).

**Installation (system dependency):**

```bash
# Ubuntu/Debian
apt install poppler-utils

# macOS
brew install poppler

# Alpine (Docker)
apk add poppler-utils
```

**Usage pattern:**

```go
import (
    "os/exec"
    "path/filepath"
)

func GenerateThumbnail(pdfPath, outputDir string, documentID uuid.UUID) (string, error) {
    outputPath := filepath.Join(outputDir, documentID.String())

    // Generate PNG of first page at 200 DPI
    cmd := exec.Command("pdftoppm",
        "-png",           // Output format
        "-f", "1",        // First page only
        "-l", "1",        // Last page (same as first)
        "-r", "200",      // DPI (200 is good for thumbnails)
        "-singlefile",    // Don't add page number suffix
        pdfPath,
        outputPath,
    )

    if err := cmd.Run(); err != nil {
        return "", fmt.Errorf("pdftoppm failed: %w", err)
    }

    return outputPath + ".png", nil
}
```

**Why not pure Go?**

| Approach            | Issue                                                   |
| ------------------- | ------------------------------------------------------- |
| `gen2brain/go-fitz` | Requires CGO + MuPDF. Deployment complexity.            |
| `pdfcpu` render     | Limited rendering quality, not designed for thumbnails. |
| ImageMagick         | Heavier dependency, slower than pdftoppm.               |

**Dockerfile consideration:**

```dockerfile
FROM golang:1.25-alpine AS builder
# ... build steps

FROM alpine:3.19
RUN apk add --no-cache poppler-utils
COPY --from=builder /app/docko /usr/local/bin/
```

---

## AI Integration

### Recommendation: Direct HTTP clients (avoid heavy SDKs)

**Confidence:** MEDIUM (SDK landscape changes rapidly)

For a Go application, consider whether you need full SDKs or just HTTP clients.

**Option A: Official SDKs (if available and stable)**

| Provider        | Go SDK                                   | Notes                              |
| --------------- | ---------------------------------------- | ---------------------------------- |
| OpenAI          | `github.com/sashabaranov/go-openai`      | Community SDK, widely used         |
| Anthropic       | `github.com/anthropics/anthropic-sdk-go` | Official SDK released 2024         |
| Google (Gemini) | `cloud.google.com/go/vertexai`           | Official, part of Google Cloud SDK |

**Option B: Direct HTTP (recommended for flexibility)**

For maximum provider flexibility, implement a simple interface and use HTTP directly:

```go
type AIProvider interface {
    TagDocument(ctx context.Context, text string, pageCount int) ([]string, error)
    DetectCorrespondent(ctx context.Context, text string) (string, error)
}

type OpenAIProvider struct {
    apiKey     string
    model      string
    httpClient *http.Client
}

type AnthropicProvider struct {
    apiKey     string
    model      string
    httpClient *http.Client
}
```

**Benefits of direct HTTP:**

- No SDK version mismatches
- Easy to add new providers
- Control over retry logic
- Smaller binary size

**Provider configuration:**

```go
type AIConfig struct {
    Provider    string  // "openai", "anthropic", "ollama"
    APIKey      string
    Model       string
    MaxPages    int     // Limit pages sent to AI (cost control)
    MaxTokens   int     // Response token limit
}
```

**Prompt design for tagging:**

```go
const tagPrompt = `Analyze this document text and suggest 3-5 descriptive tags.
Return only the tags as a JSON array of strings.
Example: ["invoice", "utilities", "2024", "electricity"]

Document text (first %d pages):
%s`
```

### Local LLM Option: Ollama

**Confidence:** MEDIUM

For cost-sensitive deployments, Ollama provides local LLM inference:

```go
type OllamaProvider struct {
    baseURL string  // Default: http://localhost:11434
    model   string  // e.g., "llama3.2", "mistral"
}

func (o *OllamaProvider) TagDocument(ctx context.Context, text string, pageCount int) ([]string, error) {
    // POST to /api/generate
    // Ollama API is simple JSON over HTTP
}
```

**Ollama benefits:**

- No API costs
- Data stays local
- Good for experimentation
- Models like Llama 3.2 are surprisingly capable

---

## Anti-Recommendations

### Do NOT Use

| Library/Approach                   | Why Not                                                                                                 |
| ---------------------------------- | ------------------------------------------------------------------------------------------------------- |
| **unidoc/unipdf**                  | Commercial license. Expensive for personal projects. Use pdfcpu + ledongthuc/pdf instead.               |
| **Elasticsearch**                  | Overkill for tens of thousands of docs. PostgreSQL FTS is sufficient and avoids operational complexity. |
| **Redis for queues**               | Unnecessary infrastructure. River with PostgreSQL is simpler and transactional.                         |
| **CGO-based PDF libraries**        | Deployment complexity. Pure Go options are sufficient for this use case.                                |
| **Heavy ORM (GORM)**               | Project already uses sqlc successfully. Don't add ORM complexity.                                       |
| **go-nfs-client for all NFS**      | Less mature than SMB options. Prefer OS mounts when possible.                                           |
| **Full LLM SDKs for simple tasks** | SDK baggage. Direct HTTP is cleaner for tag/correspondent extraction.                                   |

---

## Summary: Recommended Stack Additions

```bash
# PDF Processing
go get github.com/pdfcpu/pdfcpu/pkg/api
go get github.com/ledongthuc/pdf

# Queue System (PostgreSQL-native)
go get github.com/riverqueue/river
go get github.com/riverqueue/river/riverdriver/riverpgxv5

# SMB Client
go get github.com/hirochachacha/go-smb2

# AI (optional, if using SDKs)
go get github.com/sashabaranov/go-openai       # OpenAI
go get github.com/anthropics/anthropic-sdk-go  # Anthropic
```

**System dependencies:**

```bash
# Thumbnail generation
apt install poppler-utils  # or brew install poppler
```

---

## Version Verification Commands

Run these before adding dependencies to verify current versions:

```bash
# Check latest versions
go list -m -versions github.com/pdfcpu/pdfcpu
go list -m -versions github.com/ledongthuc/pdf
go list -m -versions github.com/riverqueue/river
go list -m -versions github.com/hirochachacha/go-smb2
go list -m -versions github.com/sashabaranov/go-openai
```

---

## Confidence Assessment

| Area                 | Confidence  | Reason                                     |
| -------------------- | ----------- | ------------------------------------------ |
| PostgreSQL FTS       | HIGH        | Stable PostgreSQL feature, well-documented |
| pdfcpu               | MEDIUM      | Training data, verify version              |
| ledongthuc/pdf       | MEDIUM      | Training data, verify version              |
| River                | MEDIUM-HIGH | Strong 2024 adoption, verify version       |
| go-smb2              | MEDIUM      | Most common choice, verify version         |
| Thumbnail (pdftoppm) | HIGH        | Industry standard approach                 |
| AI SDKs              | LOW-MEDIUM  | SDK landscape changes rapidly              |

---

_Research date: 2026-02-02_
_Note: Web verification unavailable. Verify all versions before implementation._
