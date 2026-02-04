# Phase 11: Dashboard - Context

**Gathered:** 2026-02-03
**Status:** Ready for planning

<domain>
## Phase Boundary

Root route dashboard showing system overview with stats, counts, and navigation to detail pages. Three domain sections displaying document status, processing health, and source activity. Not a landing page — a functional operations dashboard.

</domain>

<decisions>
## Implementation Decisions

### Layout & Grouping
- Group stats by domain, not by importance
- 3 sections: Documents, Processing (queues + AI combined), Sources (inboxes + network combined)
- Full-width rows for sections (stack vertically)
- Card grid (3-4 cards per row) within each section for individual stats

### Information Depth
- **Documents section:** Total count + breakdown by status (processed, pending, failed) + tag count + correspondent count
- **Processing section:** Queue counts (pending/completed/failed) + AI suggestion stats + recent activity + failures
- **Sources section:** Inbox and network source counts + enabled/disabled status + last sync time or files imported today
- Failures shown within their respective sections (not a prominent top banner)
- 5 items for recent activity lists
- AI pending suggestions count with direct link to AI review queue
- Include "today's activity" stats (documents uploaded today, jobs processed today)

### Status Indicators
- Queue health shown with badge (healthy/warning/issues)
- Warning triggers: any failed jobs OR high backlog (10+ pending)
- Source status: green dot for enabled, gray dot for disabled
- Show active AI provider name on dashboard (OpenAI/Claude/Ollama)

### Navigation
- Section headers are clickable (link to detail page)
- Explicit "View all" link also present per section for clarity
- All stat cards are clickable, navigate to relevant page/filter
- Multiple quick actions: Upload, Add Inbox, Sync Now
- Quick actions placed within their relevant sections (not at top)

### Claude's Discretion
- Exact card sizing and spacing within grid
- Specific thresholds for "issues" vs "warning" badge
- How to handle empty state for each section
- Mobile responsive breakpoints for card grid

</decisions>

<specifics>
## Specific Ideas

- Dashboard should feel like an operations console — everything at a glance
- Clicking failed count should filter/navigate to show those specific items
- AI review link should stand out if there are pending suggestions to review

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 11-dashboard*
*Context gathered: 2026-02-03*
