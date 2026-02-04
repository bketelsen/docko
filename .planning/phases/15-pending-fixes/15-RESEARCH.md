# Phase 15: Pending Fixes - Research

**Researched:** 2026-02-04
**Domain:** Bug fixes, UI improvements
**Confidence:** HIGH

## Summary

Phase 15 addresses four accumulated issues from the pending todos:
1. **Tags/Correspondents edit buttons** - Both use templ's script directive incorrectly, causing onclick handlers to fail silently
2. **Inbox error directory links** - Users have no visibility into error files or way to browse them
3. **Processing progress visibility** - Jobs show "processing" status but no indication of current step or stuck detection

The edit button fix is a straightforward templ syntax correction. The inbox error feature needs UI additions and optional file counting. The processing progress feature requires either database schema changes or SSE enhancements.

**Primary recommendation:** Fix edit buttons using `templ.JSFuncCall` instead of script blocks, add inline error counts to inbox cards, and add a `current_step` column to the jobs table with SSE broadcast for real-time visibility.

## Standard Stack

No new libraries needed - this phase uses existing patterns.

### Core (Already in Project)
| Library | Version | Purpose | Used For |
|---------|---------|---------|----------|
| templ | v0.3.977 | HTML templating | Template fixes |
| HTMX | Latest | Dynamic updates | Inbox error UI |
| SSE | N/A | Server-sent events | Progress broadcast |
| PostgreSQL | 16 | Database | Schema migration |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| os | stdlib | File operations | Counting error files |
| filepath | stdlib | Path operations | Error directory resolution |

## Architecture Patterns

### Pattern 1: templ Script vs JSFuncCall

**What:** templ provides two ways to pass Go data to JavaScript: `script` blocks and `templ.JSFuncCall`

**Problem with current implementation:**
```templ
// BROKEN - script block serializes entire struct
script editTagOnClick(tag sqlc.ListTagsWithCountsRow) {
    openEditMode(tag.ID, tag.Name, tag.Color || 'blue');
}
```

The generated JavaScript accesses `tag.ID`, but JSON serialization produces `tag.id` (lowercase). Additionally, the `tag.Color` is a `*string` which complicates the `||` fallback.

**Solution - Use templ.JSFuncCall:**
```templ
// CORRECT - passes individual values, properly JSON-encoded
@button.Button(button.Props{
    Attributes: templ.Attributes{
        "onclick": templ.JSFuncCall("openEditMode",
            tag.ID.String(),
            tag.Name,
            safeColor(tag.Color)),
    },
})
```

This approach:
- Passes each value separately with proper JSON encoding
- Converts UUID to string explicitly
- Handles nil pointers in Go before passing to JS

**Source:** Context7 templ documentation - `templ.JSFuncCall` passes JSON-encoded server-side Go arguments directly to client-side functions.

### Pattern 2: Error Count Badge on Inbox Cards

**What:** Show count of files in error directory directly on inbox cards

**Implementation:**
```go
// In inbox handler, add error count
type InboxWithErrorCount struct {
    sqlc.Inbox
    ErrorCount int
}

func countErrorFiles(errorPath string) int {
    entries, err := os.ReadDir(errorPath)
    if err != nil {
        return 0
    }
    count := 0
    for _, e := range entries {
        if !e.IsDir() && strings.HasSuffix(e.Name(), ".pdf") {
            count++
        }
    }
    return count
}
```

**Template addition:**
```templ
if errorCount > 0 {
    <span class="badge bg-destructive">
        { strconv.Itoa(errorCount) } error(s)
    </span>
}
```

### Pattern 3: Processing Step Tracking

**What:** Track current processing step in jobs table and broadcast via SSE

**Database migration:**
```sql
-- +goose Up
ALTER TABLE jobs ADD COLUMN current_step VARCHAR(50);
-- Values: 'starting', 'extracting_text', 'running_ocr', 'generating_thumbnail', 'finalizing'

-- +goose Down
ALTER TABLE jobs DROP COLUMN IF EXISTS current_step;
```

**Processor updates (internal/processing/processor.go):**
```go
// Before each step, update job's current_step
p.db.Queries.UpdateJobStep(ctx, sqlc.UpdateJobStepParams{
    ID:          job.ID,
    CurrentStep: "extracting_text",
})
p.broadcast(StatusUpdate{
    DocumentID:  docID,
    Status:      StatusProcessing,
    CurrentStep: "extracting_text",
    QueueName:   document.QueueDefault,
})
```

**StatusUpdate struct enhancement:**
```go
type StatusUpdate struct {
    DocumentID  uuid.UUID
    Status      string
    CurrentStep string  // NEW: Current processing step
    Error       string
    QueueName   string
}
```

### Pattern 4: Stuck Job Detection

**What:** Identify jobs that have been processing for too long

**SQL query:**
```sql
-- name: GetStuckJobs :many
SELECT * FROM jobs
WHERE status = 'processing'
  AND started_at < NOW() - INTERVAL '5 minutes'
ORDER BY started_at ASC;
```

**UI indicator:**
```templ
if job.Status == "processing" && isStuck(job.StartedAt) {
    @badge.Badge(badge.Props{Variant: badge.VariantDestructive}) {
        Stuck
    }
}
```

### Anti-Patterns to Avoid

- **Don't use templ `script` blocks for onclick handlers** - They serialize entire structs which causes JSON casing issues and makes nil handling complex
- **Don't poll for error file counts** - Count on page load only, not continuously
- **Don't store processing step only in memory** - If server restarts, state is lost; persist to database

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Passing Go data to JS events | Script blocks with struct access | `templ.JSFuncCall` with individual values | Proper JSON encoding, explicit types |
| File browser in app | Custom file listing modal | Link to filesystem path or defer to external filebrowser | Out of scope, adds complexity |
| Real-time progress polling | JavaScript setInterval polling | SSE with existing StatusBroadcaster | Already have SSE infrastructure |

**Key insight:** The existing SSE infrastructure for document status updates can be extended to include step information without new architectural patterns.

## Common Pitfalls

### Pitfall 1: templ Script Block JSON Casing
**What goes wrong:** JavaScript code references `tag.ID` but JSON serialization produces `tag.id` (Go struct tags use lowercase by default)
**Why it happens:** templ script blocks JSON-encode the entire Go struct; Go json tags default to lowercase
**How to avoid:** Use `templ.JSFuncCall` with individual primitive values, or access properties using lowercase JSON keys
**Warning signs:** onclick handlers do nothing, no JavaScript console errors

### Pitfall 2: UUID Serialization
**What goes wrong:** UUID types don't serialize to simple strings in script blocks
**Why it happens:** `uuid.UUID` is a `[16]byte` array that serializes as JSON array, not string
**How to avoid:** Always call `.String()` before passing UUID to JavaScript
**Warning signs:** JavaScript sees array like `[1,2,3,...]` instead of `"abc-123-..."`

### Pitfall 3: Nil Pointer Handling in JS
**What goes wrong:** Go `*string` values become `null` in JavaScript, causing `|| 'default'` to work but property access to fail
**Why it happens:** Go nil pointer serializes to JSON null
**How to avoid:** Handle nil in Go before passing to JS: `safeColor(tag.Color)`
**Warning signs:** `Cannot read property of null` errors in console

### Pitfall 4: Error Path Resolution
**What goes wrong:** Incorrect error path construction if custom `error_path` is set vs default
**Why it happens:** Two sources: inbox.ErrorPath (nullable) or default `{inbox.Path}/errors`
**How to avoid:** Reuse existing `getErrorPath` function from inbox service
**Warning signs:** Error counts show 0 when files exist

### Pitfall 5: Race Condition in Step Updates
**What goes wrong:** Job step column updated but not read before job completes
**Why it happens:** Processing is fast, UI polls too slowly
**How to avoid:** Broadcast step changes via SSE immediately when updated
**Warning signs:** UI always shows "processing" without step details

## Code Examples

### Fix 1: Tags Edit Button (templ.JSFuncCall)

```templ
// templates/pages/admin/tags.templ

// Remove the script block entirely (lines 328-330)
// - script editTagOnClick(tag sqlc.ListTagsWithCountsRow) { ... }

// Replace onclick attribute in TagCard template
@button.Button(button.Props{
    Variant: button.VariantGhost,
    Size:    button.SizeIcon,
    Attributes: templ.Attributes{
        "onclick": templ.JSFuncCall("openEditMode",
            tag.ID.String(),
            tag.Name,
            safeColor(tag.Color)),
        "title": "Edit tag",
    },
}) {
    // SVG icon unchanged
}

// safeColor helper function already exists in tags.templ (line 306-311)
```

### Fix 2: Correspondents Edit Button (templ.JSFuncCall)

```templ
// templates/pages/admin/correspondents.templ

// Remove the script block entirely (lines 491-493)
// - script editCorrespondentOnClick(correspondent ...) { ... }

// Replace onclick attribute in CorrespondentCard template
@button.Button(button.Props{
    Variant: button.VariantGhost,
    Size:    button.SizeIcon,
    Attributes: templ.Attributes{
        "onclick": templ.JSFuncCall("openEditMode",
            correspondent.ID.String(),
            correspondent.Name,
            safeNotes(correspondent.Notes)),
        "title": "Edit correspondent",
    },
}) {
    // SVG icon unchanged
}

// Add helper function
func safeNotes(notes *string) string {
    if notes == nil {
        return ""
    }
    return *notes
}
```

### Fix 3: Error Count in Handler

```go
// internal/handler/inboxes.go

type InboxWithErrorCount struct {
    Inbox      sqlc.Inbox
    ErrorCount int
}

func (h *Handler) Inboxes(c echo.Context) error {
    inboxes, err := h.db.Queries.ListInboxes(c.Request().Context())
    if err != nil {
        return err
    }

    // Add error counts
    inboxesWithCounts := make([]InboxWithErrorCount, len(inboxes))
    for i, inbox := range inboxes {
        errorPath := h.resolveErrorPath(inbox)
        inboxesWithCounts[i] = InboxWithErrorCount{
            Inbox:      inbox,
            ErrorCount: countPDFsInDir(errorPath),
        }
    }

    return admin.InboxesWithCounts(inboxesWithCounts).Render(
        c.Request().Context(), c.Response().Writer)
}

func countPDFsInDir(dir string) int {
    entries, err := os.ReadDir(dir)
    if err != nil {
        return 0
    }
    count := 0
    for _, e := range entries {
        if !e.IsDir() && strings.HasSuffix(strings.ToLower(e.Name()), ".pdf") {
            count++
        }
    }
    return count
}
```

### Fix 4: Processing Step Migration

```sql
-- internal/database/migrations/012_job_current_step.sql
-- +goose Up
ALTER TABLE jobs ADD COLUMN current_step VARCHAR(50);

-- +goose Down
ALTER TABLE jobs DROP COLUMN IF EXISTS current_step;
```

### Fix 5: Processing Step Updates

```go
// internal/processing/processor.go

// Define step constants
const (
    StepStarting            = "starting"
    StepExtractingText      = "extracting_text"
    StepRunningOCR          = "running_ocr"
    StepGeneratingThumbnail = "generating_thumbnail"
    StepFinalizing          = "finalizing"
)

func (p *Processor) HandleJob(ctx context.Context, job *sqlc.Job) error {
    // ... existing setup ...

    // Update step: starting
    p.updateStep(ctx, job.ID, docID, StepStarting)

    // Extract text
    p.updateStep(ctx, job.ID, docID, StepExtractingText)
    text, method, err := p.textExt.Extract(ctx, pdfPath)
    // ... error handling ...

    // If OCR was needed (check method)
    if method == "ocr" {
        // Step was already broadcast during OCR
    }

    // Generate thumbnail
    p.updateStep(ctx, job.ID, docID, StepGeneratingThumbnail)
    thumbPath, err := p.thumbGen.Generate(ctx, pdfPath, docID)
    // ... error handling ...

    // Finalize
    p.updateStep(ctx, job.ID, docID, StepFinalizing)
    // ... transaction commit ...
}

func (p *Processor) updateStep(ctx context.Context, jobID, docID uuid.UUID, step string) {
    // Update database
    p.db.Queries.UpdateJobStep(ctx, sqlc.UpdateJobStepParams{
        ID:          jobID,
        CurrentStep: &step,
    })

    // Broadcast to SSE subscribers
    p.broadcast(StatusUpdate{
        DocumentID:  docID,
        Status:      StatusProcessing,
        CurrentStep: step,
        QueueName:   document.QueueDefault,
    })
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| templ script blocks for complex data | templ.JSFuncCall for onclick | templ v0.2+ | Simpler, safer data passing |
| Opaque "processing" status | Step-by-step progress | This phase | Better user visibility |

**Not deprecated but discouraged:**
- templ script blocks: Still work but error-prone for complex types; prefer JSFuncCall for event handlers

## Open Questions

1. **Filebrowser Integration**
   - What we know: Pending todo mentions filebrowser links, but no filebrowser service is deployed
   - What's unclear: Should we add filebrowser service, or just show file paths?
   - Recommendation: For Phase 15, show error count badge and file path only. Filebrowser integration is a separate feature.

2. **Error File Retry Mechanism**
   - What we know: Files in error directory are failed imports
   - What's unclear: Should there be a "Retry All Errors" button?
   - Recommendation: Out of scope for Phase 15. Document as future enhancement.

3. **Stuck Job Threshold**
   - What we know: 5 minutes mentioned in pending todo
   - What's unclear: Is 5 minutes appropriate for all document sizes?
   - Recommendation: Make threshold configurable (default 5 min), flag in UI with warning

## Sources

### Primary (HIGH confidence)
- Context7 /a-h/templ - Script templates, JSFuncCall documentation
- Project codebase analysis (tags.templ, correspondents.templ, processor.go, status.go)

### Secondary (MEDIUM confidence)
- Pending todo files with problem statements and potential solutions

### Tertiary (LOW confidence)
- None

## Metadata

**Confidence breakdown:**
- Edit button fix: HIGH - Clear root cause (JSON casing), verified solution pattern
- Error count UI: HIGH - Standard file operations, existing UI patterns
- Processing progress: MEDIUM - Multiple implementation options, may need tuning
- Stuck detection: MEDIUM - Threshold needs validation

**Research date:** 2026-02-04
**Valid until:** 2026-03-04 (30 days - stable domain)
