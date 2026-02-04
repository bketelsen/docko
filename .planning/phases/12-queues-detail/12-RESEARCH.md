# Phase 12: Queues Detail - Research

**Researched:** 2026-02-03
**Domain:** Queue monitoring UI with expandable details, SSE live updates, and job management
**Confidence:** HIGH

## Summary

This phase extends the existing `/queues` route to provide drill-down queue details with expandable sections per queue. The codebase already has:
1. A complete queue system (`internal/queue/queue.go`) with job status tracking
2. SSE infrastructure via `StatusBroadcaster` for real-time updates
3. Basic queue dashboard template (`queue_dashboard.templ`) showing stats and failed jobs
4. sqlc queries for job listing, retry, and stats

The implementation requires:
1. New accordion-based UI with lazy-loaded content per queue
2. Additional sqlc queries for queue-filtered jobs with document info
3. New "dismissed" status for cleared failed jobs (migration required)
4. SSE extension for queue activity updates (reusing existing patterns)
5. Confirmation dialog for bulk operations (templUI dialog component exists)

**Primary recommendation:** Use templUI accordion component for queue expanders with HTMX lazy loading (`hx-trigger="click once"`) for expanded content. Reuse existing SSE patterns from `StatusBroadcaster` for live activity updates.

## Standard Stack

### Core (Already in Codebase)

| Library | Version | Purpose | Already Used |
|---------|---------|---------|--------------|
| templUI accordion | v1.4.0 | Collapsible queue sections | Available, not installed |
| templUI collapsible | v1.4.0 | Alternative expander pattern | Available, not installed |
| templUI dialog | v1.4.0 | Confirmation modals | Already installed |
| HTMX SSE extension | included | Real-time activity updates | Already used |
| sqlc | v1.30.0 | Type-safe database queries | Already used |

### Supporting

| Library | Purpose | When to Use |
|---------|---------|-------------|
| templUI badge | Health status indicators | Already installed |
| templUI button | Action buttons | Already installed |
| templUI card | Queue containers | Already installed |
| templUI table | Job listings | Already installed |

### Component Installation Required

```bash
# Add accordion for expandable queue sections
templui add accordion
templui add collapsible  # Alternative, may be simpler for this use case
```

## Architecture Patterns

### Recommended Project Structure

The phase extends existing files, no new directories needed:

```
templates/pages/admin/
├── queue_dashboard.templ      # Existing - will be significantly modified
├── queue_detail_*.templ       # New partials for lazy-loaded content
sqlc/queries/
├── jobs.sql                   # Add new queries for queue-specific listings
internal/handler/
├── ai.go                      # Contains queue handlers - add new endpoints
internal/database/migrations/
├── 011_job_dismissed.sql      # New migration for dismissed status
```

### Pattern 1: Lazy-Loaded Accordion Content

**What:** Queue sections that load detailed job lists only when expanded
**When to use:** User clicks to expand a queue section
**Why:** Avoids loading potentially large job lists for all queues on page load

```html
<!-- Initial collapsed state - just summary stats -->
@accordion.Item() {
    @accordion.Trigger() {
        <div class="flex justify-between w-full">
            <span>default</span>
            <div class="flex gap-2">
                @badge.Badge() { 5 pending }
                @badge.Badge(badge.Props{Variant: badge.VariantDestructive}) { 2 failed }
            </div>
        </div>
    }
    @accordion.Content() {
        <!-- Lazy load on first expand -->
        <div
            hx-get="/queues/default/details"
            hx-trigger="intersect once"
            hx-swap="innerHTML"
        >
            @skeleton.Skeleton(skeleton.Props{Class: "h-40 w-full"}) {}
        </div>
    }
}
```

### Pattern 2: SSE for Queue Activity Updates

**What:** Server-Sent Events to push job completion/failure updates
**When to use:** User has queues page open, jobs complete in background
**Why:** Real-time feedback without polling

The existing `StatusBroadcaster` sends document-level updates. For queue activity, we need:
1. A new broadcaster or extend existing one for queue-level events
2. SSE event naming: `queue-{queueName}` for swap targeting

```go
// Extend StatusUpdate or create QueueActivityUpdate
type QueueActivityUpdate struct {
    QueueName  string
    JobID      uuid.UUID
    DocumentID uuid.UUID  // For linking to document detail
    Status     string     // completed, failed
    Error      string     // if failed
    Timestamp  time.Time
}
```

```html
<!-- SSE listener for queue updates -->
<div
    hx-ext="sse"
    sse-connect="/queues/activity"
    sse-swap="queue-default"
    id="queue-default-activity"
>
    <!-- Activity list updated via SSE -->
</div>
```

### Pattern 3: Confirmation Dialog for Bulk Operations

**What:** Modal dialog before destructive bulk actions
**When to use:** Retry all, Clear all failed jobs
**Why:** Prevents accidental bulk operations

```html
@dialog.Dialog() {
    @dialog.Trigger() {
        @button.Button(button.Props{Variant: button.VariantDestructive}) {
            Clear All Failed
        }
    }
    @dialog.Content() {
        @dialog.Header() {
            @dialog.Title() { Clear Failed Jobs }
            @dialog.Description() { This will dismiss 5 failed jobs. They will remain in the database for audit purposes. }
        }
        @dialog.Footer() {
            @dialog.Close() {
                @button.Button(button.Props{Variant: button.VariantOutline}) { Cancel }
            }
            @dialog.Close() {
                @button.Button(button.Props{
                    Variant: button.VariantDestructive,
                    Attributes: templ.Attributes{
                        "hx-post": "/queues/default/clear-failed",
                        "hx-swap": "none",
                    },
                }) { Clear Jobs }
            }
        }
    }
}
```

### Pattern 4: Document Links in Job Display

**What:** Extract document_id from job payload for clickable links
**When to use:** Displaying failed or completed jobs
**Why:** User needs to navigate to document detail for investigation

Job payloads are JSONB with `document_id`. Need to:
1. Parse payload in Go handler or create a view/query that extracts it
2. Join with documents table for filename display

```sql
-- New query: Get failed jobs with document info
-- name: GetFailedJobsWithDocument :many
SELECT
    j.*,
    d.id as doc_id,
    d.original_filename
FROM jobs j
LEFT JOIN documents d ON (j.payload->>'document_id')::uuid = d.id
WHERE j.queue_name = $1 AND j.status = 'failed'
ORDER BY j.updated_at DESC
LIMIT $2;
```

### Anti-Patterns to Avoid

- **Loading all job details upfront:** Use lazy loading - a queue could have thousands of jobs
- **Custom SSE implementation:** Reuse existing StatusBroadcaster patterns
- **Deleting failed jobs:** Use "dismissed" status to preserve audit trail
- **Polling for updates:** Use SSE for real-time updates

## Don't Hand-Roll

Problems that look simple but have existing solutions:

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Collapsible UI sections | Custom JS/CSS | templUI accordion or collapsible | Accessibility, animations, state management |
| Confirmation modals | Custom modal | templUI dialog | Already used in codebase, consistent UX |
| Real-time updates | WebSocket or polling | SSE via StatusBroadcaster | Already working in codebase |
| Status badges | Custom styled spans | templUI badge | Consistent styling with dashboard |
| Job status icons | Inline SVGs | Existing `jobStatusBadge` templ | Already implemented in queue_dashboard.templ |

**Key insight:** The codebase already has 90% of the patterns needed. This phase is about composition and extension, not building new infrastructure.

## Common Pitfalls

### Pitfall 1: SSE Connection Management for Multiple Sections

**What goes wrong:** Multiple SSE connections opened when expanding queues
**Why it happens:** Each section tries to create its own SSE connection
**How to avoid:** Single SSE connection at page level, dispatch events to correct sections
**Warning signs:** MaxSubscribers (100) warnings in logs, connection exhaustion

### Pitfall 2: Job Payload Parsing in Templates

**What goes wrong:** Attempting to parse JSONB payload in templ templates
**Why it happens:** templ is logic-light, complex parsing belongs in Go
**How to avoid:** Create Go struct with pre-parsed document info, pass to template
**Warning signs:** Complex template logic, error handling in templates

### Pitfall 3: Race Condition on Retry/Clear

**What goes wrong:** UI shows stale data after retry/clear operations
**Why it happens:** HTMX swap doesn't refresh parent container
**How to avoid:** Use `HX-Trigger` header to signal refresh, or return updated section HTML
**Warning signs:** User sees old data, has to manually refresh

### Pitfall 4: Missing Dismissed Status Migration

**What goes wrong:** "Clear" deletes jobs, losing audit trail
**Why it happens:** Context specifies dismissed != deleted, but no db column exists
**How to avoid:** Add migration for `dismissed_at` column or add 'dismissed' to job_status enum
**Warning signs:** Jobs disappearing from database after clear

## Code Examples

Verified patterns from existing codebase:

### Queue Stats Aggregation (Existing)

```go
// From queue_dashboard.templ
func aggregateQueueStats(stats []sqlc.GetQueueStatsRow) map[string]map[string]int64 {
    result := make(map[string]map[string]int64)
    for _, s := range stats {
        if _, ok := result[s.QueueName]; !ok {
            result[s.QueueName] = make(map[string]int64)
        }
        result[s.QueueName][string(s.Status)] = s.Count
    }
    return result
}
```

### SSE Handler Pattern (Existing)

```go
// From internal/handler/status.go
func (h *Handler) ProcessingStatus(c echo.Context) error {
    resp := c.Response()
    resp.Header().Set("Content-Type", "text/event-stream")
    resp.Header().Set("Cache-Control", "no-cache")
    resp.Header().Set("Connection", "keep-alive")
    resp.Header().Set("X-Accel-Buffering", "no")

    ctx := c.Request().Context()
    updates := h.broadcaster.Subscribe(ctx)
    // ... handle updates
}
```

### HTMX Lazy Load Pattern (New)

```html
<!-- Accordion item with lazy-loaded content -->
<div class="border-b">
    <button
        class="flex w-full justify-between py-4 font-medium"
        onclick="this.nextElementSibling.classList.toggle('hidden')"
    >
        Queue: default
    </button>
    <div class="hidden pb-4"
        hx-get="/queues/default/details"
        hx-trigger="intersect once"
        hx-swap="innerHTML">
        <!-- Skeleton loader -->
        <div class="animate-pulse bg-muted h-32 rounded"></div>
    </div>
</div>
```

### Job Retry with Toast Feedback (Existing)

```go
// From internal/handler/ai.go
func (h *Handler) RetryJob(c echo.Context) error {
    ctx := c.Request().Context()
    jobID, err := uuid.Parse(c.Param("id"))
    if err != nil {
        return echo.NewHTTPError(http.StatusBadRequest, "invalid job id")
    }

    _, err = h.db.Queries.ResetJobForRetry(ctx, jobID)
    if err != nil {
        c.Response().Header().Set("HX-Trigger", `{"showToast": {"message": "Failed to retry job", "type": "error"}}`)
        return c.NoContent(http.StatusInternalServerError)
    }

    c.Response().Header().Set("HX-Trigger", `{"showToast": {"message": "Job queued for retry", "type": "success"}}`)
    return c.Redirect(http.StatusSeeOther, "/queues")
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Full page reload | HTMX partial updates | Already in codebase | Faster UX |
| Custom modals | templUI dialog | Already in codebase | Consistent, accessible |
| Polling for updates | SSE | Already in codebase | Real-time, efficient |

**No deprecated patterns identified** - existing codebase is current.

## Required Database Changes

### New Migration: Job Dismissed Status

The CONTEXT.md specifies that "Clear" should mark jobs as dismissed, not delete them. Current `job_status` enum has: `pending`, `processing`, `completed`, `failed`.

**Option A: Add 'dismissed' to enum** (Recommended)
```sql
-- 011_job_dismissed.sql
-- +goose Up
ALTER TYPE job_status ADD VALUE 'dismissed' AFTER 'failed';

-- +goose Down
-- Note: PostgreSQL doesn't support removing enum values
-- Would need to recreate the enum and update all references
```

**Option B: Add dismissed_at column**
```sql
-- +goose Up
ALTER TABLE jobs ADD COLUMN dismissed_at TIMESTAMPTZ;
CREATE INDEX idx_jobs_dismissed ON jobs (dismissed_at) WHERE dismissed_at IS NOT NULL;
```

Recommendation: **Option A** - cleaner, maintains status as single source of truth.

### New Queries Required

```sql
-- Get failed jobs for a specific queue with document info
-- name: GetFailedJobsForQueue :many
SELECT
    j.*,
    d.id as document_id,
    d.original_filename as document_name
FROM jobs j
LEFT JOIN LATERAL (
    SELECT id, original_filename
    FROM documents
    WHERE id = (j.payload->>'document_id')::uuid
) d ON true
WHERE j.queue_name = $1 AND j.status = 'failed'
ORDER BY j.updated_at DESC
LIMIT $2 OFFSET $3;

-- Get recent completed jobs for a queue (last 24h)
-- name: GetRecentCompletedJobsForQueue :many
SELECT
    j.*,
    d.id as document_id,
    d.original_filename as document_name
FROM jobs j
LEFT JOIN LATERAL (
    SELECT id, original_filename
    FROM documents
    WHERE id = (j.payload->>'document_id')::uuid
) d ON true
WHERE j.queue_name = $1
    AND j.status = 'completed'
    AND j.completed_at > NOW() - INTERVAL '24 hours'
ORDER BY j.completed_at DESC
LIMIT $2 OFFSET $3;

-- Clear (dismiss) failed jobs for a queue
-- name: DismissFailedJobsForQueue :exec
UPDATE jobs SET status = 'dismissed', updated_at = NOW()
WHERE queue_name = $1 AND status = 'failed';

-- Clear single failed job
-- name: DismissJob :one
UPDATE jobs SET status = 'dismissed', updated_at = NOW()
WHERE id = $1 AND status = 'failed'
RETURNING *;

-- Reset failed jobs for a specific queue (for per-queue retry all)
-- name: ResetFailedJobsForQueue :exec
UPDATE jobs SET
    status = 'pending',
    attempt = 0,
    scheduled_at = NOW(),
    visible_until = NULL,
    last_error = NULL,
    updated_at = NOW()
WHERE queue_name = $1 AND status = 'failed';
```

## Claude's Discretion Recommendations

Per CONTEXT.md, these decisions are delegated:

### 1. Multi-open vs Accordion Behavior

**Recommendation: Allow multiple open sections**

Rationale:
- Users may want to compare failed jobs across queues
- Accordion (only one open) would require extra clicks
- templUI `collapsible` component is simpler and allows multiple open

### 2. Expand Trigger (Click Row vs Click Icon)

**Recommendation: Click entire row/trigger area**

Rationale:
- Larger click target = better UX
- Standard pattern in templUI accordion
- Chevron icon rotation provides visual feedback

### 3. Exact Job Info for Completed Jobs in Activity

**Recommendation: Show minimal info**

Display for completed jobs:
- Document name (linked to detail page)
- Completion timestamp
- Duration (if available in payload)

Rationale: Completed jobs need less attention than failed jobs.

### 4. Button Placement for Retry/Clear Actions

**Recommendation: Contextual placement**

- Single job: Inline action buttons in job row (already exists)
- Bulk queue operations: In collapsed header, visible without expanding
- Destructive actions (Clear All): Require confirmation dialog

```html
<!-- Collapsed state shows bulk actions -->
@accordion.Trigger() {
    <div class="flex justify-between items-center w-full">
        <span>default</span>
        <div class="flex items-center gap-4">
            <div class="flex gap-2">
                @badge.Badge() { 5 pending }
                @badge.Badge(badge.Props{Variant: badge.VariantDestructive}) { 2 failed }
            </div>
            if failedCount > 0 {
                <div class="flex gap-2" onclick="event.stopPropagation()">
                    @button.Button(button.Props{Size: button.SizeSm, ...}) { Retry All }
                    <!-- Clear All with dialog trigger -->
                </div>
            }
        </div>
    </div>
}
```

## Open Questions

Things that couldn't be fully resolved:

1. **Queue Names List**
   - What we know: Queue names are stored in jobs table, can be discovered via GetQueueStats
   - What's unclear: Should we show queues with zero jobs? Only active queues?
   - Recommendation: Show all queues that have ever had jobs (from GetQueueStats)

2. **SSE Broadcaster Extension**
   - What we know: Existing StatusBroadcaster sends document-level updates
   - What's unclear: Should we extend it or create a separate QueueActivityBroadcaster?
   - Recommendation: Extend StatusBroadcaster to include queue-level events, avoiding a second connection

3. **Dismissed Jobs Visibility**
   - What we know: Dismissed jobs should remain for audit
   - What's unclear: Should dismissed jobs ever be shown in UI? Admin option to view dismissed?
   - Recommendation: Filter out dismissed by default, defer "show dismissed" toggle to future phase

## Sources

### Primary (HIGH confidence)
- Codebase analysis: `internal/queue/queue.go`, `internal/processing/status.go`, `internal/handler/status.go`
- templUI docs: accordion, collapsible, dialog components
- HTMX docs via Context7: SSE extension, lazy loading patterns

### Secondary (MEDIUM confidence)
- templUI website: https://templui.io/docs/components/accordion
- templUI website: https://templui.io/docs/components/collapsible

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - existing codebase already has all patterns
- Architecture: HIGH - extending existing patterns, not creating new ones
- Pitfalls: HIGH - based on direct codebase analysis
- Database changes: MEDIUM - dismissed status approach is recommendation, needs validation

**Research date:** 2026-02-03
**Valid until:** 2026-03-03 (30 days - stable domain)
