# Phase 2: Ingestion - Research

**Researched:** 2026-02-02
**Domain:** File upload (web + inbox watching), duplicate detection, PDF validation
**Confidence:** HIGH

## Summary

This phase implements document ingestion through two channels: web upload with drag-and-drop and automated inbox directory watching. The research focused on five key technical domains: (1) file upload handling in Echo with HTMX progress tracking, (2) filesystem watching with fsnotify, (3) PDF validation using magic bytes, (4) full-page drag-and-drop UI patterns, and (5) toast notifications for user feedback.

The existing codebase already has the document service (`internal/document/document.go`) with SHA-256 hashing and duplicate detection, and the storage service (`internal/storage/storage.go`) with file operations. This phase builds on that foundation to add the ingestion entry points (web upload handler and inbox watcher service).

**Primary recommendation:** Use fsnotify for inbox watching, Echo's multipart form handling with XMLHttpRequest progress events for uploads, and h2non/filetype for PDF validation. Toast notifications via templUI with HTMX out-of-band swaps for user feedback.

## Standard Stack

The established libraries/tools for this domain:

### Core

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| github.com/fsnotify/fsnotify | latest | OS-level file watching for inbox | De-facto standard for Go file watching, cross-platform (inotify/kqueue/ReadDirectoryChangesW) |
| github.com/h2non/filetype | latest | PDF validation via magic bytes | Dependency-free, only needs first 262 bytes, fast detection |
| github.com/labstack/echo/v4 | v4.15.0 | HTTP handler with multipart form support | Already in use, excellent file upload support |

### Supporting

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| htmx.org | 2.0.4 | Progress events, OOB swaps for toasts | Already in use for HTMX interactions |
| templUI toast | latest | Toast notification component | Install via `templui add toast` |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| fsnotify | polling with os.Stat | Polling wastes CPU, slower detection; fsnotify is near-instant |
| h2non/filetype | net/http.DetectContentType | Standard lib only reads 512 bytes and less accurate for PDFs |
| h2non/filetype | pdfcpu validation | pdfcpu is heavier; magic bytes sufficient for basic PDF detection |

**Installation:**
```bash
go get github.com/fsnotify/fsnotify
go get github.com/h2non/filetype
templui add toast
```

## Architecture Patterns

### Recommended Project Structure

```
internal/
  inbox/
    watcher.go       # Inbox directory watcher service
    config.go        # Inbox configuration management
  handler/
    upload.go        # Web upload handlers (POST /upload, drag-drop)
  document/
    document.go      # Existing: Ingest() method already handles storage + hash + duplicate
templates/
  pages/admin/
    upload.templ     # Upload page with drop zone
  partials/
    upload_result.templ    # Individual file upload result
    toast.templ            # Toast notification wrapper
static/
  js/
    upload.js        # Drag-and-drop + progress bar JavaScript
```

### Pattern 1: Inbox Watcher Service

**What:** Long-running goroutine that watches inbox directories and triggers document ingestion
**When to use:** Startup - runs continuously until shutdown

```go
// Source: fsnotify documentation
type InboxWatcher struct {
    watcher   *fsnotify.Watcher
    docSvc    *document.Service
    inboxes   map[string]*InboxConfig  // path -> config
    mu        sync.RWMutex
}

func (w *InboxWatcher) Run(ctx context.Context) error {
    for {
        select {
        case event, ok := <-w.watcher.Events:
            if !ok {
                return nil
            }
            if event.Has(fsnotify.Create) || event.Has(fsnotify.Write) {
                w.handleFile(ctx, event.Name)
            }
        case err, ok := <-w.watcher.Errors:
            if !ok {
                return nil
            }
            slog.Error("watcher error", "error", err)
        case <-ctx.Done():
            return ctx.Err()
        }
    }
}
```

### Pattern 2: File Upload Handler with Echo

**What:** Handle multipart form uploads with individual file processing
**When to use:** POST /upload endpoint

```go
// Source: Echo file upload cookbook
func (h *Handler) Upload(c echo.Context) error {
    form, err := c.MultipartForm()
    if err != nil {
        return err
    }
    files := form.File["files"]

    var results []UploadResult
    for _, file := range files {
        result := h.processUpload(c.Request().Context(), file)
        results = append(results, result)
    }

    return templates.UploadResults(results).Render(c.Request().Context(), c.Response())
}
```

### Pattern 3: Full-Page Drop Zone with Overlay

**What:** Document-level drag events show overlay, drop triggers upload
**When to use:** Any page where drag-and-drop upload is needed

```javascript
// Source: MDN Drag and Drop API
document.addEventListener('dragenter', (e) => {
    if ([...e.dataTransfer.items].some(item => item.kind === 'file')) {
        e.preventDefault();
        document.getElementById('drop-overlay').classList.remove('hidden');
    }
});

document.addEventListener('dragover', (e) => {
    if ([...e.dataTransfer.items].some(item => item.kind === 'file')) {
        e.preventDefault();
        e.dataTransfer.dropEffect = 'copy';
    }
});

document.addEventListener('drop', (e) => {
    e.preventDefault();
    document.getElementById('drop-overlay').classList.add('hidden');
    const files = [...e.dataTransfer.files].filter(f => f.type === 'application/pdf');
    if (files.length > 0) {
        uploadFiles(files);
    }
});
```

### Pattern 4: Parallel Upload with Individual Progress

**What:** Upload multiple files simultaneously with per-file progress tracking
**When to use:** User drags multiple files

```javascript
// Source: XMLHttpRequest progress events
function uploadFile(file, index) {
    return new Promise((resolve, reject) => {
        const xhr = new XMLHttpRequest();
        const formData = new FormData();
        formData.append('file', file);

        xhr.upload.onprogress = (e) => {
            if (e.lengthComputable) {
                const pct = Math.round((e.loaded / e.total) * 100);
                updateProgress(index, pct);
            }
        };

        xhr.onload = () => {
            if (xhr.status >= 200 && xhr.status < 300) {
                resolve(JSON.parse(xhr.responseText));
            } else {
                reject(new Error(xhr.statusText));
            }
        };

        xhr.open('POST', '/api/upload');
        xhr.send(formData);
    });
}

// Upload all files in parallel
async function uploadFiles(files) {
    const promises = files.map((file, i) => uploadFile(file, i));
    const results = await Promise.all(promises);
    // Show toast notification
}
```

### Pattern 5: Toast Notification with OOB Swap

**What:** Show toast after upload completion using HTMX out-of-band swap
**When to use:** After successful/failed upload

```html
<!-- Server returns this alongside normal response -->
<div id="toast-container" hx-swap-oob="beforeend">
    <div class="toast toast-success"
         hx-get="/_empty"
         hx-trigger="load delay:4s"
         hx-swap="outerHTML">
        3 documents uploaded successfully
    </div>
</div>
```

### Anti-Patterns to Avoid

- **Loading entire file into memory:** Use io.Copy with streaming, not ioutil.ReadAll
- **Watching individual files:** Watch directory, filter by filename - editors use atomic writes
- **Blocking on file copy:** The existing document.Ingest() already handles this well
- **Relying on file extension for PDF check:** Always validate magic bytes

## Don't Hand-Roll

Problems that look simple but have existing solutions:

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| File type detection | Check extension or content-type header | h2non/filetype magic bytes | Extensions can be wrong, headers can be spoofed |
| File system watching | Polling loop with os.Stat | fsnotify | Polling wastes CPU, misses fast changes |
| SHA-256 hashing | Custom implementation | crypto/sha256 (already in storage.go) | Standard lib is correct and fast |
| Progress tracking | Custom AJAX wrapper | XMLHttpRequest with upload.onprogress | Fetch API doesn't support upload progress |
| Toast notifications | Custom JS notification | templUI toast component | Already styled, handles dismiss, HTMX-ready |

**Key insight:** The existing `document.Service.Ingest()` already handles the hard parts (copy+hash in single pass, duplicate detection, transaction with queue). New code just needs to provide the entry points.

## Common Pitfalls

### Pitfall 1: fsnotify Write Event Fires Multiple Times

**What goes wrong:** A single file write triggers multiple Write events as the file is written in chunks
**Why it happens:** OS reports each write() syscall as an event; large files have many writes
**How to avoid:** Debounce events per file - wait 100-500ms after last event before processing
**Warning signs:** Same file processed multiple times, duplicate detection fires repeatedly

```go
// Debounce pattern
type debouncer struct {
    timers map[string]*time.Timer
    mu     sync.Mutex
}

func (d *debouncer) Debounce(path string, delay time.Duration, fn func()) {
    d.mu.Lock()
    defer d.mu.Unlock()

    if timer, ok := d.timers[path]; ok {
        timer.Stop()
    }
    d.timers[path] = time.AfterFunc(delay, fn)
}
```

### Pitfall 2: File Not Ready When Event Fires

**What goes wrong:** File still being written when Create event fires, resulting in partial reads or "file in use" errors
**Why it happens:** Create event fires when file is created, not when write completes
**How to avoid:** Wait for file to stabilize (no size change for N ms) or handle Write events after Create
**Warning signs:** Truncated files, "access denied" errors on Windows

### Pitfall 3: Drag Events Fire on Child Elements

**What goes wrong:** dragleave fires when dragging over child elements, hiding overlay prematurely
**Why it happens:** Drag events bubble, child elements trigger enter/leave on parent
**How to avoid:** Use a counter - increment on dragenter, decrement on dragleave, hide when counter hits 0
**Warning signs:** Overlay flickers during drag

```javascript
let dragCounter = 0;
document.addEventListener('dragenter', (e) => {
    e.preventDefault();
    dragCounter++;
    overlay.classList.remove('hidden');
});
document.addEventListener('dragleave', (e) => {
    e.preventDefault();
    dragCounter--;
    if (dragCounter === 0) {
        overlay.classList.add('hidden');
    }
});
```

### Pitfall 4: Fetch API Cannot Track Upload Progress

**What goes wrong:** Using fetch() for upload means no progress bar
**Why it happens:** Fetch API ReadableStream is for download, not upload; upload progress not in spec
**How to avoid:** Use XMLHttpRequest with xhr.upload.onprogress
**Warning signs:** Progress bar stuck at 0 or jumps directly to 100

### Pitfall 5: Inbox Watcher Doesn't Process Existing Files

**What goes wrong:** Files added while service was down are ignored
**Why it happens:** fsnotify only watches for new events, doesn't scan existing files
**How to avoid:** On startup, scan each inbox directory and process all .pdf files before starting watcher
**Warning signs:** Documents present in inbox but not imported after restart

## Code Examples

Verified patterns from official sources:

### PDF Validation with Magic Bytes

```go
// Source: h2non/filetype documentation
import "github.com/h2non/filetype"

func IsPDF(path string) (bool, error) {
    // Only need first 262 bytes for detection
    buf := make([]byte, 262)
    f, err := os.Open(path)
    if err != nil {
        return false, err
    }
    defer f.Close()

    n, err := f.Read(buf)
    if err != nil && err != io.EOF {
        return false, err
    }

    return filetype.Is(buf[:n], "pdf"), nil
}
```

### fsnotify Watcher Setup

```go
// Source: fsnotify pkg.go.dev documentation
func NewInboxWatcher(paths []string) (*InboxWatcher, error) {
    watcher, err := fsnotify.NewWatcher()
    if err != nil {
        return nil, fmt.Errorf("create watcher: %w", err)
    }

    for _, path := range paths {
        if err := watcher.Add(path); err != nil {
            watcher.Close()
            return nil, fmt.Errorf("watch %s: %w", path, err)
        }
    }

    return &InboxWatcher{watcher: watcher}, nil
}
```

### Echo Multipart File Upload

```go
// Source: Echo file upload cookbook
func (h *Handler) UploadSingle(c echo.Context) error {
    file, err := c.FormFile("file")
    if err != nil {
        return echo.NewHTTPError(http.StatusBadRequest, "no file provided")
    }

    src, err := file.Open()
    if err != nil {
        return err
    }
    defer src.Close()

    // Save to temp file, then pass to document service
    tmpFile, err := os.CreateTemp("", "upload-*.pdf")
    if err != nil {
        return err
    }
    defer os.Remove(tmpFile.Name())
    defer tmpFile.Close()

    if _, err := io.Copy(tmpFile, src); err != nil {
        return err
    }

    doc, isDupe, err := h.docSvc.Ingest(c.Request().Context(), tmpFile.Name(), file.Filename)
    if err != nil {
        return err
    }

    // Return result (template or JSON)
}
```

### HTMX File Upload Form

```html
<!-- Source: htmx.org/examples/file-upload -->
<form hx-encoding="multipart/form-data"
      hx-post="/upload"
      hx-target="#upload-results">
    <input type="file" name="files" multiple accept=".pdf">
    <button type="submit">Upload</button>
    <progress id="upload-progress" value="0" max="100"></progress>
</form>

<script>
htmx.on('#upload-form', 'htmx:xhr:progress', function(evt) {
    htmx.find('#upload-progress').setAttribute('value',
        evt.detail.loaded / evt.detail.total * 100);
});
</script>
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Polling directories | fsnotify inotify/kqueue | Standard since Go 1.4 | Near-instant detection vs. polling delay |
| Fetch API for uploads | XMLHttpRequest with progress | N/A - Fetch never had upload progress | Only way to show upload progress |
| Custom toast JS | HTMX OOB swaps | HTMX 1.0+ | No custom JS needed, server-controlled |

**Deprecated/outdated:**
- `ioutil.ReadFile`: Use `os.ReadFile` (Go 1.16+)
- Polling-based file watchers: Inefficient, use fsnotify

## Open Questions

Things that couldn't be fully resolved:

1. **Inbox configuration persistence**
   - What we know: UI can override env var configuration, multiple directories supported
   - What's unclear: Where to store configuration - database table or config file?
   - Recommendation: New `inboxes` table in database with path, enabled, error_path columns; allows UI management and persists across restarts

2. **Maximum file size limit**
   - What we know: Echo can limit via middleware, large files should stream not buffer
   - What's unclear: What's a reasonable limit for PDF documents?
   - Recommendation: Default 100MB limit configurable via env var; most PDFs are under 50MB

3. **Concurrent inbox processing**
   - What we know: Multiple files may appear simultaneously in inbox
   - What's unclear: How many concurrent ingestions to allow?
   - Recommendation: Use a semaphore/worker pool with 4 concurrent ingestions (matches queue worker default)

## Sources

### Primary (HIGH confidence)
- [fsnotify pkg.go.dev](https://pkg.go.dev/github.com/fsnotify/fsnotify) - API documentation, Watcher type, Event handling
- [fsnotify GitHub](https://github.com/fsnotify/fsnotify) - README, limitations, recursive watching status
- [h2non/filetype pkg.go.dev](https://pkg.go.dev/github.com/h2non/filetype) - PDF detection API
- [MDN Drag and Drop API](https://developer.mozilla.org/en-US/docs/Web/API/HTML_Drag_and_Drop_API/File_drag_and_drop) - Event handlers, dataTransfer API
- [htmx.org File Upload](https://htmx.org/examples/file-upload/) - hx-encoding, progress events

### Secondary (MEDIUM confidence)
- [Echo File Upload Cookbook](https://echo.labstack.com/docs/cookbook/file-upload) - Multipart form handling
- [templUI Toast](https://templui.io/docs/components/toast) - Component API, HTMX integration
- [HTMX hx-swap-oob](https://htmx.org/attributes/hx-swap-oob/) - Out-of-band swap pattern
- [yarlson.dev HTMX Toasts](https://yarlson.dev/blog/htmx-toast/) - Toast with auto-dismiss pattern

### Tertiary (LOW confidence)
- WebSearch results for drag-and-drop patterns - community patterns, need validation
- WebSearch results for parallel upload - general patterns, not Go/HTMX specific

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - fsnotify and filetype are well-documented, official Echo docs available
- Architecture: HIGH - Patterns derived from official documentation and existing codebase patterns
- Pitfalls: MEDIUM - Some based on official docs, others from community experience

**Research date:** 2026-02-02
**Valid until:** 2026-03-02 (30 days - stable libraries, mature ecosystem)
