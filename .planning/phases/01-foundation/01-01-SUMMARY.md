---
phase: 01-foundation
plan: 01
subsystem: database
tags: [postgresql, sqlc, storage, uuid, jobs]

# Dependency graph
requires: []
provides:
  - documents table with UUID primary key and content hash uniqueness
  - jobs table with SKIP LOCKED compatible schema for queue processing
  - document_events table for audit trail
  - storage service with UUID-based file sharding
affects: [01-02, 02-ingestion, 03-processing]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - UUID primary keys with gen_random_uuid()
    - Content hash for duplicate detection
    - SKIP LOCKED for safe concurrent job dequeue
    - 2-level directory sharding for file storage

key-files:
  created:
    - internal/database/migrations/003_documents.sql
    - sqlc/queries/documents.sql
    - sqlc/queries/jobs.sql
    - internal/storage/storage.go
  modified: []

key-decisions:
  - "Use gen_random_uuid() over uuid_generate_v4() for consistency"
  - "Job visibility timeout of 5 minutes for processing"
  - "2-level UUID sharding (ab/c1/uuid.ext) for storage paths"
  - "One correspondent per document (not many-to-many)"

patterns-established:
  - "UUID sharding: first 2 chars / next 2 chars / full-uuid.ext"
  - "Job dequeue with SKIP LOCKED and visible_until for stale job recovery"
  - "Document events for audit trail with duration_ms tracking"

# Metrics
duration: 3min
completed: 2026-02-02
---

# Phase 01 Plan 01: Database Schema and Storage Service Summary

**PostgreSQL schema for documents, jobs, and events with UUID-sharded file storage service**

## Performance

- **Duration:** 3 min
- **Started:** 2026-02-02T20:21:05Z
- **Completed:** 2026-02-02T20:23:48Z
- **Tasks:** 3
- **Files modified:** 4

## Accomplishments

- Created documents table with UUID PK, content hash uniqueness constraint, and PDF metadata fields
- Created jobs table with SKIP LOCKED-compatible schema for safe concurrent queue processing
- Created document_events table for complete audit trail of document processing
- Created tags, correspondents, and junction tables for future categorization features
- Built storage service with 2-level UUID sharding and atomic copy-with-hash operation

## Task Commits

Each task was committed atomically:

1. **Task 1: Create database migration** - `9074594` (feat)
2. **Task 2: Create sqlc queries** - `c41b242` (feat)
3. **Task 3: Create storage service** - `68b0a08` (feat)

## Files Created/Modified

- `internal/database/migrations/003_documents.sql` - Migration for all 7 new tables
- `sqlc/queries/documents.sql` - Document and event CRUD queries
- `sqlc/queries/jobs.sql` - Job queue queries with SKIP LOCKED
- `internal/storage/storage.go` - File storage with UUID sharding and hash-while-copy

## Decisions Made

- **gen_random_uuid() over uuid_generate_v4():** Using built-in PostgreSQL function rather than extension
- **5-minute visibility timeout:** Jobs become re-available after 5 minutes if worker crashes
- **2-level UUID sharding:** First 2 chars then next 2 chars creates balanced directory tree
- **One correspondent per document:** Using 1:1 relationship rather than many-to-many for simplicity

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Database schema ready for ingestion pipeline
- Storage service ready for file operations
- Job queue ready for background processing
- sqlc queries generated and type-safe

---
*Phase: 01-foundation*
*Completed: 2026-02-02*
