---
phase: 08-ai-integration
plan: 01
subsystem: database
tags: [postgres, sqlc, enums, ai-settings, ai-suggestions, ai-usage]

# Dependency graph
requires:
  - phase: 03-processing
    provides: jobs table for queue integration
  - phase: 02-ingestion
    provides: documents table for foreign key references
provides:
  - ai_settings table for global AI configuration
  - ai_suggestions table for per-document suggestions with confidence scores
  - ai_usage table for token tracking and cost monitoring
  - sqlc queries for all AI CRUD operations
affects: [08-02, 08-03, 08-04, 08-05, 08-06]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - Singleton table pattern (id=1 CHECK constraint)
    - Enum types for suggestion_status and suggestion_type
    - Partial index for pending suggestions

key-files:
  created:
    - internal/database/migrations/009_ai_integration.sql
    - sqlc/queries/ai.sql
  modified: []

key-decisions:
  - "Singleton pattern for ai_settings with CHECK (id = 1)"
  - "Dual threshold system: auto_apply (0.85) and review (0.50)"
  - "Partial index on pending status for efficient review queue queries"
  - "DECIMAL(3,2) for confidence scores (0.00 to 1.00)"

patterns-established:
  - "Suggestion workflow: pending -> accepted/rejected/auto_applied"
  - "Usage tracking per AI request for cost monitoring"

# Metrics
duration: 2min
completed: 2026-02-03
---

# Phase 8 Plan 1: AI Database Schema Summary

**AI integration database layer with settings singleton, suggestion workflow tables, and usage tracking for cost monitoring**

## Performance

- **Duration:** 2 min
- **Started:** 2026-02-03T19:40:36Z
- **Completed:** 2026-02-03T19:42:00Z
- **Tasks:** 2
- **Files modified:** 2

## Accomplishments
- Created ai_settings singleton table for global AI configuration (provider, max pages, thresholds)
- Created ai_suggestions table with confidence scores, status workflow, and document links
- Created ai_usage table for tracking tokens per request for cost monitoring
- Generated all sqlc queries including settings CRUD, suggestion workflow, and usage stats

## Task Commits

Each task was committed atomically:

1. **Task 1: Create AI integration database schema** - `6671bcd` (feat)
2. **Task 2: Create sqlc queries for AI integration** - `9f7d2df` (feat)

## Files Created/Modified
- `internal/database/migrations/009_ai_integration.sql` - AI schema with settings, suggestions, and usage tables
- `sqlc/queries/ai.sql` - 15 queries for settings, suggestions, and usage operations
- `internal/database/sqlc/ai.sql.go` - Generated Go code (14KB)
- `internal/database/sqlc/models.go` - Generated model structs and enum types

## Decisions Made
- Used singleton pattern with CHECK (id = 1) for ai_settings to enforce single row
- Dual confidence thresholds: auto_apply (0.85) for automatic application, review (0.50) for showing in UI
- Partial index on status WHERE pending for efficient review queue queries
- DECIMAL(3,2) type for confidence scores (0.00 to 1.00 range)
- Optional job_id with ON DELETE SET NULL to preserve suggestions when jobs are cleaned up
- resolved_at and resolved_by fields for audit trail on suggestion workflow

## Deviations from Plan
None - plan executed exactly as written.

## Issues Encountered
None

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Database schema complete and ready for AI service layer (08-02)
- sqlc queries available for settings management, suggestion CRUD, and usage tracking
- Enums (SuggestionStatus, SuggestionType) generated for type-safe Go code

---
*Phase: 08-ai-integration*
*Completed: 2026-02-03*
