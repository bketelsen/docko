---
phase: 02-ingestion
plan: 05
subsystem: ingestion-integration
tags: [inbox-ui, wiring, htmx, server-startup, graceful-shutdown]

# Dependency graph
requires:
  - phase: 02-01
    provides: document service with upload/ingest methods
  - phase: 02-02
    provides: upload UI and handlers
  - phase: 02-03
    provides: inbox database schema
  - phase: 02-04
    provides: inbox watcher service
provides:
  - Complete ingestion system integration
  - Inbox management UI at /inboxes
  - Inbox watcher starts on server startup
  - Graceful shutdown of inbox service
affects: [phase-03-processing, admin-navigation]

# Tech tracking
tech-stack:
  added: []
  patterns: [htmx-partial-updates, toggle-switch-ui, expandable-details]

key-files:
  modified:
    - cmd/server/main.go
    - internal/handler/handler.go
    - static/js/upload.js
  created:
    - internal/handler/inboxes.go
    - templates/pages/admin/inboxes.templ

key-decisions:
  - "Inbox watcher runs in background goroutine with cancellable context"
  - "Inbox service stopped before queue workers on shutdown"
  - "HTMX partial updates for inbox toggle and delete operations"
  - "Expandable details section loads events on demand"

patterns-established:
  - "Toggle switch pattern with HTMX swap"
  - "Expandable section with lazy-loaded content"
  - "Status indicator (green/red/gray) for service health"

# Metrics
duration: ~30min (includes checkpoint verification)
completed: 2026-02-03
---

# Phase 02 Plan 05: Integration and UI Summary

**Complete ingestion system with inbox management UI, server wiring, and graceful shutdown**

## Performance

- **Duration:** ~30 min (includes human verification checkpoint)
- **Started:** 2026-02-02T21:31:38Z
- **Completed:** 2026-02-03T01:01:32Z
- **Tasks:** 4 (3 auto + 1 checkpoint)
- **Files created:** 2
- **Files modified:** 3

## Accomplishments

- Inbox service wired into main.go with background goroutine startup
- Graceful shutdown sequence: server -> inbox watcher -> queue workers
- Inbox management handler with full CRUD operations
- Inbox management UI with status indicators, toggle switches, and event history
- Upload handler verified working with drag-and-drop functionality

## Task Commits

Each task was committed atomically:

1. **Task 1: Wire upload handler and inbox watcher in main.go** - `3070579` (feat)
2. **Task 2: Create inbox management handler** - `92454d7` (feat)
3. **Task 3: Create inbox management UI template** - `aa2a59f` (feat)
4. **Bug fix: Add Accept header to upload XHR** - `a8c1b40` (fix)

## Files Modified

- `cmd/server/main.go` - Added inbox service initialization, startup, and shutdown
- `internal/handler/handler.go` - Added inboxSvc to Handler, registered inbox routes
- `static/js/upload.js` - Added Accept: application/json header for proper response handling

## Files Created

- `internal/handler/inboxes.go` - Inbox management handlers (CRUD, toggle, events)
- `templates/pages/admin/inboxes.templ` - Inbox management UI with HTMX integration

## Decisions Made

- **Background goroutine for inbox watcher:** Uses cancellable context for clean shutdown
- **Shutdown order:** Stop inbox watcher before queue workers to prevent orphaned jobs
- **HTMX partial updates:** Toggle and delete operations swap individual inbox cards
- **Lazy-loaded events:** Expandable details section fetches events on demand

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed upload XHR Accept header**
- **Found during:** Checkpoint verification (Task 4)
- **Issue:** Upload responses returned HTML instead of JSON due to missing Accept header
- **Fix:** Added `xhr.setRequestHeader('Accept', 'application/json')` before send
- **Files modified:** static/js/upload.js
- **Commit:** a8c1b40

## Issues Encountered

None beyond the Accept header fix discovered during verification.

## Phase 02 Success Criteria Verification

All five phase success criteria from ROADMAP.md are now met:

1. **Drag-and-drop upload works** - Verified at /upload
2. **Bulk upload works** - Multiple file selection functional
3. **Inbox auto-detects PDFs** - Inbox watcher processes files automatically
4. **Duplicates detected by hash** - SHA256 hash comparison in document service
5. **Duplicate handling configurable per inbox** - delete/rename/skip options in UI

## User Setup Required

None - inbox configuration is done through the web UI at /inboxes.

## Next Phase Readiness

- Phase 02 (Ingestion) complete
- Ready for Phase 03 (Processing) - OCR, text extraction, AI processing
- All ingestion components integrated and tested
- Document storage and deduplication working

---
*Phase: 02-ingestion*
*Completed: 2026-02-03*
