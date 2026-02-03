# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-02-02)

**Core value:** Find any document instantly AND automate the tagging/filing that's currently manual
**Current focus:** Phase 4 - Viewing (Complete)

## Current Position

Phase: 4 of 8 (Viewing)
Plan: 3 of 3 in current phase
Status: Phase complete
Last activity: 2026-02-03 - Completed 04-03-PLAN.md

Progress: [################----] 80%

## Performance Metrics

**Velocity:**
- Total plans completed: 16
- Average duration: 5.3 min
- Total execution time: 1.5 hours

**By Phase:**

| Phase | Plans | Total | Avg/Plan |
|-------|-------|-------|----------|
| 01-foundation | 3 | 10 min | 3.3 min |
| 02-ingestion | 5 | 39 min | 7.8 min |
| 03-processing | 5 | 28 min | 5.6 min |
| 04-viewing | 3 | 10 min | 3.3 min |

**Recent Trend:**

- Last 5 plans: 03-05 (15 min), 04-01 (2 min), 04-02 (5 min), 04-03 (3 min)
- Trend: Phase 4 completed efficiently

*Updated after each plan completion*

## Accumulated Context

### Decisions

Decisions are logged in PROJECT.md Key Decisions table.
Recent decisions affecting current work:

- Use gen_random_uuid() over uuid_generate_v4() for UUID generation
- 5-minute visibility timeout for job queue processing
- 2-level UUID sharding (ab/c1/uuid.ext) for storage paths
- One correspondent per document (1:1 relationship)
- Full jitter formula: random(0, min(cap, base * 2^attempt)) for retry backoff
- Default 4 workers per queue with 1-second poll interval
- Copy file first, then check for duplicate (avoids holding file in memory)
- Multiple inboxes in database (not config file) for UI management
- duplicate_action enum per inbox (delete/rename/skip)
- INBOX_PATH env var optional for default inbox
- 500ms debounce delay for file watcher events
- 4 concurrent inbox workers via semaphore
- PDF validation via magic bytes before ingestion
- Inbox watcher runs in background goroutine with cancellable context
- HTMX partial updates for inbox toggle and delete operations
- OCRmyPDF runs as persistent Docker service (like postgres) with inotify watcher
- App communicates with OCR via shared volumes (ocr-input, ocr-output)
- Thumbnails generated in app container, OCR in separate service
- ThumbnailPath returns .webp extension to match generated thumbnails
- 2-minute timeout prevents hanging on corrupt PDFs
- Placeholder fallback for unrenderable PDFs instead of error
- Bind mounts for OCR volumes (storage/ocr-input, storage/ocr-output) instead of Docker named volumes
- 100-char threshold to determine if embedded text is sufficient for search
- 5-minute timeout for OCR processing, 500ms polling interval
- Queue workers start on startup after handler registration
- Quarantine returns nil so job is marked completed (failure handled gracefully)
- SSE sends HTML partials (not JSON) for HTMX sse-swap compatibility
- 30-second heartbeat keeps SSE connections alive
- 100 subscriber limit for StatusBroadcaster
- docSvc.FileExists helper wraps storage.FileExists for handler access
- ServeThumbnail checks ThumbnailGenerated flag before attempting to serve
- Text extraction status shown instead of OCR status (TextContent field available)
- Storage path not displayed (computed dynamically, not stored)
- PDF.js 4.x legacy build for non-module script compatibility
- HTMX beforeend swap to append modal to body
- Canvas-based rendering with devicePixelRatio support for high-DPI

### Pending Todos

None yet.

### Blockers/Concerns

None yet.

## Session Continuity

Last session: 2026-02-03T14:16:37Z
Stopped at: Completed 04-03-PLAN.md (Phase 04 complete)
Resume file: None

---
*Next action: Begin Phase 05 - Search*
