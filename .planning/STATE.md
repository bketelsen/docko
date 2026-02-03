# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-02-02)

**Core value:** Find any document instantly AND automate the tagging/filing that's currently manual
**Current focus:** Phase 3 - Processing

## Current Position

Phase: 3 of 8 (Processing)
Plan: 1 of 5 in current phase
Status: In progress
Last activity: 2026-02-03 - Completed 03-01-PLAN.md

Progress: [#######---] 69%

## Performance Metrics

**Velocity:**
- Total plans completed: 9
- Average duration: 6.0 min
- Total execution time: 0.90 hours

**By Phase:**

| Phase | Plans | Total | Avg/Plan |
|-------|-------|-------|----------|
| 01-foundation | 3 | 10 min | 3.3 min |
| 02-ingestion | 5 | 39 min | 7.8 min |
| 03-processing | 1 | 5 min | 5.0 min |

**Recent Trend:**
- Last 5 plans: 02-02 (3 min), 02-03 (2 min), 02-04 (3 min), 02-05 (30 min), 03-01 (5 min)
- Trend: 03-01 back to normal speed (no checkpoint)

*Updated after each plan completion*

## Accumulated Context

### Decisions

Decisions are logged in PROJECT.md Key Decisions table.
Recent decisions affecting current work:

- Use gen_random_uuid() over uuid_generate_v4() for UUID generation
- 5-minute visibility timeout for job queue processing
- 2-level UUID sharding (ab/c1/uuid.ext) for storage paths
- One correspondent per document (1:1 relationship)
- Full jitter formula: random(0, min(cap, base * 2^attempt)) for retry backoff
- Default 4 workers per queue with 1-second poll interval
- Copy file first, then check for duplicate (avoids holding file in memory)
- Queue workers not started yet (no handlers registered)
- Multiple inboxes in database (not config file) for UI management
- duplicate_action enum per inbox (delete/rename/skip)
- INBOX_PATH env var optional for default inbox
- 500ms debounce delay for file watcher events
- 4 concurrent inbox workers via semaphore
- PDF validation via magic bytes before ingestion
- Inbox watcher runs in background goroutine with cancellable context
- HTMX partial updates for inbox toggle and delete operations
- OCRmyPDF runs as persistent Docker service (like postgres) with inotify watcher
- App communicates with OCR via shared volumes (ocr-input, ocr-output)
- Thumbnails generated in app container, OCR in separate service

### Pending Todos

None yet.

### Blockers/Concerns

None yet.

## Session Continuity

Last session: 2026-02-03T01:51:47Z
Stopped at: Completed 03-01-PLAN.md
Resume file: None

---
*Next action: Continue Phase 03 - Execute 03-02-PLAN.md (text extraction implementation)*
