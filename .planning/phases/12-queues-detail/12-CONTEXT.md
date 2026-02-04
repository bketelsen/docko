# Phase 12: Queues Detail - Context

**Gathered:** 2026-02-04
**Status:** Ready for planning

<domain>
## Phase Boundary

Enhanced queues route with expandable details for failed jobs and recent activity. Users can view queue health, inspect failures with full error context, retry or clear failed jobs, and monitor recent completions in real-time. This extends the existing queue dashboard with drill-down capabilities.

</domain>

<decisions>
## Implementation Decisions

### Expander interaction
- Collapsed summary shows queue name, job counts (pending/failed/completed), and health status indicator
- Lazy-load expanded content when user opens a queue (faster initial page load)

### Failed job display
- Full error detail shown, expandable if long (for debugging)
- Limited to ~10-20 jobs initially, with "show more" to load additional
- Each job shows: document name/ID, attempt count, last retry time, failed timestamp, error message
- Chronological order only (most recent failures first, no sorting controls)

### Retry/clear actions
- Single job retry is instant (no confirmation needed)
- Bulk operations (retry all, clear all failed) require confirmation dialog
- Bulk operations available per queue
- "Clear" marks job as dismissed (keeps audit trail, does not delete from database)

### Recent activity
- Time-based: shows last 24 hours by default
- "Show more" loads additional history
- Live updates via SSE as jobs complete
- Document names are clickable links to document detail page

### Claude's Discretion
- Multi-open vs accordion behavior for expanders
- Expand trigger (click row vs click icon)
- Exact job info shown for completed jobs in activity section
- Button placement pattern for retry/clear actions

</decisions>

<specifics>
## Specific Ideas

- Reuse existing SSE patterns from processing status updates (StatusBroadcaster)
- Health status indicator should match dashboard pattern (healthy/warning/issues badges)
- Dismissed status for cleared jobs preserves audit trail without cluttering the UI

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 12-queues-detail*
*Context gathered: 2026-02-04*
