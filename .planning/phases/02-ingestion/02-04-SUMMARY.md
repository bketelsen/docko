---
phase: 02-ingestion
plan: 04
subsystem: inbox
tags: [fsnotify, file-watcher, debouncing, pdf-validation, filetype]

# Dependency graph
requires:
  - phase: 02-03
    provides: inbox database schema (inboxes, inbox_events tables)
  - phase: 01-03
    provides: document service with Ingest() method
provides:
  - fsnotify-based directory watcher with debouncing
  - Inbox service coordinating watcher and document ingestion
  - PDF validation via magic bytes
  - Duplicate handling per inbox settings
  - Error file handling with move to error subdirectory
affects: [02-05, upload-handlers, server-startup]

# Tech tracking
tech-stack:
  added: [github.com/fsnotify/fsnotify]
  patterns: [debounced-file-watching, semaphore-concurrency-limiting]

key-files:
  created:
    - internal/inbox/watcher.go
    - internal/inbox/service.go

key-decisions:
  - "500ms debounce delay handles most write patterns"
  - "4 concurrent ingestion workers via semaphore"
  - "PDF validation via magic bytes before ingestion"
  - "Delete source file on successful import (per CONTEXT.md)"

patterns-established:
  - "Debouncer pattern: per-path timer reset on new events"
  - "Semaphore for limiting concurrent operations"
  - "Error files moved to subdirectory with timestamp suffix"

# Metrics
duration: 3min
completed: 2026-02-02
---

# Phase 02 Plan 04: Inbox Watcher Summary

**fsnotify-based inbox watcher with 500ms debouncing, PDF validation, and semaphore-limited concurrent ingestion**

## Performance

- **Duration:** 3 min
- **Started:** 2026-02-02T21:26:18Z
- **Completed:** 2026-02-02T21:28:54Z
- **Tasks:** 2
- **Files created:** 2

## Accomplishments
- Inbox watcher using fsnotify with per-file debouncing to handle chunked writes
- Service coordinates watcher with document ingestion, limits to 4 concurrent workers
- Scans existing files on startup (handles files added while service stopped)
- PDF validation via magic bytes prevents non-PDF ingestion attempts

## Task Commits

Each task was committed atomically:

1. **Task 1: Create inbox watcher with fsnotify and debouncing** - `3180c0a` (feat)
2. **Task 2: Create inbox service coordinating watcher and ingestion** - `40ac1e2` (feat)

## Files Created

- `internal/inbox/watcher.go` - fsnotify watcher with debouncer, watches directories for PDF files
- `internal/inbox/service.go` - Service coordinating watcher, document ingestion, and inbox configuration

## Decisions Made

- **500ms debounce delay:** Handles most write patterns (large files, atomic saves) without excessive latency
- **4 concurrent workers:** Matches queue worker default, prevents resource exhaustion
- **Magic bytes validation:** More reliable than extension check, uses h2non/filetype library already in go.mod
- **Timestamp suffix for error files:** Format `{base}_{YYYYMMDD-HHMMSS}.pdf` avoids collisions

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None - fsnotify dependency already partially present (upgraded from v1.7.0 to v1.9.0).

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Inbox watching infrastructure complete
- Ready for integration with server startup (plan 05)
- Ready for web upload handlers (plan 05)
- Existing document service Ingest() method works correctly with inbox service

---
*Phase: 02-ingestion*
*Completed: 2026-02-02*
