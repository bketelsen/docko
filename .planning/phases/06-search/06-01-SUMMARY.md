---
phase: 06-search
plan: 01
subsystem: database
tags: [postgresql, full-text-search, tsvector, gin-index, sqlc]

# Dependency graph
requires:
  - phase: 05-organization
    provides: correspondents table, document_correspondents join, tags/document_tags
provides:
  - search_vector generated column on documents
  - GIN index for fast full-text search
  - SearchDocuments query with optional filters
  - CountSearchDocuments for pagination totals
affects: [06-02 search-ui, 06-03 search-api]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Generated STORED tsvector column for auto-updating search vector"
    - "Boolean flag pattern for optional sqlc filters"
    - "websearch_to_tsquery for safe user input handling"

key-files:
  created:
    - internal/database/migrations/007_search_vector.sql
  modified:
    - sqlc/queries/documents.sql

key-decisions:
  - "Generated STORED column instead of trigger for automatic search_vector updates"
  - "websearch_to_tsquery for safe user input (no syntax errors on malformed queries)"
  - "Boolean flag pattern for optional filters (sqlc limitation workaround)"
  - "Tag filter uses AND logic (must have ALL selected tags, not any)"

patterns-established:
  - "Boolean flag pattern: has_X + X_value for optional WHERE conditions"
  - "ts_headline only computed when query provided (performance optimization)"

# Metrics
duration: 3min
completed: 2026-02-03
---

# Phase 06 Plan 01: Search Infrastructure Summary

**PostgreSQL full-text search with generated tsvector column, GIN index, and SearchDocuments query supporting optional filters for query, correspondent, tags, and date range**

## Performance

- **Duration:** 3 min
- **Started:** 2026-02-03T16:34:24Z
- **Completed:** 2026-02-03T16:38:20Z
- **Tasks:** 2
- **Files modified:** 2

## Accomplishments

- Added search_vector generated column that auto-updates when original_filename or text_content changes
- Created GIN index for fast full-text search queries
- SearchDocuments query with optional filters: full-text query, correspondent, date range, tags (AND logic)
- CountSearchDocuments query for pagination totals
- Returns rank and headline snippet when search query provided

## Task Commits

Each task was committed atomically:

1. **Task 1: Add search_vector column and GIN index migration** - `55a0617` (feat)
2. **Task 2: Create SearchDocuments query with filters** - `b7e267f` (feat)

## Files Created/Modified

- `internal/database/migrations/007_search_vector.sql` - Migration adding search_vector column and GIN index
- `sqlc/queries/documents.sql` - Added SearchDocuments and CountSearchDocuments queries

## Decisions Made

- **Generated STORED column:** Automatically updates search_vector when document content changes, no trigger needed
- **websearch_to_tsquery:** Handles user input safely, no syntax errors on malformed queries (vs plainto_tsquery or to_tsquery)
- **Boolean flag pattern:** sqlc doesn't support dynamic WHERE, so we use `has_X` boolean + value pairs for optional filters
- **Tag AND logic:** Tag filter requires ALL selected tags (using HAVING COUNT = tag_count), not ANY tag

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

- Database was not running initially - started PostgreSQL via `docker compose up -d`
- sqlc generate failed during air rebuild because database schema was stale - ran sqlc manually after migration applied

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Search infrastructure complete
- Ready for search UI (06-02) and API handler (06-03)
- SearchDocuments returns all data needed for search results display (rank, headline, correspondent)

---
*Phase: 06-search*
*Completed: 2026-02-03*
