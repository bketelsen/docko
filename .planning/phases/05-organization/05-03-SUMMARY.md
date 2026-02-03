---
phase: 05-organization
plan: 03
subsystem: database, ui
tags: [sqlc, transactions, htmx, merge, correspondents]

# Dependency graph
requires:
  - phase: 05-organization/02
    provides: Correspondent CRUD operations and management UI
provides:
  - Correspondent merge functionality with atomic transactions
  - Multi-select UI for correspondent merging
  - Notes consolidation during merge
affects: [05-organization, document-management]

# Tech tracking
tech-stack:
  added: []
  patterns: [transaction-based merge operations, multi-select merge UI]

key-files:
  created: []
  modified:
    - sqlc/queries/correspondents.sql
    - internal/handler/correspondents.go
    - internal/handler/handler.go
    - templates/pages/admin/correspondents.templ

key-decisions:
  - "Merge uses database transaction for atomicity"
  - "Notes from merged correspondents prefixed with source name"
  - "Merge mode shows only when 2+ correspondents exist"
  - "Target selection prevents merging target into itself"

patterns-established:
  - "Transaction pattern: Begin, WithTx, operations, Commit/Rollback"
  - "Merge UI pattern: hidden checkboxes shown via JS toggle"

# Metrics
duration: 5min
completed: 2026-02-03
---

# Phase 05 Plan 03: Correspondent Merge Summary

**Correspondent merge with atomic transaction, multi-select UI, and notes consolidation**

## Performance

- **Duration:** 5 min
- **Started:** 2026-02-03T15:07:33Z
- **Completed:** 2026-02-03T15:13:00Z
- **Tasks:** 3
- **Files modified:** 4

## Accomplishments
- SQL queries for merge: update document references, get notes, append notes, batch delete
- Transactional merge handler ensuring atomic operation
- Merge mode UI with checkbox selection, target picker, and confirmation dialog
- Notes from merged correspondents preserved with attribution

## Task Commits

Each task was committed atomically:

1. **Task 1: Add merge SQL queries** - `55e1c7f` (feat)
2. **Task 2: Implement merge handler with transaction** - `64546b9` (feat)
3. **Task 3: Add merge UI to correspondent management page** - `911d522` (feat)

## Files Created/Modified
- `sqlc/queries/correspondents.sql` - Added 4 merge-related queries
- `internal/handler/correspondents.go` - Added MergeCorrespondents handler with transaction
- `internal/handler/handler.go` - Registered POST /correspondents/merge route
- `templates/pages/admin/correspondents.templ` - Added merge mode UI with checkboxes and target selector

## Decisions Made
- Merge uses database transaction to ensure atomicity (all or nothing)
- Notes from merged correspondents are prefixed with "--- Merged from {name} ---" for attribution
- Merge mode button only appears when there are 2+ correspondents
- Target cannot be in the selection list (validation in both JS and handler)
- CorrespondentList partial template added for HTMX response after merge

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
- Pre-existing compilation errors from incomplete 05-04 plan changes required restoration of documents.go and document_detail.templ to committed state (Rule 3 - Blocking)
- Generated sqlc code is gitignored, only source SQL file committed

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- Correspondent merge fully functional
- Ready for tag merge functionality (05-04) or document tagging
- Pattern established for future entity merge operations

---
*Phase: 05-organization*
*Completed: 2026-02-03*
