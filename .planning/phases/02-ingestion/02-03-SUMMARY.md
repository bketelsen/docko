---
phase: 02-ingestion
plan: 03
subsystem: database
tags: [postgres, sqlc, inbox, configuration]

# Dependency graph
requires:
  - phase: 01-foundation
    provides: database connection, goose migrations, sqlc setup
provides:
  - Inboxes table for multiple directory configuration
  - inbox_events table for file processing logs
  - duplicate_action enum (delete/rename/skip)
  - InboxConfig with INBOX_PATH environment variable
affects: [inbox-watcher, file-processing, admin-settings]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - inbox directory configuration via database with env var default

key-files:
  created:
    - internal/database/migrations/004_inboxes.sql
    - sqlc/queries/inboxes.sql
  modified:
    - internal/config/config.go

key-decisions:
  - "Multiple inbox directories in database, not config file"
  - "duplicate_action enum per inbox (delete/rename/skip)"
  - "INBOX_PATH env var for optional default inbox"

patterns-established:
  - "Inbox events logged for all file processing actions"
  - "Error path per inbox for failed file handling"

# Metrics
duration: 2min
completed: 2026-02-02
---

# Phase 2 Plan 3: Inbox Configuration Summary

**Database schema and config for multiple inbox directories with duplicate handling and env var defaults**

## Performance

- **Duration:** 2 min
- **Started:** 2026-02-02T21:21:50Z
- **Completed:** 2026-02-02T21:23:44Z
- **Tasks:** 3
- **Files modified:** 3

## Accomplishments
- Inboxes table with path, name, enabled, error_path, duplicate_action columns
- inbox_events table for tracking all file processing activity
- duplicate_action enum supporting delete/rename/skip per inbox
- InboxConfig in config.go with INBOX_PATH and related env vars

## Task Commits

Each task was committed atomically:

1. **Task 1: Create inboxes migration** - `554192f` (feat)
2. **Task 2: Create inbox sqlc queries** - `97b2713` (feat) - committed with parallel plan
3. **Task 3: Add inbox configuration to config.go** - `3574005` (feat)

## Files Created/Modified
- `internal/database/migrations/004_inboxes.sql` - Inboxes and inbox_events tables with duplicate_action enum
- `sqlc/queries/inboxes.sql` - CRUD queries for inbox management plus event logging
- `internal/config/config.go` - InboxConfig struct with INBOX_PATH, ErrorSubdir, MaxFileSizeMB, ScanIntervalMs

## Decisions Made
- **Multiple inboxes in database:** Allows UI management without config file editing
- **duplicate_action per inbox:** Different directories may need different handling (delete original, rename, or skip)
- **Optional INBOX_PATH:** No default inbox created unless explicitly configured via env var
- **Event logging:** All file processing logged for debugging and audit trail

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
- Migration needed to run before sqlc could generate queries (tables must exist for introspection)
- Task 2 query file was captured in a parallel plan's commit (97b2713) due to concurrent execution

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- Schema ready for inbox watcher service implementation
- Config ready for inbox path configuration
- sqlc queries available for all CRUD operations

---
*Phase: 02-ingestion*
*Completed: 2026-02-02*
