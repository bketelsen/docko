---
phase: 09-minimum-words
plan: 02
subsystem: processing
tags: [validation, word-count, filtering]

dependency-graph:
  requires: ["09-01"]
  provides: ["word-count-validation", "threshold-ui"]
  affects: []

tech-stack:
  added: []
  patterns:
    - "strings.Fields for Unicode-safe word splitting"
    - "Configurable threshold with 0 = disabled"
    - "Non-blocking settings fetch (warn and continue)"

key-files:
  created: []
  modified:
    - internal/processing/processor.go
    - internal/handler/ai.go
    - templates/pages/admin/ai_settings.templ

decisions:
  - id: word-count-after-extraction
    summary: "Check word count AFTER text extraction, BEFORE thumbnail"
    rationale: "Need text to count words; avoid wasting work on rejected docs"

metrics:
  duration: "3 min"
  completed: "2026-02-03"
---

# Phase 09 Plan 02: Word Count Validation Summary

Word count validation in processor with configurable UI threshold

## What Was Built

### Word Count Validation in Processor
Added word count check to `internal/processing/processor.go`:
- Checks minimum word count after text extraction succeeds
- Uses `strings.Fields()` for Unicode-safe word splitting
- Documents below threshold are quarantined with descriptive message
- Threshold of 0 disables the check (all documents pass)
- Settings fetch failure is non-blocking (warns and continues)

### Handler Validation Fix
Updated `internal/handler/ai.go`:
- Added upper bound (10000) to min_word_count validation
- Defaults to 0 if value is invalid or out of range

### AI Settings UI
Added min_word_count input to `templates/pages/admin/ai_settings.templ`:
- Number input with min=0, max=10000 constraints
- Displays current value from database
- Help text explains 0 = disabled behavior

## Technical Details

**Word Count Logic:**
```go
settings, err := p.db.Queries.GetAISettings(ctx)
if err != nil {
    slog.Warn("failed to get ai settings for word count check", "error", err)
    // Continue processing - don't block on settings fetch failure
} else if settings.MinWordCount > 0 {
    wordCount := len(strings.Fields(text))
    if wordCount < int(settings.MinWordCount) {
        reason := fmt.Sprintf("document has %d words (minimum required: %d)",
            wordCount, settings.MinWordCount)
        return p.quarantine(ctx, docID, reason)
    }
}
```

**Processing Order:**
1. Extract text from PDF
2. Check word count against threshold (new)
3. Generate thumbnail
4. Update document record

## Commits

| Hash | Type | Description |
|------|------|-------------|
| d49b244 | feat | Add word count validation to processor |
| 7ac3a8e | fix | Add upper bound validation for min_word_count |
| ad9a19a | feat | Add min_word_count input to AI settings form |

## Deviations from Plan

None - plan executed exactly as written.

## Verification

- [x] `strings.Fields` used for word counting in processor
- [x] Handler parses and validates min_word_count (0-10000)
- [x] Template has min_word_count input field
- [x] Build succeeds with no errors

## Next Phase Readiness

Phase 09 is complete. The minimum word count feature is fully functional:
- Database schema (09-01)
- Validation logic and UI (09-02)

Users can now configure a minimum word count threshold in AI Settings to reject documents with insufficient text content.
