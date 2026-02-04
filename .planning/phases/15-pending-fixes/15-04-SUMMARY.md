---
phase: 15-pending-fixes
plan: 04
subsystem: ui
tags: [templ, javascript, onclick, modal]

# Dependency graph
requires:
  - phase: 15-pending-fixes
    provides: JSFuncCall pattern for onclick handlers established in 15-01
provides:
  - Working edit buttons on Tags and Correspondents pages
affects: []

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "templ.JSFuncCall().Call for onclick in templ.Attributes maps"

key-files:
  created: []
  modified:
    - templates/pages/admin/tags.templ
    - templates/pages/admin/correspondents.templ

key-decisions:
  - "templ.JSFuncCall().Call required to get string value for onclick (not ComponentScript struct)"

patterns-established:
  - "When using templ.JSFuncCall in templ.Attributes map, always use .Call to get the escaped JS string"

# Metrics
duration: 1min
completed: 2026-02-04
---

# Phase 15 Plan 04: Fix Edit Button onclick Handlers Summary

**Fixed Tags and Correspondents edit buttons by appending .Call to templ.JSFuncCall() to return escaped JS string instead of ComponentScript struct**

## Performance

- **Duration:** 1 min
- **Started:** 2026-02-04T15:42:52Z
- **Completed:** 2026-02-04T15:43:38Z
- **Tasks:** 1
- **Files modified:** 2

## Accomplishments
- Fixed Tags page Edit button to open modal with tag name/color pre-filled
- Fixed Correspondents page Edit button to open modal with name/notes pre-filled
- Documented pattern: always use .Call when passing JSFuncCall to templ.Attributes

## Task Commits

Each task was committed atomically:

1. **Task 1: Fix templ.JSFuncCall onclick handlers** - `2eef5e9` (fix)

## Files Created/Modified
- `templates/pages/admin/tags.templ` - Added .Call to JSFuncCall on line 243
- `templates/pages/admin/correspondents.templ` - Added .Call to JSFuncCall on line 461

## Decisions Made
- templ.JSFuncCall().Call required to get string value for onclick (not ComponentScript struct) - this was documented in the plan and confirmed correct

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- Edit buttons now functional on both Tags and Correspondents pages
- Gap closure plan 15-05 (Inbox error links) ready for execution
- After 15-05, UAT re-run recommended to confirm all fixes

---
*Phase: 15-pending-fixes*
*Completed: 2026-02-04*
