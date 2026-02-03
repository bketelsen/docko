# Phase 9: Minimum Number of Words Import Block - Research

**Researched:** 2026-02-03
**Domain:** Document validation and configurable settings in Go/PostgreSQL
**Confidence:** HIGH

## Summary

This phase adds a configurable minimum word count threshold that blocks document import when extracted text falls below the threshold. The feature is straightforward: the existing codebase has established patterns for singleton settings tables (ai_settings), processing pipeline with status handling (processing.Processor), and user feedback via HTMX toasts. The implementation integrates naturally into the existing text extraction step.

The research focused on three areas: (1) where to inject the word count check in the processing pipeline, (2) how to store/manage the threshold setting following existing patterns, and (3) how to communicate rejections to users via the established UI patterns.

**Primary recommendation:** Add word count validation after text extraction in `processing.Processor.HandleJob()`, store threshold in existing `ai_settings` table (or new singleton table), and quarantine documents with insufficient words using the existing quarantine mechanism with clear error messaging.

## Standard Stack

This phase requires no new libraries - all functionality exists in Go standard library and existing codebase patterns.

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `strings` | stdlib | `strings.Fields()` for word counting | Simple, handles all Unicode whitespace, no external deps |
| `sqlc` | existing | Type-safe queries for settings | Already used throughout codebase |
| `goose` | existing | Database migrations | Already used for schema changes |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `templ` | existing | Settings UI template | Already used for all admin pages |
| `htmx` | existing | Toast notifications | Already used for user feedback |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| `strings.Fields()` | regex `[\S]+` | Regex is slower and unnecessary; Fields handles whitespace correctly |
| New settings table | Extend ai_settings | Adding column to ai_settings keeps settings consolidated |

**Installation:**
```bash
# No new dependencies required
```

## Architecture Patterns

### Where Word Count Check Fits

The check should occur AFTER text extraction succeeds but BEFORE the document is marked as completed:

```
HandleJob()
  ├── Get document
  ├── Extract text (existing)
  ├── [NEW] Check word count against threshold
  │   └── If below threshold → quarantine with reason
  ├── Generate thumbnail (existing)
  └── Update document as completed
```

### Recommended Approach

**Option A (Recommended): Add to existing ai_settings table**

The ai_settings table already follows the singleton pattern with `CHECK(id=1)`. Adding `min_word_count INTEGER NOT NULL DEFAULT 0` keeps all document processing settings in one place.

Rationale:
- AI settings and word count both affect document processing decisions
- Single settings page for all processing-related configuration
- 0 = disabled (no check), consistent with other "off" defaults
- Avoids creating yet another singleton settings table

**Option B: Create dedicated import_settings table**

More separation of concerns but adds another table and UI section for a single field.

### Pattern: Singleton Settings Table

From existing `ai_settings` implementation:

```sql
-- Singleton pattern with CHECK constraint
CREATE TABLE ai_settings (
    id INTEGER PRIMARY KEY DEFAULT 1 CHECK (id = 1),
    -- ... columns
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Always insert default row
INSERT INTO ai_settings (id) VALUES (1);
```

### Pattern: Quarantine with Reason

From existing `processor.go`:

```go
// quarantine moves a document to failed status after repeated failures
func (p *Processor) quarantine(ctx context.Context, docID uuid.UUID, reason string) error {
    // Updates status to failed, logs event, broadcasts SSE
    // Returns nil so job completes (failure handled gracefully)
}
```

The quarantine pattern is perfect for word count rejection:
- Document stored (originals preserved)
- Clear error message shown to user
- Processing marked as failed, not stuck in pending
- User can see what happened via document detail page

### Anti-Patterns to Avoid

- **Blocking at ingestion time:** Don't check word count during upload - text extraction happens asynchronously. User would have no feedback.

- **Deleting documents that fail check:** The file is already stored. Quarantine preserves it for manual review.

- **Hard-coded threshold:** Must be configurable. Some users want strict filtering, others want everything.

- **Character count instead of word count:** Words are more intuitive for users (a 50-character document could have 10 words or 1 long word).

## Don't Hand-Roll

Problems that look simple but have existing solutions:

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Word counting | Custom tokenizer | `strings.Fields()` | Handles all Unicode whitespace correctly |
| Settings singleton | Manual id management | CHECK(id=1) constraint | Database enforces single row |
| Error display | Custom modal | Existing quarantine + toast | Consistent UX with existing failures |
| Status tracking | New status enum | Existing `processing_status = 'failed'` | Failed is the right status for rejected docs |

**Key insight:** This feature is validation, not new functionality. It should use existing patterns for settings, status handling, and user feedback. No new infrastructure needed.

## Common Pitfalls

### Pitfall 1: Checking at Wrong Pipeline Stage

**What goes wrong:** Checking word count during upload means no text exists yet (extraction is async). Checking after thumbnail generation means wasted work.
**Why it happens:** Not understanding the async processing flow
**How to avoid:** Check immediately after text extraction, before thumbnail generation
**Warning signs:** Looking for places to add the check in upload.go or inbox/service.go

### Pitfall 2: Empty vs Zero Word Count

**What goes wrong:** Different handling of empty string vs whitespace-only text vs text with 1-2 words
**Why it happens:** Edge cases not considered
**How to avoid:** `strings.Fields()` returns empty slice for empty/whitespace-only strings. Check `len(words) < threshold` covers all cases.
**Warning signs:** Special-casing for empty strings separately from word count

### Pitfall 3: Threshold of 0 Semantics

**What goes wrong:** Unclear if 0 means "no minimum" or "reject everything"
**Why it happens:** Not defining the semantics clearly
**How to avoid:** Define 0 as "disabled" (skip check entirely). Document this in UI help text.
**Warning signs:** Treating threshold=0 as a valid minimum to check against

### Pitfall 4: No User Feedback on Rejection

**What goes wrong:** Document silently fails, user doesn't know why
**Why it happens:** Forgetting to set clear processing_error message
**How to avoid:** Set detailed error: "Rejected: document contains X words (minimum required: Y)"
**Warning signs:** Generic "processing failed" errors without word count specifics

### Pitfall 5: Forgetting Inbox/Network Source Documents

**What goes wrong:** Feature only works for web uploads, inbox-imported docs bypass check
**Why it happens:** Testing only the upload flow
**How to avoid:** The check is in Processor.HandleJob, which ALL documents go through regardless of source
**Warning signs:** Adding the check to upload handler instead of processor

## Code Examples

### Word Counting (Go stdlib)

```go
// Source: https://gophersnippets.com/how-to-count-the-number-of-words-in-a-string
import "strings"

func countWords(text string) int {
    return len(strings.Fields(text))
}

// strings.Fields splits on any whitespace (space, tab, newline, etc.)
// Returns empty slice for empty or whitespace-only strings
```

### Threshold Check in Processor

```go
// After text extraction, before thumbnail generation
// Reference: existing processor.go HandleJob pattern

text, method, err := p.textExt.Extract(ctx, pdfPath)
if err != nil {
    // existing error handling
}

// NEW: Check word count threshold
wordCount := len(strings.Fields(text))
settings, _ := p.db.Queries.GetAISettings(ctx) // or new GetImportSettings
if settings.MinWordCount > 0 && wordCount < int(settings.MinWordCount) {
    reason := fmt.Sprintf("document has %d words (minimum required: %d)",
        wordCount, settings.MinWordCount)
    return p.quarantine(ctx, docID, reason)
}

// Continue with thumbnail generation...
```

### Migration Pattern (from ai_settings)

```sql
-- +goose Up
ALTER TABLE ai_settings
    ADD COLUMN min_word_count INTEGER NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE ai_settings
    DROP COLUMN IF EXISTS min_word_count;
```

### Settings Query Update

```sql
-- name: GetAISettings :one
SELECT * FROM ai_settings WHERE id = 1;

-- name: UpdateAISettings :one (add min_word_count parameter)
UPDATE ai_settings SET
    preferred_provider = $1,
    max_pages = $2,
    auto_process = $3,
    auto_apply_threshold = $4,
    review_threshold = $5,
    min_word_count = $6,  -- NEW
    updated_at = NOW()
WHERE id = 1
RETURNING *;
```

### UI Settings Form Addition

```templ
<!-- Add to existing AISettings template -->
<div>
    <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
        Minimum Word Count
    </label>
    <input
        type="number"
        name="min_word_count"
        min="0"
        max="10000"
        value={ strconv.Itoa(int(settings.MinWordCount)) }
        class="w-32 rounded-lg border-gray-300 dark:border-gray-600 dark:bg-gray-700 dark:text-gray-100"
    />
    <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
        Documents with fewer words are rejected. Set to 0 to disable.
    </p>
</div>
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| N/A | This is a new feature | - | - |

**Deprecated/outdated:**
- None - this is a new feature built on established patterns

## Open Questions

1. **Should rejected documents be listed differently?**
   - What we know: Documents are marked as `processing_status = 'failed'` with specific error message
   - What's unclear: Should there be a filter/view specifically for "insufficient text" rejections?
   - Recommendation: For v1, use existing failed status. Add specific filters in Phase 12 (Queues Detail) if needed.

2. **Rename ai_settings to processing_settings?**
   - What we know: Adding non-AI settings to ai_settings table may confuse the naming
   - What's unclear: Is renaming worth a migration?
   - Recommendation: Keep as ai_settings for v1. Both features relate to document processing. Rename can be done later if scope grows.

3. **Per-inbox threshold overrides?**
   - What we know: Different sources might need different thresholds (e.g., image-heavy PDFs from one source)
   - What's unclear: Is this needed now?
   - Recommendation: Defer to future phase. Single global threshold covers 90% of use cases.

## Sources

### Primary (HIGH confidence)
- Existing codebase: `internal/processing/processor.go` - processing pipeline
- Existing codebase: `internal/database/migrations/009_ai_integration.sql` - singleton settings pattern
- Existing codebase: `internal/handler/ai.go` - settings UI handler pattern
- Existing codebase: `templates/pages/admin/ai_settings.templ` - settings form pattern

### Secondary (MEDIUM confidence)
- [GopherSnippets - Word counting](https://gophersnippets.com/how-to-count-the-number-of-words-in-a-string) - strings.Fields pattern
- [Sling Academy - Word counting in Go](https://www.slingacademy.com/article/how-to-count-words-and-characters-in-a-string-in-go/) - validation of approach

### Tertiary (LOW confidence)
- None

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - no new libraries, all existing patterns
- Architecture: HIGH - clear integration point in existing processor
- Pitfalls: HIGH - straightforward feature with known edge cases

**Research date:** 2026-02-03
**Valid until:** 90 days (stable patterns, no external dependencies)
