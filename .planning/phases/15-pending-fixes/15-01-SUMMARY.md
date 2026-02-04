---
phase: 15-pending-fixes
plan: 01
subsystem: ui
tags: [templ, javascript, htmx, uuid]

# Dependency graph
requires:
  - phase: 05-organization
    provides: Tag and correspondent management pages
provides:
  - Working edit buttons on Tags and Correspondents admin pages
affects: []

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "templ.JSFuncCall for onclick handlers with Go data"

key-files:
  created: []
  modified:
    - templates/pages/admin/tags.templ
    - templates/pages/admin/correspondents.templ

key-decisions:
  - "Use templ.JSFuncCall instead of script blocks for onclick handlers"
  - "Call .String() on uuid.UUID to serialize as string instead of byte array"
  - "Add safeNotes helper for nullable string handling"

patterns-established:
  - "templ.JSFuncCall pattern: templ.JSFuncCall('funcName', arg1.String(), arg2, helperFunc(arg3))"

# Metrics
duration: 3 min
completed: 2026-02-04
---

# Phase 15 Plan 01: Edit Button Fix Summary

**Fixed edit buttons on Tags and Correspondents pages using templ.JSFuncCall to properly serialize Go data for JavaScript**

## Performance

- **Duration:** 3 min
- **Started:** 2026-02-04T15:04:10Z
- **Completed:** 2026-02-04T15:07:00Z
- **Tasks:** 2
- **Files modified:** 2

## Accomplishments

- Fixed Tags page edit button that was failing silently
- Fixed Correspondents page edit button that was failing silently
- Replaced broken templ script blocks with templ.JSFuncCall
- Added safeNotes helper function for nullable string handling

## Task Commits

Each task was committed atomically:

1. **Task 1: Fix tags edit button using templ.JSFuncCall** - `c34129e` (fix)
2. **Task 2: Fix correspondents edit button using templ.JSFuncCall** - `97ab9fd` (fix)

## Files Created/Modified

- `templates/pages/admin/tags.templ` - Replaced editTagOnClick script block with templ.JSFuncCall in onclick attribute
- `templates/pages/admin/correspondents.templ` - Replaced editCorrespondentOnClick script block with templ.JSFuncCall, added safeNotes helper

## Decisions Made

1. **Use templ.JSFuncCall over script blocks** - templ.JSFuncCall properly JSON-encodes each argument individually, avoiding the JSON casing issue where Go struct fields get lowercase keys in JSON serialization
2. **Call .String() on uuid.UUID** - UUID types serialize as byte arrays when passed through templ script blocks; calling .String() explicitly produces the expected hyphenated UUID string
3. **Add helper functions for nullable fields** - safeNotes helper handles nil *string pointers gracefully, returning empty string when nil

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Edit buttons now work correctly on Tags and Correspondents pages
- Pattern established for future templ onclick handlers with Go data
- Ready for remaining Phase 15 plans (inbox error links, processing progress visibility)

---
*Phase: 15-pending-fixes*
*Completed: 2026-02-04*
