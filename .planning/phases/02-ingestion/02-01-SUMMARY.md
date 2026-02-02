---
phase: 02-ingestion
plan: 01
subsystem: api
tags: [pdf, upload, filetype, htmx, echo]

# Dependency graph
requires:
  - phase: 01-foundation
    provides: Document service with Ingest() method, storage, queue
provides:
  - Upload handler with PDF validation via magic bytes
  - POST /upload and POST /api/upload endpoints
  - Duplicate detection with is_duplicate response flag
  - JSON and HTMX response format support
affects: [02-02, 02-03, 03-processing]

# Tech tracking
tech-stack:
  added: [h2non/filetype, templui/toast]
  patterns: [magic-byte validation, multipart upload handling, content negotiation]

key-files:
  created: [internal/handler/upload.go, components/toast/toast.templ]
  modified: [internal/handler/handler.go, cmd/server/main.go, go.mod]

key-decisions:
  - "Use h2non/filetype for PDF validation via magic bytes (more reliable than extension)"
  - "Support both JSON (API) and HTML (HTMX) response formats via Accept header"
  - "Return 200 with is_duplicate flag for duplicates instead of error"

patterns-established:
  - "Content negotiation: Check Accept header for application/json vs HTML"
  - "Upload flow: temp file -> validate -> ingest -> cleanup"
  - "Multi-status response: 207 when batch upload has mixed results"

# Metrics
duration: 3min
completed: 2026-02-02
---

# Phase 02 Plan 01: Upload Handler Summary

**PDF upload handler with magic-byte validation, duplicate detection, and HTMX/JSON response support**

## Performance

- **Duration:** 3 min
- **Started:** 2026-02-02T21:20:48Z
- **Completed:** 2026-02-02T21:23:23Z
- **Tasks:** 3
- **Files modified:** 5

## Accomplishments

- Upload handler with UploadSingle and UploadMultiple methods
- PDF validation using h2non/filetype magic bytes (first 262 bytes)
- Non-PDF files rejected with 400 "Only PDF files are allowed"
- Duplicate files return 200 with existing document ID and is_duplicate flag
- Routes registered: GET /upload, POST /upload, POST /api/upload

## Task Commits

Each task was committed atomically:

1. **Task 1: Install dependencies and add toast component** - `9380609` (chore)
2. **Task 2: Create upload handler with PDF validation** - `042fa93` (feat)
3. **Task 3: Update Handler struct and register upload routes** - `60ea8be` (feat)

## Files Created/Modified

- `internal/handler/upload.go` - Upload handlers with PDF validation and response handling
- `internal/handler/handler.go` - Added docSvc field and upload routes
- `cmd/server/main.go` - Wire document service to handler
- `components/toast/toast.templ` - Toast component for upload notifications
- `go.mod` - Added h2non/filetype dependency

## Decisions Made

- **h2non/filetype for PDF validation**: More reliable than file extension checking; validates actual file content via magic bytes
- **Accept header content negotiation**: Allows same endpoint to serve API clients (JSON) and web UI (HTMX partials)
- **Duplicate returns 200, not error**: Duplicates are not errors; return existing document info with is_duplicate flag

## Deviations from Plan

None - plan executed exactly as written.

Note: fsnotify dependency not added to go.mod because nothing imports it yet. It will be properly added when inbox watcher is implemented in later plans. This is expected Go behavior.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Upload handler ready for UI integration (Plan 02)
- Document service integration complete
- Routes protected with auth middleware
- Ready for upload.js implementation in Plan 02

---
*Phase: 02-ingestion*
*Completed: 2026-02-02*
