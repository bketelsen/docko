---
phase: 03-processing
plan: 05
subsystem: ui
tags: [sse, htmx, real-time, status-updates, templ]

# Dependency graph
requires:
  - phase: 03-04
    provides: Processing queue and job handler
  - phase: 02-ingestion
    provides: Document model and file storage
provides:
  - Real-time SSE status updates (HTML partials)
  - StatusBroadcaster for pub/sub
  - Document status UI components
  - Retry handler for failed documents
  - Documents admin page
affects: [04-search, 05-organize]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - SSE with HTML partials (not JSON) for HTMX compatibility
    - Status broadcaster pub/sub pattern
    - HTMX sse-swap for live DOM updates

key-files:
  created:
    - internal/processing/status.go
    - internal/handler/status.go
    - templates/pages/admin/documents.templ
    - templates/partials/document_status.templ
    - templates/partials/bulk_progress.templ
  modified:
    - internal/handler/handler.go
    - internal/handler/documents.go
    - internal/processing/processor.go
    - cmd/server/main.go
    - templates/layouts/admin.templ

key-decisions:
  - "SSE sends HTML partials (not JSON) for HTMX sse-swap compatibility"
  - "StatusBroadcaster uses sync.RWMutex with subscriber limit (100 max)"
  - "30-second heartbeat keeps SSE connections alive"
  - "Document UUID used consistently for storage paths and database"

patterns-established:
  - "SSE HTML partial pattern: render templ component to buffer, send as SSE data"
  - "Status badge pattern: inline-flex with conditional rendering based on status"
  - "Retry pattern: reset status to pending, re-enqueue job"

# Metrics
duration: 15min
completed: 2026-02-03
---

# Phase 03 Plan 05: Status Display Summary

**Real-time SSE status updates with HTML partials for HTMX, document list page with live Processing/Complete/Failed badges, and retry handler for failed documents**

## Performance

- **Duration:** 15 min
- **Started:** 2026-02-03T02:08:00Z
- **Completed:** 2026-02-03T02:23:37Z
- **Tasks:** 3
- **Files modified:** 12

## Accomplishments

- StatusBroadcaster manages SSE subscriptions with context-aware cleanup
- SSE endpoint streams HTML partials (not JSON) for HTMX compatibility
- Processor broadcasts status changes during processing lifecycle
- Document status badges with Processing/Complete/Failed states
- Retry button re-queues failed documents for processing
- Admin documents page with live SSE status updates
- Bulk progress summary shows "X of Y processed"

## Task Commits

Each task was committed atomically:

1. **Task 1: Implement status broadcaster and SSE endpoint** - `0758cda` (feat)
2. **Task 2: Create status UI components and retry handler** - `5ad5eab` (feat)
3. **Task 3: Human verification** - Checkpoint approved by user

**Bug fix during verification:** `8cc4751` (fix) - consistent UUID for storage/database

**Plan metadata:** (this commit)

## Files Created/Modified

- `internal/processing/status.go` - StatusBroadcaster with Subscribe/Broadcast pub/sub
- `internal/handler/status.go` - SSE endpoint streaming HTML partials
- `templates/pages/admin/documents.templ` - Document list with SSE status updates
- `templates/partials/document_status.templ` - Status badge component
- `templates/partials/bulk_progress.templ` - Bulk upload progress summary
- `internal/handler/documents.go` - Retry handler and document list
- `internal/processing/processor.go` - Broadcast status changes during processing
- `internal/handler/handler.go` - Route registration and broadcaster injection
- `templates/layouts/admin.templ` - Added HTMX SSE extension
- `cmd/server/main.go` - StatusBroadcaster initialization
- `sqlc/queries/documents.sql` - Clear processing error on retry

## Decisions Made

- **SSE HTML partials:** HTMX sse-swap expects HTML content, not JSON. Render templ components to buffer and send as SSE data field.
- **30-second heartbeat:** Keeps SSE connections alive through proxies and load balancers.
- **100 subscriber limit:** Prevents resource exhaustion from too many SSE connections.
- **Consistent UUID:** Fixed bug where upload generated one UUID for database but different UUID for storage path - now uses single UUID throughout.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Inconsistent UUID for document storage and database**
- **Found during:** Task 3 (human verification)
- **Issue:** Upload handler generated UUID for database insert, but then CreateDocument generated a different UUID for storage path
- **Fix:** Pass documentID to CreateDocument so same UUID is used for both database and file storage
- **Files modified:** internal/handler/documents.go, internal/document/document.go
- **Verification:** Upload now uses same UUID in database and storage path
- **Committed in:** 8cc4751

---

**Total deviations:** 1 auto-fixed (bug)
**Impact on plan:** Bug fix was essential for correct file storage/retrieval. No scope creep.

## Issues Encountered

None - SSE endpoint, status components, and retry handler all worked as planned after bug fix.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Phase 3 (Processing) complete
- Full document pipeline: upload -> storage -> OCR -> text extraction -> thumbnail -> status updates
- Ready for Phase 4 (Search) - documents have extracted text for indexing
- Ready for Phase 5 (Organize) - documents ready for tagging and organization

---
*Phase: 03-processing*
*Completed: 2026-02-03*
