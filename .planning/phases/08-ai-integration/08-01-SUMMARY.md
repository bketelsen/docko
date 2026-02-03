---
phase: 08-ai-integration
plan: 01
subsystem: database
tags: [ai, postgresql, sqlc, suggestions, usage-tracking]

# Dependency graph
requires:
  - phase: 03-processing
    provides: jobs table for job_id foreign key
  - phase: 02-ingestion
    provides: documents table for document_id foreign key
provides:
  - AI settings singleton table for global configuration
  - AI suggestions table with confidence scores and status workflow
  - AI usage tracking table for cost monitoring
  - CRUD queries for AI configuration and suggestions
affects: [08-02-ai-service, 08-03-ai-ui]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - Singleton table pattern with CHECK constraint for global settings
    - Enum types for suggestion status and type
    - Foreign key to jobs table for tracing suggestions to processing jobs

key-files:
  created:
    - internal/database/migrations/009_ai_integration.sql
    - sqlc/queries/ai.sql
  modified: []

key-decisions:
  - "Singleton pattern with CHECK(id=1) for ai_settings table"
  - "Separate suggestion_status enum (pending/accepted/rejected/auto_applied)"
  - "Separate suggestion_type enum (tag/correspondent)"
  - "DECIMAL(3,2) for confidence scores (0.00-1.00 range)"
  - "Nullable job_id with ON DELETE SET NULL for suggestion tracing"
  - "Partial index on status='pending' for efficient pending suggestion queries"

patterns-established:
  - "AI suggestion workflow: pending -> accepted/rejected/auto_applied"
  - "Token tracking per AI request for cost monitoring"

# Metrics
duration: 4min
completed: 2026-02-03
---

# Phase 8 Plan 1: AI Integration Database Schema Summary

**Database schema for AI settings, suggestions, and usage tracking with singleton settings pattern and status workflow**

## Performance

- **Duration:** 4 min
- **Started:** 2026-02-03T19:38:34Z
- **Completed:** 2026-02-03T19:41:53Z
- **Tasks:** 2
- **Files modified:** 2

## Accomplishments

- Created migration 009 with ai_settings, ai_suggestions, and ai_usage tables
- Implemented singleton pattern for global AI settings with CHECK constraint
- Added suggestion status and type enums for workflow management
- Created comprehensive sqlc queries for all CRUD operations

## Task Commits

Each task was committed atomically:

1. **Task 1: Create AI integration database schema** - `6671bcd` (feat)
2. **Task 2: Create sqlc queries for AI integration** - `9f7d2df` (feat)

## Files Created/Modified

- `internal/database/migrations/009_ai_integration.sql` - AI tables and enums
- `sqlc/queries/ai.sql` - CRUD queries for AI integration
- `internal/database/sqlc/ai.sql.go` - Generated Go code (auto-generated)
- `internal/database/sqlc/models.go` - AiSetting, AiSuggestion, AiUsage structs (auto-generated)

## Decisions Made

- Used singleton pattern with `CHECK (id = 1)` for ai_settings to enforce single row
- Chose `DECIMAL(3,2)` for confidence scores to store values like 0.85 (85% confidence)
- Added `is_new` flag to track if suggestion is for a new tag/correspondent not in taxonomy
- Used `resolved_by` VARCHAR(50) to distinguish 'auto' vs 'user' resolution
- Created partial index on `status = 'pending'` for efficient pending suggestions queries

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Database schema ready for AI service implementation
- Queries available for settings management, suggestion creation, and usage tracking
- No blockers for 08-02 (AI Service Implementation)

---
*Phase: 08-ai-integration*
*Completed: 2026-02-03*
