# Phase 4: Viewing - Research

**Researched:** 2026-02-02
**Domain:** PDF viewing, file serving, document detail UI
**Confidence:** HIGH

## Summary

This phase implements PDF viewing in a modal overlay, file download capabilities, and a document detail page with metadata tabs. The implementation builds on existing infrastructure: the document service already provides `OriginalPath()` and `ThumbnailPath()` methods, and sqlc queries exist for fetching document metadata.

The standard approach uses PDF.js for in-browser PDF rendering with custom controls, Echo's built-in file serving methods for downloads, and templUI components (Dialog, Tabs, Breadcrumb) for the UI. HTMX patterns enable the modal to load PDF viewer content on-demand.

**Primary recommendation:** Use PDF.js with custom canvas rendering for the modal viewer, Echo's `c.Inline()` and `c.Attachment()` for file serving, and templUI's Dialog component for the modal wrapper with keyboard/backdrop dismiss.

## Standard Stack

The established libraries/tools for this domain:

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| PDF.js | 5.4.624 | Browser PDF rendering | Mozilla's official PDF viewer, only standard for pure-JS PDF rendering |
| pdfjs-dist | latest | NPM distribution of PDF.js | Pre-built distribution for easy integration |
| templUI Dialog | 1.4.0 | Modal wrapper component | Already in project, provides accessibility, JS API |
| templUI Tabs | 1.4.0 | Tabbed content panels | Already in project, handles tab switching |
| templUI Breadcrumb | 1.4.0 | Navigation breadcrumbs | Already in project, semantic structure |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| Echo c.Inline() | built-in | Serve PDF for viewing | Display PDF in iframe or new tab fallback |
| Echo c.Attachment() | built-in | Serve PDF for download | Trigger browser download dialog |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| PDF.js canvas rendering | Browser native PDF (iframe/embed) | Less control over UI, inconsistent across browsers |
| PDF.js | pdf-lib | pdf-lib is for creation/editing, not viewing |
| Custom modal | templUI Dialog | templUI handles accessibility, focus trapping, keyboard |

**Installation:**
```bash
# templUI components (run from project root)
templui add dialog tabs breadcrumb

# PDF.js via CDN (add to template head)
# No npm install needed - use unpkg CDN
```

## Architecture Patterns

### Recommended Project Structure
```
internal/handler/
  documents.go          # Add ViewDocument, DownloadDocument handlers
templates/
  pages/admin/
    document_detail.templ   # Document detail page with tabs
  partials/
    pdf_viewer.templ        # PDF viewer modal content (loaded via HTMX)
static/js/
    pdf-viewer.js           # PDF.js initialization and controls
components/
    dialog/                 # templUI Dialog (to be added)
    tabs/                   # templUI Tabs (to be added)
    breadcrumb/             # templUI Breadcrumb (to be added)
```

### Pattern 1: File Serving Endpoints
**What:** Separate endpoints for viewing (inline) vs downloading (attachment)
**When to use:** Any document/file serving scenario
**Example:**
```go
// Source: Echo documentation - https://echo.labstack.com/docs/cookbook/file-download

// View PDF in browser (for iframe or direct link)
func (h *Handler) ViewPDF(c echo.Context) error {
    docID, _ := uuid.Parse(c.Param("id"))
    doc, _ := h.db.Queries.GetDocument(ctx, docID)
    pdfPath := h.docSvc.OriginalPath(doc)
    return c.Inline(pdfPath, doc.OriginalFilename)
}

// Download PDF as attachment
func (h *Handler) DownloadPDF(c echo.Context) error {
    docID, _ := uuid.Parse(c.Param("id"))
    doc, _ := h.db.Queries.GetDocument(ctx, docID)
    pdfPath := h.docSvc.OriginalPath(doc)
    return c.Attachment(pdfPath, doc.OriginalFilename)
}
```

### Pattern 2: HTMX Modal Loading
**What:** Load modal content on-demand, not embedded in page
**When to use:** Heavy content like PDF viewers that shouldn't load until needed
**Example:**
```html
<!-- Source: https://htmx.org/examples/modal-custom/ -->

<!-- Button triggers modal load -->
<button
    hx-get="/documents/{id}/viewer"
    hx-target="body"
    hx-swap="beforeend">
    View PDF
</button>

<!-- Server returns modal HTML with PDF viewer -->
<!-- Modal is appended to body, removed on close -->
```

### Pattern 3: PDF.js Canvas Rendering
**What:** Load PDF into canvas with custom controls
**When to use:** Modal viewer with zoom/navigation controls
**Example:**
```javascript
// Source: https://mozilla.github.io/pdf.js/examples/

// Load PDF document
const loadingTask = pdfjsLib.getDocument(pdfUrl);
const pdf = await loadingTask.promise;

// Render a page
const page = await pdf.getPage(pageNum);
const scale = 1.5;
const viewport = page.getViewport({ scale });

const canvas = document.getElementById('pdf-canvas');
const context = canvas.getContext('2d');

// Handle high-DPI displays
const outputScale = window.devicePixelRatio || 1;
canvas.width = Math.floor(viewport.width * outputScale);
canvas.height = Math.floor(viewport.height * outputScale);
canvas.style.width = Math.floor(viewport.width) + "px";
canvas.style.height = Math.floor(viewport.height) + "px";

const transform = outputScale !== 1
    ? [outputScale, 0, 0, outputScale, 0, 0]
    : null;

await page.render({
    canvasContext: context,
    transform: transform,
    viewport: viewport
}).promise;
```

### Pattern 4: templUI Dialog with HTMX
**What:** Use Dialog.Content standalone for HTMX-loaded modals
**When to use:** Modal content that's fetched dynamically
**Example:**
```templ
// Source: https://templui.io/docs/components/dialog

// Modal returned by HTMX endpoint
templ PDFViewerModal(docID string, filename string) {
    @dialog.Content(dialog.ContentProps{
        ID: "pdf-viewer-modal",
        Open: true,
    }) {
        @dialog.Header() {
            @dialog.Title() { { filename } }
        }
        <div id="pdf-container">
            <canvas id="pdf-canvas"></canvas>
        </div>
        @dialog.Footer() {
            @dialog.Close() { Close }
        }
    }
}
```

### Anti-Patterns to Avoid
- **Embedding full PDF.js viewer in page HTML:** Heavy, loads even when not viewing. Load modal content on-demand.
- **Using iframe for modal viewer:** Less control over UI, can't implement custom zoom/nav.
- **Serving files without auth check:** Always verify user is authenticated before serving documents.
- **Hardcoding file paths:** Use document service methods (OriginalPath, ThumbnailPath).

## Don't Hand-Roll

Problems that look simple but have existing solutions:

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| PDF rendering in browser | Custom PDF parser | PDF.js | PDF spec is 1000+ pages, edge cases everywhere |
| Modal accessibility | Custom focus trap | templUI Dialog | Focus trapping, keyboard nav, aria labels |
| Tab switching | Manual show/hide | templUI Tabs | State management, keyboard nav, accessibility |
| File download headers | Manual header setting | Echo c.Attachment() | Handles filename escaping, Content-Disposition |
| Breadcrumb structure | Manual nav links | templUI Breadcrumb | Semantic HTML, proper separators, accessibility |

**Key insight:** The UI components (modal, tabs, breadcrumbs) all have accessibility requirements that are easy to get wrong. templUI handles aria attributes, keyboard navigation, and focus management.

## Common Pitfalls

### Pitfall 1: PDF.js Worker Not Configured
**What goes wrong:** PDF rendering fails silently or throws "worker not found" error
**Why it happens:** PDF.js requires a web worker for performance; worker path must be set
**How to avoid:** Set workerSrc before loading any PDF
```javascript
pdfjsLib.GlobalWorkerOptions.workerSrc =
    'https://unpkg.com/pdfjs-dist@5.4.624/build/pdf.worker.min.mjs';
```
**Warning signs:** Console errors about workers, PDFs not rendering

### Pitfall 2: High-DPI Display Blurry Rendering
**What goes wrong:** PDF looks blurry on Retina/HiDPI displays
**Why it happens:** Canvas not scaled for devicePixelRatio
**How to avoid:** Always account for devicePixelRatio when setting canvas dimensions
**Warning signs:** PDF looks fuzzy compared to native viewer

### Pitfall 3: Modal Not Cleaning Up
**What goes wrong:** Old modal content persists, multiple modals stack, memory leaks
**Why it happens:** Modal HTML appended to body but not removed on close
**How to avoid:** Remove modal from DOM on close, not just hide
```javascript
// On close, remove the modal element entirely
modalElement.remove();
```
**Warning signs:** Modal shows stale content on re-open

### Pitfall 4: File Path Traversal
**What goes wrong:** Attacker accesses files outside document storage
**Why it happens:** Using user input directly in file paths
**How to avoid:** Always use document service methods that construct paths from UUID only
```go
// GOOD: Path constructed from UUID only
pdfPath := h.docSvc.OriginalPath(doc)

// BAD: Using user-provided filename in path
pdfPath := filepath.Join(storagePath, c.Param("filename"))
```
**Warning signs:** Security scanner findings, unexpected file access logs

### Pitfall 5: Missing Content-Type for PDF
**What goes wrong:** Browser tries to download instead of display, or shows raw bytes
**Why it happens:** Server not setting Content-Type: application/pdf
**How to avoid:** Use Echo's c.Inline() which sets Content-Type automatically, or set manually
**Warning signs:** PDF downloads instead of displaying in viewer

## Code Examples

Verified patterns from official sources:

### Echo File Serving
```go
// Source: https://echo.labstack.com/docs/cookbook/file-download

// Inline viewing (browser displays)
func (h *Handler) ViewDocument(c echo.Context) error {
    ctx := c.Request().Context()
    docID, err := uuid.Parse(c.Param("id"))
    if err != nil {
        return echo.NewHTTPError(http.StatusBadRequest, "invalid document ID")
    }

    doc, err := h.db.Queries.GetDocument(ctx, docID)
    if err != nil {
        return echo.NewHTTPError(http.StatusNotFound, "document not found")
    }

    pdfPath := h.docSvc.OriginalPath(&doc)
    if !h.storage.FileExists(pdfPath) {
        return echo.NewHTTPError(http.StatusNotFound, "file not found")
    }

    return c.Inline(pdfPath, doc.OriginalFilename)
}

// Attachment download (browser downloads)
func (h *Handler) DownloadDocument(c echo.Context) error {
    // Same logic as above, but:
    return c.Attachment(pdfPath, doc.OriginalFilename)
}
```

### PDF.js Initialization
```javascript
// Source: https://mozilla.github.io/pdf.js/examples/

// Initialize PDF.js
const pdfjsLib = window['pdfjs-dist/build/pdf'];
pdfjsLib.GlobalWorkerOptions.workerSrc =
    'https://unpkg.com/pdfjs-dist@5.4.624/build/pdf.worker.min.mjs';

// State
let pdfDoc = null;
let pageNum = 1;
let scale = 1.0;

// Load document
async function loadPDF(url) {
    const loadingTask = pdfjsLib.getDocument(url);
    pdfDoc = await loadingTask.promise;
    document.getElementById('page-count').textContent = pdfDoc.numPages;
    renderPage(pageNum);
}

// Render page
async function renderPage(num) {
    const page = await pdfDoc.getPage(num);
    const viewport = page.getViewport({ scale });

    const canvas = document.getElementById('pdf-canvas');
    const ctx = canvas.getContext('2d');

    // High-DPI support
    const outputScale = window.devicePixelRatio || 1;
    canvas.width = Math.floor(viewport.width * outputScale);
    canvas.height = Math.floor(viewport.height * outputScale);
    canvas.style.width = Math.floor(viewport.width) + 'px';
    canvas.style.height = Math.floor(viewport.height) + 'px';

    const transform = outputScale !== 1
        ? [outputScale, 0, 0, outputScale, 0, 0]
        : null;

    await page.render({
        canvasContext: ctx,
        transform: transform,
        viewport: viewport
    }).promise;

    document.getElementById('page-num').textContent = num;
}

// Navigation
function prevPage() {
    if (pageNum <= 1) return;
    pageNum--;
    renderPage(pageNum);
}

function nextPage() {
    if (pageNum >= pdfDoc.numPages) return;
    pageNum++;
    renderPage(pageNum);
}

// Zoom
function zoomIn() {
    scale += 0.25;
    renderPage(pageNum);
}

function zoomOut() {
    if (scale <= 0.5) return;
    scale -= 0.25;
    renderPage(pageNum);
}
```

### templUI Dialog JavaScript API
```javascript
// Source: https://templui.io/docs/components/dialog

// Open modal programmatically (useful for keyboard shortcuts)
window.tui.dialog.open("pdf-viewer-modal");

// Close modal
window.tui.dialog.close("pdf-viewer-modal");

// Check if open
const isOpen = window.tui.dialog.isOpen("pdf-viewer-modal");
```

### HTMX Modal Pattern
```html
<!-- Source: https://htmx.org/examples/modal-custom/ -->

<!-- Trigger button in document list or detail page -->
<button
    class="btn-primary"
    hx-get="/documents/{{ doc.ID }}/viewer"
    hx-target="body"
    hx-swap="beforeend"
    hx-trigger="click">
    View PDF
</button>

<!-- Keyboard shortcut trigger (optional) -->
<script>
document.addEventListener('keydown', (e) => {
    if (e.key === 'Escape') {
        const modal = document.getElementById('pdf-viewer-modal');
        if (modal) {
            window.tui.dialog.close('pdf-viewer-modal');
            modal.remove();
        }
    }
});
</script>
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Browser PDF plugin (Adobe) | PDF.js pure JavaScript | 2011 | No plugin dependencies |
| pdf.js v2 worker setup | pdf.js v5 ES modules | 2023 | Use .mjs imports, modern bundling |
| Custom modal JS | Native dialog + templUI | 2022 | Better accessibility, less code |
| jQuery for DOM | HTMX + vanilla JS | 2020+ | Simpler, no jQuery dependency |

**Deprecated/outdated:**
- **PDF.js legacy builds**: Only needed for IE11 which is EOL. Use modern build.
- **embed/object tags for PDF**: Inconsistent cross-browser. Use PDF.js or iframe.
- **Adobe Reader plugin**: Discontinued. PDF.js is the standard.

## Open Questions

Things that couldn't be fully resolved:

1. **Mobile PDF viewer performance**
   - What we know: PDF.js works on mobile but can be slow for large PDFs
   - What's unclear: Whether to show simplified view or full controls on mobile
   - Recommendation: Start with same modal (responsive), optimize if performance issues arise

2. **PDF.js module format**
   - What we know: PDF.js 5.x uses ES modules (.mjs files)
   - What's unclear: Whether unpkg CDN serves modules correctly with CORS
   - Recommendation: Test CDN approach first; fall back to hosting files locally if issues

## Sources

### Primary (HIGH confidence)
- [PDF.js Examples](https://mozilla.github.io/pdf.js/examples/) - Canvas rendering, viewport, page navigation
- [Echo File Download Cookbook](https://echo.labstack.com/docs/cookbook/file-download) - c.Inline(), c.Attachment() usage
- [templUI Dialog](https://templui.io/docs/components/dialog) - Dialog component props, JS API, HTMX integration
- [templUI Tabs](https://templui.io/docs/components/tabs) - Tabs component props and usage
- [templUI Breadcrumb](https://templui.io/docs/components/breadcrumb) - Breadcrumb navigation component
- [HTMX Custom Modal](https://htmx.org/examples/modal-custom/) - On-demand modal loading pattern

### Secondary (MEDIUM confidence)
- [PDF.js GitHub](https://github.com/mozilla/pdf.js) - Package name (pdfjs-dist), version info
- [Echo pkg.go.dev](https://pkg.go.dev/github.com/labstack/echo/v4) - Context methods documentation

### Tertiary (LOW confidence)
- None - all claims verified with primary sources

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - All libraries verified with official docs
- Architecture: HIGH - Patterns from official examples and existing codebase
- Pitfalls: HIGH - Known issues documented in GitHub issues and official docs

**Research date:** 2026-02-02
**Valid until:** 2026-03-02 (30 days - PDF.js and templUI are stable)
