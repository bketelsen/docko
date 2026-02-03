---
phase: 08-ai-integration
plan: 05
subsystem: ui
tags: [htmx, templ, queue-management, review-workflow]

# Dependency graph
requires:
  - phase: 08-03
    provides: AI service with suggestion storage and application
  - phase: 08-04
    provides: AI settings page and handler foundation
provides:
  - Review queue for pending AI suggestions
  - Queue dashboard for job monitoring
  - Retry controls for failed jobs
affects: []

# Tech tracking
tech-stack:
  added: []
  patterns:
    - HTMX outerHTML swap for row removal on accept/reject
    - Paginated list queries with COUNT for total
    - Job statistics aggregation by queue and status

key-files:
  created:
    - templates/pages/admin/ai_review.templ
    - templates/pages/admin/queue_dashboard.templ
  modified:
    - sqlc/queries/jobs.sql
    - internal/handler/ai.go
    - internal/ai/service.go
    - internal/handler/handler.go
    - templates/layouts/admin.templ

key-decisions:
  - "Reuse truncateFilename from document_detail.templ instead of duplicating"
  - "int64 for sqlc pagination params to match generated types"
  - "ApplySuggestionManual uses transaction for atomic tag/correspondent creation"

patterns-established:
  - "HTMX row removal: return empty string with outerHTML swap"
  - "Queue stats aggregation: GROUP BY queue_name, status"

# Metrics
duration: 7min
completed: 2026-02-03
---

# Phase 8 Plan 5: Review Queue and Queue Dashboard Summary

**Review queue for pending AI suggestions with accept/reject workflow, plus queue dashboard showing job statistics and retry controls**

## Performance

- **Duration:** 7 min
- **Started:** 2026-02-03T19:56:10Z
- **Completed:** 2026-02-03T20:03:07Z
- **Tasks:** 4
- **Files modified:** 7

## Accomplishments
- Review queue page lists pending AI suggestions with document links
- Accept/reject buttons apply or dismiss suggestions with HTMX updates
- Queue dashboard shows per-queue status counts (pending/processing/completed/failed)
- Failed jobs list with individual and bulk retry controls
- Recent activity table showing latest jobs across all queues

## Task Commits

Each task was committed atomically:

1. **Task 1: Add job statistics queries** - `2a69282` (feat)
2. **Task 2: Create review queue handlers and template** - `6c039a2` (feat)
3. **Task 3: Create queue dashboard handlers and template** - `181469e` (feat)
4. **Task 4: Register routes and add navigation** - `643e89a` (feat)

## Files Created/Modified
- `sqlc/queries/jobs.sql` - Added GetQueueStats, ListFailedJobs, ResetJobForRetry, ResetAllFailedJobs, GetRecentJobs
- `templates/pages/admin/ai_review.templ` - Review queue page with suggestion table and pagination
- `templates/pages/admin/queue_dashboard.templ` - Dashboard with stats cards, failed jobs, recent activity
- `internal/handler/ai.go` - Added ReviewQueuePage, AcceptSuggestion, RejectSuggestion, QueueDashboardPage, RetryJob, RetryAllFailedJobs
- `internal/ai/service.go` - Added ApplySuggestionManual method for user-accepted suggestions
- `internal/handler/handler.go` - Registered /ai/review, /ai/suggestions/:id/accept|reject, /queues routes
- `templates/layouts/admin.templ` - Added Queues link to navigation sidebar

## Decisions Made
- Reused truncateFilename helper from document_detail.templ to avoid redeclaration error (same package)
- ApplySuggestionManual wraps apply logic in transaction for atomicity
- Queue stats use GROUP BY aggregation for efficient counting
- Return empty string with outerHTML swap to remove rows on accept/reject

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
- truncateFilename redeclared error: resolved by using existing function from document_detail.templ
- Linter removing unused uuid import: resolved by adding handlers in same edit as import

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- Review queue ready for users to manage pending suggestions
- Queue dashboard provides visibility into job processing health
- All routes protected with RequireAuth middleware
- Ready for phase 08-06 (AI queue worker integration)

---
*Phase: 08-ai-integration*
*Completed: 2026-02-03*
