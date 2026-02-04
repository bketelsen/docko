---
phase: 12-queues-detail
plan: 01
subsystem: database
tags: [sqlc, postgres, enum, jobs, queues]

# Dependency graph
requires:
  - phase: 03-processing
    provides: jobs table with job_status enum
provides:
  - dismissed status for soft-clearing failed jobs
  - queue-specific job queries with document info via LEFT JOIN LATERAL
  - bulk operations for dismiss and retry failed jobs
affects: [12-02, 12-03]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - LEFT JOIN LATERAL for safe JSONB payload extraction
    - dismissed status preserves audit trail while hiding from active lists

key-files:
  created:
    - internal/database/migrations/011_job_dismissed.sql
  modified:
    - sqlc/queries/jobs.sql

key-decisions:
  - "dismissed status added after failed in enum (preserves audit trail)"
  - "LEFT JOIN LATERAL for safe document_id extraction from JSONB payload"
  - "Down migration left as no-op (PostgreSQL enum value removal limitation)"

patterns-established:
  - "Queue detail queries return document name via LEFT JOIN LATERAL"
  - "Bulk operations (dismiss, retry) use execrows return type"

# Metrics
duration: 3min
completed: 2026-02-04
---

# Phase 12 Plan 01: Job Dismissed Status and Queue Queries Summary

**Dismissed job status enum value and 6 queue-specific queries with LEFT JOIN LATERAL for document info**

## Performance

- **Duration:** 3 min
- **Started:** 2026-02-04T02:09:28Z
- **Completed:** 2026-02-04T02:12:00Z
- **Tasks:** 2
- **Files modified:** 2

## Accomplishments
- Added 'dismissed' value to job_status enum for soft-clearing failed jobs
- Created 6 new sqlc queries for queue detail views with document information
- LEFT JOIN LATERAL pattern safely extracts document_id from JSONB payload

## Task Commits

Each task was committed atomically:

1. **Task 1: Create migration for dismissed status** - `ab700ea` (feat)
2. **Task 2: Add queue-specific job queries with document info** - `228a4aa` (feat)

## Files Created/Modified
- `internal/database/migrations/011_job_dismissed.sql` - Adds 'dismissed' value to job_status enum
- `sqlc/queries/jobs.sql` - 6 new queries for queue detail views

## New Queries

| Query | Purpose | Return Type |
|-------|---------|-------------|
| GetFailedJobsForQueue | Failed jobs with document name | many (row with job + doc info) |
| GetRecentCompletedJobsForQueue | Completed jobs in last 24h | many (row with job + doc info) |
| DismissFailedJobsForQueue | Bulk dismiss all failed jobs | execrows (affected count) |
| DismissJob | Dismiss single failed job | one (updated job) |
| ResetFailedJobsForQueue | Retry all failed jobs in queue | execrows (affected count) |
| GetQueueNames | Distinct queue names | many (strings) |

## Decisions Made
- Used LEFT JOIN LATERAL for safe JSONB payload extraction (handles missing document_id or deleted documents)
- Down migration is no-op (PostgreSQL cannot remove enum values without recreating type)
- 'dismissed' added after 'failed' to maintain logical enum ordering

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

- Database not running initially - started with `docker compose up -d`
- sqlc generated files are gitignored (regenerated at build time) - only query source files committed

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- Migration ready to run (will execute automatically on app startup)
- All 6 queries ready for use in queue detail handler (Plan 02)
- JobStatusDismissed constant available in sqlc models

---
*Phase: 12-queues-detail*
*Completed: 2026-02-04*
