---
phase: 15-pending-fixes
plan: 02
subsystem: ui
tags: [inbox, error-handling, pdf, file-count, badge]

# Dependency graph
requires:
  - phase: 02-ingestion
    provides: inbox system with error directory handling
provides:
  - Error count badges on inbox cards showing failed import visibility
  - InboxWithErrorCount type for template rendering
  - Helper functions for PDF counting and error path resolution
affects: [inbox-management, error-visibility, ui-polish]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Wrapper struct pattern for extending data with computed values"
    - "Template-defined types to avoid import cycles"

key-files:
  created: []
  modified:
    - internal/handler/inboxes.go
    - templates/pages/admin/inboxes.templ

key-decisions:
  - "InboxWithErrorCount type defined in template package to avoid import cycle"
  - "countPDFsInDir returns 0 if directory doesn't exist (graceful handling)"
  - "Error count shown in two places: name header badge and error path info"

patterns-established:
  - "Wrapper struct in template package for computed display values"
  - "Separate template for with-counts vs without-counts rendering"

# Metrics
duration: 3min
completed: 2026-02-04
---

# Phase 15 Plan 02: Inbox Error Count Badges Summary

**Error count badges on inbox cards showing PDF file count in error directories with destructive color styling**

## Performance

- **Duration:** 3 min
- **Started:** 2026-02-04T15:04:43Z
- **Completed:** 2026-02-04T15:08:39Z
- **Tasks:** 2
- **Files modified:** 2

## Accomplishments
- Inbox cards now display error count badge when error files exist
- Error path section shows file count alongside resolved path
- Handler calculates error counts by scanning error directories on page load

## Task Commits

Each task was committed atomically:

1. **Task 1: Add error count calculation to inbox handler** - `be27bba` (feat)
2. **Task 2: Add error count badge to inbox template** - `65b3105` (feat)

## Files Created/Modified
- `internal/handler/inboxes.go` - Added countPDFsInDir, resolveErrorPath helpers and InboxesPage modification
- `templates/pages/admin/inboxes.templ` - Added InboxWithErrorCount type, InboxesWithCounts and InboxCardWithErrors templates

## Decisions Made
- InboxWithErrorCount type defined in template package to avoid import cycle between handler and template
- countPDFsInDir returns 0 if directory doesn't exist or can't be read (graceful degradation)
- Error count displayed in two locations: badge next to inbox name, and count in error path section
- Keep original InboxCard template for HTMX responses (CreateInbox, ToggleInbox return cards without counts)

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
- Import cycle detected when trying to define InboxWithErrorCount in handler package - resolved by moving type to template package

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- Error count badges ready for use
- Future enhancement: Link to filebrowser for error directory browsing

---
*Phase: 15-pending-fixes*
*Completed: 2026-02-04*
