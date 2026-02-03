---
phase: 03-processing
plan: 04
subsystem: processing
tags: [queue, async, job-handler, text-extraction, thumbnail, orchestration]

# Dependency graph
requires:
  - phase: 03-02
    provides: TextExtractor for embedded text and OCR fallback
  - phase: 03-03
    provides: ThumbnailGenerator for PDF to WebP conversion
provides:
  - Processor orchestrating text extraction and thumbnail generation
  - Queue handler for process_document job type
  - All-or-nothing transaction for processing completion
  - Quarantine mechanism for failed documents
affects: [04-viewing, 05-ai-tagging, 06-search]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Job handler pattern: HandleJob receives sqlc.Job and returns error"
    - "All-or-nothing transaction: both steps must succeed for completion"
    - "Quarantine pattern: set status to failed after max retries exhausted"

key-files:
  created:
    - internal/processing/processor.go
    - internal/processing/processor_test.go
  modified:
    - cmd/server/main.go

key-decisions:
  - "Queue workers start on application startup (after handler registration)"
  - "Processing status set to 'processing' before extraction/thumbnail"
  - "Quarantine returns nil so job is marked completed (failure handled gracefully)"

patterns-established:
  - "Processor orchestration: coordinate multiple extraction steps in single handler"
  - "All-or-nothing: transaction wraps all database updates for atomicity"
  - "Queue-processor integration: RegisterHandler + Start pattern"

# Metrics
duration: 4min
completed: 2026-02-03
---

# Phase 03 Plan 04: Processing Job Handler Summary

**Processing job handler orchestrating text extraction and thumbnail generation with all-or-nothing transaction pattern and quarantine for repeated failures**

## Performance

- **Duration:** 4 min
- **Started:** 2026-02-03T01:59:30Z
- **Completed:** 2026-02-03T02:03:36Z
- **Tasks:** 2
- **Files modified:** 3

## Accomplishments

- Processor orchestrates text extraction (03-02) and thumbnail generation (03-03)
- Queue handler registered for process_document job type
- All-or-nothing transaction ensures both steps succeed before marking complete
- Quarantine mechanism handles documents that fail after 3 retries
- Queue workers start automatically on application startup
- Processing events logged for audit trail

## Task Commits

Each task was committed atomically:

1. **Task 1: Implement processing job handler** - `d92d267` (feat)
2. **Task 2: Wire processor to queue and start workers** - `dac76b1` (feat)

## Files Created/Modified

- `internal/processing/processor.go` - Processor with HandleJob and quarantine
- `internal/processing/processor_test.go` - Tests documenting behavior patterns
- `cmd/server/main.go` - Register handler, check dependencies, start workers

## Decisions Made

1. **Queue workers start on startup** - After handler registration, q.Start() begins processing immediately. Previous versions deferred this since no handlers were registered.

2. **Quarantine returns nil** - When a document fails after max retries, quarantine() updates status to 'failed' and returns nil. This marks the job as completed (we've handled the failure) rather than leaving it in a retry loop.

3. **Processing status updated early** - Set to 'processing' before extraction begins, providing visibility into what's currently being worked on.

4. **Context for graceful shutdown** - Queue workers receive a cancellable context. On shutdown, cancel is called before q.Stop() to signal workers to finish current work.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None - both tasks completed without issues.

## User Setup Required

None - queue workers start automatically. Processing dependencies (pdftoppm, cwebp) are checked on startup with a warning if missing.

## Next Phase Readiness

- Documents uploaded via inbox are now processed automatically
- Text content is extracted and stored in database
- Thumbnails are generated and stored in storage/thumbnails
- Processing status and events available for UI display
- Ready for Phase 04 (Viewing) to display processed documents

---
*Phase: 03-processing*
*Completed: 2026-02-03*
