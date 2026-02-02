---
phase: 01-foundation
plan: 02
subsystem: queue
tags: [postgresql, jobs, queue, worker, backoff, jitter]

# Dependency graph
requires:
  - phase: 01-01
    provides: jobs table with SKIP LOCKED compatible schema
provides:
  - job queue implementation with worker pool
  - exponential backoff with full jitter for retries
  - transactional job enqueue support
affects: [01-03, 02-ingestion, 03-processing]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - Worker pool with graceful shutdown
    - Exponential backoff with full jitter (AWS Builders' Library)
    - Handler registration pattern for job types

key-files:
  created:
    - internal/queue/queue.go
    - internal/queue/queue_test.go
  modified: []

key-decisions:
  - "Full jitter formula: random(0, min(cap, base * 2^attempt))"
  - "Default 4 workers per queue"
  - "1-second poll interval for job pickup"

patterns-established:
  - "Job handler registration: RegisterHandler(jobType, func) before Start()"
  - "EnqueueTx for transactional job creation within existing transaction"
  - "Graceful shutdown: Stop() waits for all workers to complete"

# Metrics
duration: 3min
completed: 2026-02-02
---

# Phase 01 Plan 02: Job Queue Implementation Summary

**PostgreSQL-backed job queue with worker pool, SKIP LOCKED dequeue, and exponential backoff with full jitter**

## Performance

- **Duration:** 3 min
- **Started:** 2026-02-02T20:26:12Z
- **Completed:** 2026-02-02T20:29:00Z
- **Tasks:** 2
- **Files modified:** 2

## Accomplishments

- Implemented Queue struct with configurable worker pool and handler registration
- Used SKIP LOCKED pattern via sqlc DequeueJobs for atomic job claiming
- Added exponential backoff with full jitter per AWS Builders' Library formula
- Jobs marked as failed after max_attempts exhausted
- Clean shutdown via Stop() waits for all workers to complete
- EnqueueTx supports transactional job creation

## Task Commits

Each task was committed atomically:

1. **Task 1: Implement job queue with worker loop** - `bb06434` (feat)
2. **Task 2: Add unit test for retry delay calculation** - `43daa84` (test)

## Files Created/Modified

- `internal/queue/queue.go` - Queue struct with worker pool, Enqueue, EnqueueTx, RegisterHandler, Start, Stop
- `internal/queue/queue_test.go` - Tests for backoff calculation, default config, and handler registration

## Decisions Made

- **Full jitter formula:** Using AWS Builders' Library recommended approach: random(0, min(cap, base * 2^attempt))
- **Default 4 workers:** Reasonable default for concurrent job processing
- **1-second poll interval:** Balance between responsiveness and database load

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Queue ready for document ingestion pipeline
- Handler registration allows easy addition of job types
- Transactional enqueue enables atomic document creation + job enqueue
- Tests verify retry logic correctness

---
*Phase: 01-foundation*
*Completed: 2026-02-02*
