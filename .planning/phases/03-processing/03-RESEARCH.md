# Phase 3: Processing - Research

**Researched:** 2026-02-02
**Domain:** PDF text extraction, OCR, thumbnail generation, async processing
**Confidence:** HIGH

## Summary

This phase implements document processing: extracting text from PDFs (for search in Phase 6) and generating thumbnails (for viewing in Phase 4). Processing runs asynchronously via the queue infrastructure from Phase 1.

The research reveals a hybrid approach is optimal: use `pdfcpu` or `ledongthuc/pdf` for embedded text extraction (pure Go), OCRmyPDF Docker container for scanned page OCR, and `pdftoppm` + `cwebp` for thumbnail generation. The user decision to use Tesseract via Docker aligns well with OCRmyPDF, which bundles Tesseract with preprocessing (deskew, rotation) and produces better results than raw Tesseract.

For live status updates, HTMX's SSE extension provides the cleanest solution with Go's standard HTTP handler supporting Server-Sent Events natively. Polling is a simpler fallback if SSE proves complex.

**Primary recommendation:** Use OCRmyPDF Docker container for OCR (it handles the "has text?" detection internally with `--skip-text` mode), extract text via `--sidecar` option, and generate thumbnails with poppler-utils (`pdftoppm`) followed by `cwebp` for WebP conversion. All external tools run via `exec.Command` with proper error handling.

## Standard Stack

The established libraries/tools for this domain:

### Core

| Library/Tool | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `jbarlow83/ocrmypdf-alpine` | latest | OCR + text extraction | Bundles Tesseract with preprocessing, handles mixed content PDFs, outputs sidecar text file |
| `github.com/ledongthuc/pdf` | latest | Pure Go PDF text extraction | Extract embedded text before OCR, detect if pages need OCR |
| `poppler-utils` (pdftoppm) | system | PDF to PNG conversion | Fastest PDF renderer, used by most PDF tooling |
| `cwebp` | system | PNG to WebP conversion | Official WebP encoder from Google |

### Supporting

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `github.com/pdfcpu/pdfcpu` | latest | PDF metadata, page count | Get page count, validate PDF structure |
| `os/exec` | stdlib | External command execution | Run pdftoppm, cwebp, docker commands |
| `github.com/cenkalti/backoff/v4` | v4.x | Retry with exponential backoff | Wrap processing with configurable retry |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| OCRmyPDF Docker | Tesseract Docker directly | OCRmyPDF handles preprocessing, skip-text logic; raw Tesseract requires more custom code |
| ledongthuc/pdf | pdfcpu text extraction | pdfcpu's ExtractText is newer but less documented; ledongthuc/pdf is proven |
| pdftoppm | ImageMagick | ImageMagick requires policy changes for PDF, slower than pdftoppm |
| SSE | Polling | Polling simpler but less efficient; SSE provides instant updates |

**Installation:**
```bash
# Go dependencies
go get github.com/ledongthuc/pdf
go get github.com/pdfcpu/pdfcpu/pkg/api
go get github.com/cenkalti/backoff/v4

# Docker (add to docker-compose.yml)
# OCRmyPDF runs as ephemeral container, not a service

# System tools (for thumbnail generation in app container)
# poppler-utils provides pdftoppm
# libwebp provides cwebp
```

## Architecture Patterns

### Recommended Project Structure

```
internal/
  processing/
    processor.go      # Main processing service
    ocr.go            # OCRmyPDF Docker integration
    thumbnail.go      # Thumbnail generation
    text.go           # Text extraction (embedded + OCR)
    status.go         # Processing status management
  handler/
    sse.go            # SSE endpoint for status updates
templates/
  partials/
    document_status.templ  # Status badge component
static/
  images/
    placeholder.webp       # Placeholder for failed thumbnails
```

### Pattern 1: Processing Job Handler

**What:** Queue handler that orchestrates text extraction and thumbnail generation
**When to use:** Registered with queue for `process_document` job type

```go
// Source: Existing queue infrastructure from Phase 1
func (p *Processor) HandleJob(ctx context.Context, job *sqlc.Job) error {
    var payload document.IngestPayload
    if err := json.Unmarshal(job.Payload, &payload); err != nil {
        return fmt.Errorf("unmarshal payload: %w", err)
    }

    doc, err := p.docSvc.GetByID(ctx, payload.DocumentID)
    if err != nil {
        return fmt.Errorf("get document: %w", err)
    }

    pdfPath := p.docSvc.OriginalPath(doc)

    // Extract text (embedded + OCR if needed)
    text, err := p.extractText(ctx, pdfPath, doc.ID)
    if err != nil {
        return fmt.Errorf("extract text: %w", err)
    }

    // Generate thumbnail
    if err := p.generateThumbnail(ctx, pdfPath, doc.ID); err != nil {
        return fmt.Errorf("generate thumbnail: %w", err)
    }

    // Update document with extracted text
    if err := p.saveText(ctx, doc.ID, text); err != nil {
        return fmt.Errorf("save text: %w", err)
    }

    return nil
}
```

### Pattern 2: Hybrid Text Extraction

**What:** Extract embedded text first, OCR only pages without text
**When to use:** Every document - maximize speed while ensuring completeness

```go
// Source: ledongthuc/pdf + OCRmyPDF pattern
func (p *Processor) extractText(ctx context.Context, pdfPath string, docID uuid.UUID) (string, error) {
    // First: try pure Go extraction for embedded text
    text, hasText, err := p.extractEmbeddedText(pdfPath)
    if err != nil {
        slog.Warn("embedded text extraction failed, falling back to OCR",
            "doc_id", docID, "error", err)
    }

    if hasText && len(strings.TrimSpace(text)) > 100 {
        // Document has sufficient embedded text
        p.logEvent(ctx, docID, "text_extracted", map[string]any{
            "method": "embedded",
            "length": len(text),
        })
        return text, nil
    }

    // Need OCR - use OCRmyPDF with --skip-text to preserve existing text
    text, err = p.ocrWithSidecar(ctx, pdfPath, docID)
    if err != nil {
        return "", fmt.Errorf("ocr extraction: %w", err)
    }

    return text, nil
}

func (p *Processor) extractEmbeddedText(pdfPath string) (string, bool, error) {
    f, r, err := pdf.Open(pdfPath)
    if err != nil {
        return "", false, err
    }
    defer f.Close()

    var buf bytes.Buffer
    reader, err := r.GetPlainText()
    if err != nil {
        return "", false, err
    }

    if _, err := buf.ReadFrom(reader); err != nil {
        return "", false, err
    }

    text := buf.String()
    hasText := len(strings.TrimSpace(text)) > 0
    return text, hasText, nil
}
```

### Pattern 3: OCRmyPDF Docker Integration

**What:** Run OCRmyPDF as ephemeral Docker container with volume mounts
**When to use:** When PDF needs OCR (scanned or image-only pages)

```go
// Source: OCRmyPDF Docker documentation
func (p *Processor) ocrWithSidecar(ctx context.Context, pdfPath string, docID uuid.UUID) (string, error) {
    // Create temp directory for output
    tmpDir, err := os.MkdirTemp("", "ocr-*")
    if err != nil {
        return "", fmt.Errorf("create temp dir: %w", err)
    }
    defer os.RemoveAll(tmpDir)

    outputPDF := filepath.Join(tmpDir, "output.pdf")
    sidecarTxt := filepath.Join(tmpDir, "output.txt")

    // Run OCRmyPDF with sidecar text output
    // --skip-text: don't OCR pages that already have text
    // --sidecar: output text to separate file
    // -l eng: English language
    cmd := exec.CommandContext(ctx, "docker", "run",
        "--rm",
        "-v", fmt.Sprintf("%s:/input:ro", filepath.Dir(pdfPath)),
        "-v", fmt.Sprintf("%s:/output", tmpDir),
        "jbarlow83/ocrmypdf-alpine",
        "--skip-text",
        "--sidecar", "/output/output.txt",
        "-l", "eng",
        fmt.Sprintf("/input/%s", filepath.Base(pdfPath)),
        "/output/output.pdf",
    )

    output, err := cmd.CombinedOutput()
    if err != nil {
        return "", fmt.Errorf("ocrmypdf failed: %w\noutput: %s", err, output)
    }

    // Read sidecar text file
    text, err := os.ReadFile(sidecarTxt)
    if err != nil {
        return "", fmt.Errorf("read sidecar: %w", err)
    }

    return string(text), nil
}
```

### Pattern 4: Thumbnail Generation Pipeline

**What:** PDF first page to WebP thumbnail
**When to use:** Every document during processing

```go
// Source: pdftoppm + cwebp documentation
func (p *Processor) generateThumbnail(ctx context.Context, pdfPath string, docID uuid.UUID) error {
    thumbPath := p.storage.PathForUUID(storage.CategoryThumbnails, docID, ".webp")

    // Ensure directory exists
    if err := os.MkdirAll(filepath.Dir(thumbPath), 0755); err != nil {
        return fmt.Errorf("create thumb dir: %w", err)
    }

    tmpDir, err := os.MkdirTemp("", "thumb-*")
    if err != nil {
        return fmt.Errorf("create temp dir: %w", err)
    }
    defer os.RemoveAll(tmpDir)

    pngPath := filepath.Join(tmpDir, "thumb")

    // Step 1: PDF to PNG (first page only, 300px width)
    // -f 1 -singlefile: first page only
    // -scale-to 300: scale to 300px width
    cmd := exec.CommandContext(ctx, "pdftoppm",
        "-png",
        "-f", "1",
        "-singlefile",
        "-scale-to", "300",
        pdfPath,
        pngPath,
    )
    if output, err := cmd.CombinedOutput(); err != nil {
        // Use placeholder for corrupt/unrenderable PDFs
        return p.usePlaceholder(thumbPath)
    }

    // Step 2: PNG to WebP
    cmd = exec.CommandContext(ctx, "cwebp",
        "-q", "80",
        pngPath+".png",
        "-o", thumbPath,
    )
    if output, err := cmd.CombinedOutput(); err != nil {
        return fmt.Errorf("cwebp failed: %w\noutput: %s", err, output)
    }

    return nil
}

func (p *Processor) usePlaceholder(thumbPath string) error {
    // Copy static placeholder to thumbnail location
    return p.storage.CopyFile(p.placeholderPath, thumbPath)
}
```

### Pattern 5: SSE Status Updates

**What:** Server-Sent Events for real-time processing status
**When to use:** Document list and detail views

```go
// Source: HTMX SSE extension + Go stdlib
func (h *Handler) ProcessingStatus(c echo.Context) error {
    w := c.Response()
    w.Header().Set("Content-Type", "text/event-stream")
    w.Header().Set("Cache-Control", "no-cache")
    w.Header().Set("Connection", "keep-alive")

    ctx := c.Request().Context()

    // Subscribe to processing updates
    updates := h.processor.Subscribe(ctx)
    defer h.processor.Unsubscribe(updates)

    for {
        select {
        case <-ctx.Done():
            return nil
        case update := <-updates:
            // Send SSE event
            data, _ := json.Marshal(update)
            fmt.Fprintf(w, "event: status\ndata: %s\n\n", data)

            if f, ok := w.(http.Flusher); ok {
                f.Flush()
            }
        }
    }
}
```

```html
<!-- HTMX SSE client -->
<div hx-ext="sse" sse-connect="/api/processing/status">
    <div sse-swap="status" hx-swap="innerHTML">
        <!-- Status updates replace this content -->
    </div>
</div>
```

### Pattern 6: All-or-Nothing Transaction

**What:** Both text extraction AND thumbnail must succeed
**When to use:** Job completion - implements user's "all-or-nothing" requirement

```go
func (p *Processor) HandleJob(ctx context.Context, job *sqlc.Job) error {
    // ... extract text and generate thumbnail ...

    // Only mark complete if BOTH succeeded
    tx, err := p.db.Pool.Begin(ctx)
    if err != nil {
        return err
    }
    defer tx.Rollback(ctx)

    qtx := p.db.Queries.WithTx(tx)

    // Save extracted text
    if err := qtx.UpdateDocumentText(ctx, sqlc.UpdateDocumentTextParams{
        ID:          docID,
        TextContent: text,
    }); err != nil {
        return fmt.Errorf("save text: %w", err)
    }

    // Update processing status
    if err := qtx.SetDocumentProcessed(ctx, docID); err != nil {
        return fmt.Errorf("set processed: %w", err)
    }

    // Log success event
    if _, err := qtx.CreateDocumentEvent(ctx, sqlc.CreateDocumentEventParams{
        DocumentID: docID,
        EventType:  "processing_complete",
        Payload:    []byte(`{"text_length": ` + strconv.Itoa(len(text)) + `}`),
    }); err != nil {
        return fmt.Errorf("log event: %w", err)
    }

    return tx.Commit(ctx)
}
```

### Anti-Patterns to Avoid

- **Running OCR on every page:** Always try embedded text first; OCR is 10-100x slower
- **Synchronous thumbnail generation:** Always use queue; PDF rendering can hang
- **Ignoring OCRmyPDF exit codes:** Different exit codes mean different failures (password, corrupt, etc.)
- **Single retry strategy:** Use exponential backoff with jitter per Phase 1 research
- **Polling for status without limits:** Always set polling timeout or use SSE with heartbeat

## Don't Hand-Roll

Problems that look simple but have existing solutions:

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| OCR text extraction | Tesseract wrapper | OCRmyPDF | Handles preprocessing, deskew, rotation, skip-text logic |
| PDF page detection for OCR | Custom heuristic | OCRmyPDF --skip-text | Built-in logic to detect which pages need OCR |
| PDF to image conversion | ImageMagick | pdftoppm (poppler) | Faster, no policy restrictions, purpose-built |
| PNG to WebP | Go image library | cwebp | Optimized encoder, better compression |
| Retry with backoff | Custom loop | cenkalti/backoff | Full jitter, configurable, context-aware |
| Real-time updates | Custom WebSocket | HTMX SSE extension | Works with existing HTMX stack, simpler than WS |

**Key insight:** PDF processing has decades of edge cases (encryption, malformed files, mixed content). Use battle-tested tools rather than building from scratch. The "simple" custom solution will fail on real-world PDFs.

## Common Pitfalls

### Pitfall 1: OCRmyPDF Docker Volume Permissions

**What goes wrong:** Permission denied when writing output files
**Why it happens:** Docker runs as different user than host process
**How to avoid:** Use `--user "$(id -u):$(id -g)"` when running container, or use stdin/stdout mode
**Warning signs:** "Permission denied" errors, empty output files

```bash
# Correct: Match host user
docker run --rm --user "$(id -u):$(id -g)" \
  -v "$PWD:/data" jbarlow83/ocrmypdf-alpine ...

# Alternative: Use stdin/stdout (recommended)
cat input.pdf | docker run --rm -i jbarlow83/ocrmypdf-alpine - - > output.pdf
```

### Pitfall 2: Password-Protected PDFs

**What goes wrong:** Processing fails silently or with cryptic error
**Why it happens:** OCRmyPDF exits with specific code for encrypted PDFs
**How to avoid:** Detect encrypted PDFs before processing, quarantine with clear error message
**Warning signs:** Exit code from OCRmyPDF, "encryption must be removed" message

```go
// Check for encryption before processing
func isEncrypted(pdfPath string) (bool, error) {
    cmd := exec.Command("qpdf", "--show-encryption", pdfPath)
    output, _ := cmd.CombinedOutput()
    return strings.Contains(string(output), "File is encrypted"), nil
}
```

### Pitfall 3: Corrupt PDFs Hanging Processing

**What goes wrong:** pdftoppm or OCRmyPDF hangs indefinitely on malformed PDF
**Why it happens:** PDF parsers can enter infinite loops on crafted/corrupted files
**How to avoid:** Always use context timeout on exec.Command
**Warning signs:** Workers stuck, job timeout exceeded without progress

```go
// Always use timeout
ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
defer cancel()
cmd := exec.CommandContext(ctx, "pdftoppm", ...)
```

### Pitfall 4: Large PDFs Exhausting Memory

**What goes wrong:** OOM when processing large PDFs
**Why it happens:** Rendering full pages at high DPI, keeping all text in memory
**How to avoid:** Use streaming where possible, limit resolution, process pages individually
**Warning signs:** Container/process killed, memory spike during processing

```go
// Limit resolution for thumbnails
cmd := exec.CommandContext(ctx, "pdftoppm",
    "-r", "72",           // Low DPI for thumbnails
    "-scale-to", "300",   // Limit output size
    "-f", "1", "-singlefile",  // First page only
    ...)
```

### Pitfall 5: SSE Connection Accumulation

**What goes wrong:** Too many open SSE connections exhaust server resources
**Why it happens:** Clients reconnect on page refresh, old connections not cleaned up
**How to avoid:** Use proper context cancellation, implement connection limits
**Warning signs:** Increasing memory usage, file descriptor exhaustion

```go
// Always respect context cancellation
select {
case <-ctx.Done():
    return nil  // Client disconnected, clean up
case update := <-updates:
    // Send update
}
```

## Code Examples

Verified patterns from official sources:

### ledongthuc/pdf Text Extraction

```go
// Source: pkg.go.dev/github.com/ledongthuc/pdf
import "github.com/ledongthuc/pdf"

func extractText(pdfPath string) (string, error) {
    f, r, err := pdf.Open(pdfPath)
    if err != nil {
        return "", fmt.Errorf("open pdf: %w", err)
    }
    defer f.Close()

    var buf bytes.Buffer
    reader, err := r.GetPlainText()
    if err != nil {
        return "", fmt.Errorf("get text: %w", err)
    }

    if _, err := buf.ReadFrom(reader); err != nil {
        return "", fmt.Errorf("read text: %w", err)
    }

    return buf.String(), nil
}
```

### OCRmyPDF via Docker with Sidecar

```bash
# Source: ocrmypdf.readthedocs.io/en/latest/docker.html
docker run --rm -i jbarlow83/ocrmypdf-alpine \
    --skip-text \
    --sidecar /dev/stderr \
    -l eng \
    - - < input.pdf > output.pdf 2> text.txt
```

### pdftoppm First Page Thumbnail

```bash
# Source: poppler-utils documentation
pdftoppm -png -f 1 -singlefile -scale-to 300 input.pdf output
# Creates output.png (first page, 300px width)
```

### cwebp PNG to WebP

```bash
# Source: developers.google.com/speed/webp/docs/cwebp
cwebp -q 80 input.png -o output.webp
```

### HTMX SSE Connection

```html
<!-- Source: htmx.org/extensions/sse/ -->
<div hx-ext="sse" sse-connect="/api/status" sse-swap="update">
    <!-- Content swapped when "update" event received -->
</div>

<!-- Close connection on specific event -->
<div hx-ext="sse" sse-connect="/api/status" sse-close="complete">
    ...
</div>
```

### Go SSE Handler with Echo

```go
// Source: threedots.tech/post/live-website-updates-go-sse-htmx/
func SSEHandler(c echo.Context) error {
    w := c.Response()
    w.Header().Set("Content-Type", "text/event-stream")
    w.Header().Set("Cache-Control", "no-cache")

    flusher, ok := w.(http.Flusher)
    if !ok {
        return echo.NewHTTPError(http.StatusInternalServerError, "SSE not supported")
    }

    ctx := c.Request().Context()
    ticker := time.NewTicker(5 * time.Second)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            return nil
        case <-ticker.C:
            fmt.Fprintf(w, "event: heartbeat\ndata: ping\n\n")
            flusher.Flush()
        }
    }
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Tesseract directly | OCRmyPDF wrapper | Established | Better preprocessing, skip-text logic, sidecar output |
| ImageMagick PDF | pdftoppm (poppler) | Established | Faster, no policy issues, better rendering |
| PNG thumbnails | WebP thumbnails | 2020+ | 25-30% smaller files, better compression |
| Polling for updates | Server-Sent Events | Established | Lower server load, instant updates |
| Full page OCR | Skip pages with text | OCRmyPDF default | 10-100x faster for mixed documents |

**Deprecated/outdated:**
- `ghostscript` for PDF rendering: Use poppler (pdftoppm) instead - faster, fewer security issues
- Direct Tesseract calls: OCRmyPDF handles preprocessing that significantly improves accuracy
- PNG for web thumbnails: WebP provides better compression with same quality

## Open Questions

Things that couldn't be fully resolved:

1. **OCRmyPDF exit codes**
   - What we know: Exit code -1 for "already has text", other codes for encryption/corruption
   - What's unclear: Complete list of exit codes and their meanings
   - Recommendation: Capture stderr for error messages, treat non-zero as failure with retry

2. **Optimal OCR quality vs speed tradeoff**
   - What we know: OCRmyPDF has "fast" mode vs default quality
   - What's unclear: Whether fast mode is sufficient for search indexing
   - Recommendation: Use default quality initially, benchmark if too slow

3. **SSE vs polling for this use case**
   - What we know: SSE is more efficient, polling is simpler
   - What's unclear: Connection management complexity at scale
   - Recommendation: Start with SSE, fall back to polling if issues arise

4. **Quarantine mechanism details**
   - What we know: Failed documents should be quarantined
   - What's unclear: Storage location, cleanup policy, recovery workflow
   - Recommendation: Create `quarantine` directory under storage, log errors to document_events

## Sources

### Primary (HIGH confidence)
- [OCRmyPDF Docker documentation](https://ocrmypdf.readthedocs.io/en/latest/docker.html) - Container usage, volume mounts
- [OCRmyPDF Advanced features](https://ocrmypdf.readthedocs.io/en/latest/advanced.html) - Skip-text, sidecar options
- [ledongthuc/pdf pkg.go.dev](https://pkg.go.dev/github.com/ledongthuc/pdf) - Text extraction API
- [HTMX SSE Extension](https://htmx.org/extensions/sse/) - SSE client-side integration
- [Three Dots Labs SSE + Go](https://threedots.tech/post/live-website-updates-go-sse-htmx/) - Go server implementation
- [Google cwebp documentation](https://developers.google.com/speed/webp/docs/cwebp) - WebP encoding options

### Secondary (MEDIUM confidence)
- [pdfcpu GitHub](https://github.com/pdfcpu/pdfcpu) - PDF metadata, page count
- [pdftoppm documentation](https://linuxcommandlibrary.com/man/pdftoppm) - PDF to image options
- [cenkalti/backoff](https://github.com/cenkalti/backoff) - Retry library patterns

### Tertiary (LOW confidence)
- Blog posts on testing exec.Command in Go - patterns for mocking
- Community discussions on OCRmyPDF exit codes - incomplete information

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - Official documentation for all recommended tools
- Architecture: HIGH - Based on existing queue infrastructure and proven patterns
- Pitfalls: HIGH - Verified against OCRmyPDF documentation and community issues
- Code examples: HIGH - Verified against official documentation

**Research date:** 2026-02-02
**Valid until:** 2026-03-02 (30 days - mature tools, stable APIs)
