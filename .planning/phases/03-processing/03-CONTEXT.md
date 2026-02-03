# Phase 3: Processing - Context

**Gathered:** 2026-02-02
**Status:** Ready for planning

<domain>
## Phase Boundary

Extract text from PDFs and generate thumbnails asynchronously after upload. Text enables search (Phase 6). Thumbnails enable viewing (Phase 4). Processing happens via the queue infrastructure established in Phase 1.

</domain>

<decisions>
## Implementation Decisions

### OCR Handling
- Always OCR scanned PDFs (no embedded text = OCR)
- English language only
- Prefer embedded text where available, OCR only blank/image-only pages
- Use Tesseract via Docker service (similar to existing postgres container)

### Thumbnail Generation
- Medium size: 300px width
- WebP format
- First page only
- Use placeholder image for corrupt/empty/unrenderable PDFs

### Processing Feedback
- Show processing status (Processing/Complete/Failed) per document
- Status visible in both document list and detail views
- Live updates without page refresh (polling or SSE)
- Bulk uploads show summary count (3 of 10 processed) plus individual status

### Failure Behavior
- Retry 3x with backoff before marking failed
- All-or-nothing: both text extraction AND thumbnail must succeed
- Show error message and retry button to user on failure
- Quarantine documents that fail repeatedly (corrupt, password-protected)

### Testing
- Solid test coverage required for this phase

### Claude's Discretion
- Specific polling/SSE implementation for live updates
- Tesseract Docker image choice and configuration
- PDF rendering library for thumbnails
- Placeholder image design
- Quarantine storage location and mechanism
- Backoff timing parameters

</decisions>

<specifics>
## Specific Ideas

- Tesseract should run as a Docker service alongside postgres (docker-compose)
- Processing should leverage existing queue infrastructure from Phase 1

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 03-processing*
*Context gathered: 2026-02-02*
