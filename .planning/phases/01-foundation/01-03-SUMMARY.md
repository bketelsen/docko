---
phase: 01-foundation
plan: 03
subsystem: api
tags: [go, document-service, storage, queue, transaction]

# Dependency graph
requires:
  - phase: 01-01
    provides: "Storage abstraction with UUID sharding and CopyAndHash"
  - phase: 01-02
    provides: "Job queue with EnqueueTx for transactional job creation"
provides:
  - "Document service coordinating storage, database, and queue"
  - "Ingest() for atomic document creation with duplicate detection"
  - "Event logging with timing and error details"
  - "STORAGE_PATH environment configuration"
affects: [02-upload, 03-processing, 04-search]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Transaction for document + event + job creation"
    - "Duplicate detection by content hash before storage"
    - "File cleanup on transaction failure"

key-files:
  created:
    - internal/document/document.go
  modified:
    - internal/config/config.go
    - cmd/server/main.go

key-decisions:
  - "Copy file first, then check for duplicate (avoids holding file in memory)"
  - "Clean up copied file on any failure (transaction rollback doesn't delete files)"
  - "Queue workers not started yet (no handlers registered)"

patterns-established:
  - "Document operations go through document.Service"
  - "Ingest returns (doc, isDuplicate, error) tuple"
  - "All document events have duration_ms for performance tracking"

# Metrics
duration: 4min
completed: 2026-02-02
---

# Phase 01 Plan 03: Document Service Summary

**Document service coordinating storage, database, and queue with atomic Ingest() and content-hash duplicate detection**

## Performance

- **Duration:** 4 min
- **Started:** 2026-02-02T15:31:00Z
- **Completed:** 2026-02-02T15:35:00Z
- **Tasks:** 3
- **Files modified:** 3

## Accomplishments
- Document service as single entry point for document operations
- Atomic Ingest(): file copy + hash -> duplicate check -> transaction (doc + event + job)
- Content-hash duplicate detection returns existing document instead of creating new
- Event logging with timing for performance tracking
- Services wired in main.go with graceful shutdown

## Task Commits

Each task was committed atomically:

1. **Task 1: Add STORAGE_PATH to config** - `4582a6e` (feat)
2. **Task 2: Create document service** - `a6a3fe7` (feat)
3. **Task 3: Wire services in main.go** - `a695310` (feat)

## Files Created/Modified
- `internal/document/document.go` - Document service with Ingest, GetByID, GetByHash, LogEvent
- `internal/config/config.go` - Added StorageConfig with STORAGE_PATH
- `cmd/server/main.go` - Wire storage, queue, document services with graceful shutdown

## Decisions Made
- Copy file before checking duplicate: simpler flow, avoids holding file in memory
- Clean up copied file on any transaction failure: database rollback doesn't delete files
- Queue workers not started: no handlers registered yet, will start when processing phase adds handlers
- Ingest returns (doc, isDuplicate, error): caller can distinguish new vs existing document

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None

## User Setup Required

None - no external service configuration required. STORAGE_PATH defaults to ./storage.

## Next Phase Readiness
- Document service ready for upload handlers (Phase 2)
- Queue initialized and ready for worker handlers (Phase 3)
- All foundation pieces connected: storage, database, queue, document service

---
*Phase: 01-foundation*
*Completed: 2026-02-02*
