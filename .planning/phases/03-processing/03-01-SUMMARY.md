---
phase: 03-processing
plan: 01
subsystem: database, infra
tags: [postgresql, docker, ocrmypdf, webp, tesseract]

# Dependency graph
requires:
  - phase: 02-ingestion
    provides: documents table, file storage infrastructure
provides:
  - processing_status enum (pending, processing, completed, failed)
  - Processing columns on documents (text_content, thumbnail_generated, processing_error, processed_at)
  - OCRmyPDF Docker service with inotify file watching
  - Shared volumes (ocr-input, ocr-output) for app-to-OCR communication
  - Dockerfile with thumbnail tools (pdftoppm, cwebp)
  - Placeholder thumbnail for failed processing
affects: [03-02-text-extraction, 03-03-thumbnails, 03-04-status-display, 04-viewing]

# Tech tracking
tech-stack:
  added: [jbarlow83/ocrmypdf (docker), poppler-utils, libwebp-tools, inotify-tools]
  patterns: [persistent docker service with inotify watcher, shared volume communication]

key-files:
  created:
    - internal/database/migrations/005_processing.sql
    - Dockerfile
    - static/images/placeholder.webp
  modified:
    - sqlc/queries/documents.sql
    - docker-compose.yml

key-decisions:
  - "OCRmyPDF runs as persistent Docker service (like postgres) rather than ephemeral containers"
  - "App communicates with OCR via shared volumes (ocr-input, ocr-output)"
  - "Thumbnails generated in app container, OCR in separate service"

patterns-established:
  - "Docker service pattern: persistent container watching for input via inotify"
  - "Volume-based IPC: app writes to input volume, reads from output volume"

# Metrics
duration: 5min
completed: 2026-02-03
---

# Phase 03 Plan 01: Processing Infrastructure Summary

**Processing schema with status tracking, OCRmyPDF persistent Docker service with inotify watcher, and placeholder thumbnail for failures**

## Performance

- **Duration:** 5 min
- **Started:** 2026-02-03T01:47:15Z
- **Completed:** 2026-02-03T01:51:47Z
- **Tasks:** 2
- **Files modified:** 5

## Accomplishments
- Processing status enum and columns added to documents table
- OCRmyPDF service running alongside postgres via docker compose
- Shared volume architecture for app-to-OCR communication
- Dockerfile with thumbnail tools ready for production deployment
- Placeholder thumbnail for graceful failure handling

## Task Commits

Each task was committed atomically:

1. **Task 1: Add processing schema migration** - `f1d4ac6` (feat)
2. **Task 2: Add OCRmyPDF Docker service and thumbnail tools** - `10b347e` (feat)

## Files Created/Modified
- `internal/database/migrations/005_processing.sql` - Adds processing_status enum and columns
- `sqlc/queries/documents.sql` - New queries for processing status updates
- `docker-compose.yml` - OCRmyPDF service with inotify watcher
- `Dockerfile` - Multi-stage build with poppler-utils and libwebp-tools
- `static/images/placeholder.webp` - 8x8 gray WebP placeholder

## Decisions Made
- OCRmyPDF as persistent Docker service (vs ephemeral containers) - follows pattern established by postgres, simpler volume management
- Ubuntu-based ocrmypdf image uses apt-get for inotify-tools - different base than expected alpine
- Thumbnail tools in app Dockerfile, OCR in separate service - separation of concerns

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
- OCRmyPDF container uses Ubuntu (not Alpine), required apt-get instead of apk - detected and fixed via container logs
- Container entrypoint override needed to run custom watch script - resolved by adding explicit entrypoint in compose

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- Database schema ready for text_content and processing_status storage
- OCRmyPDF service operational and watching /input volume
- Next plans (03-02, 03-03) can implement text extraction and thumbnail generation
- Placeholder available for UI to display on failed thumbnails

---
*Phase: 03-processing*
*Completed: 2026-02-03*
