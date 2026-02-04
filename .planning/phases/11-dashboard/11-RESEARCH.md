# Phase 11: Dashboard - Research

**Researched:** 2026-02-03
**Domain:** Go/Templ Dashboard with templUI Components
**Confidence:** HIGH

## Summary

This phase implements an operations dashboard at the root route, replacing the current placeholder dashboard. The research focuses on three areas: (1) data aggregation patterns using sqlc queries, (2) templUI component composition for stat cards and status indicators, and (3) Go template patterns for passing dashboard data to templates.

The codebase already has strong patterns established. The queue dashboard (`queue_dashboard.templ`) provides an excellent reference for stat cards, badge status indicators, and table-in-card layouts. The sqlc query patterns for counts and aggregations are well-established in existing files. The decisions from CONTEXT.md specify the exact structure: three domain sections (Documents, Processing, Sources) with card grids within each.

**Primary recommendation:** Create a `DashboardData` struct to aggregate all stats in the handler, then pass to a single template that composes the three sections using existing templUI card patterns.

## Standard Stack

This phase uses only what's already in the codebase - no new dependencies needed.

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| templ | v1.4.0 | Template generation | Already in use |
| templUI | v1.4.0 | UI components | Already installed, card/badge/table/button ready |
| sqlc | v1.30.0 | Type-safe queries | Already generating queries |
| Echo | v4 | HTTP routing | Already handling routes |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| tailwind-merge-go | latest | CSS class merging | Already used by templUI components |

### Alternatives Considered
None - the stack is locked by prior phases.

## Architecture Patterns

### Recommended Dashboard Structure

```
templates/pages/admin/dashboard.templ    # Main dashboard template (replace existing placeholder)
internal/handler/admin.go               # Handler fetches data, passes to template
sqlc/queries/dashboard.sql              # New file for dashboard-specific aggregations
```

### Pattern 1: Data Aggregation in Handler

**What:** Handler aggregates all dashboard data into a single struct before rendering.
**When to use:** Always for dashboard pages with multiple data sources.
**Example:**
```go
// Source: Existing pattern in queue_dashboard handler (internal/handler/ai.go:198)
type DashboardData struct {
    // Documents section
    DocumentStats DocumentStats
    TagCount      int
    CorrespondentCount int

    // Processing section
    QueueStats    []sqlc.GetQueueStatsRow
    FailedJobCount int64
    PendingSuggestions int64
    RecentJobs    []sqlc.Job

    // Sources section
    InboxStats    InboxStats
    NetworkStats  NetworkStats

    // Active provider
    ActiveProvider string
}

func (h *Handler) AdminDashboard(c echo.Context) error {
    ctx := c.Request().Context()

    data := DashboardData{}

    // Aggregate all data (errors logged, gracefully degrade)
    // ...

    return admin.Dashboard(data).Render(ctx, c.Response().Writer)
}
```

### Pattern 2: Stat Card with Header Row Layout

**What:** Card with header containing title on left, icon/badge on right, content below.
**When to use:** All numeric stat displays.
**Example:**
```templ
// Source: Existing pattern in dashboard.templ (templates/pages/admin/dashboard.templ:81)
@card.Card() {
    @card.Header(card.HeaderProps{Class: "flex flex-row items-center justify-between space-y-0 pb-2"}) {
        @card.Title(card.TitleProps{Class: "text-sm font-medium text-muted-foreground"}) {
            { title }
        }
        // Icon or badge on right
    }
    @card.Content(card.ContentProps{Class: "pt-0"}) {
        <div class="text-2xl font-bold">{ value }</div>
        <p class="text-xs text-muted-foreground">{ subtitle }</p>
    }
}
```

### Pattern 3: Section with Clickable Header

**What:** Full-width section with clickable title linking to detail page.
**When to use:** Each domain section (Documents, Processing, Sources).
**Example:**
```templ
// Based on CONTEXT.md requirements
<section class="space-y-4">
    <div class="flex items-center justify-between">
        <a href="/documents" class="group">
            <h2 class="text-lg font-semibold hover:text-primary transition-colors">
                Documents
                <svg class="inline w-4 h-4 opacity-0 group-hover:opacity-100 transition-opacity">...</svg>
            </h2>
        </a>
        <a href="/documents" class="text-sm text-muted-foreground hover:text-primary">View all</a>
    </div>
    <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
        // Stat cards
    </div>
</section>
```

### Pattern 4: Badge Status Indicator

**What:** Badge showing health status (healthy/warning/issues).
**When to use:** Queue health, source status.
**Example:**
```templ
// Source: Existing pattern in queue_dashboard.templ (templates/pages/admin/queue_dashboard.templ:216)
// Adapted for health status
templ healthBadge(health string) {
    switch health {
        case "healthy":
            @badge.Badge(badge.Props{Class: "bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200 border-transparent"}) {
                Healthy
            }
        case "warning":
            @badge.Badge(badge.Props{Class: "bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200 border-transparent"}) {
                Warning
            }
        case "issues":
            @badge.Badge(badge.Props{Variant: badge.VariantDestructive}) {
                Issues
            }
    }
}
```

### Pattern 5: Status Dot Indicator

**What:** Small colored dot for enabled/disabled state.
**When to use:** Source enabled status (inboxes, network sources).
**Example:**
```templ
// Common pattern for status indicators
templ statusDot(enabled bool) {
    if enabled {
        <span class="inline-block w-2 h-2 rounded-full bg-green-500" title="Enabled"></span>
    } else {
        <span class="inline-block w-2 h-2 rounded-full bg-muted-foreground" title="Disabled"></span>
    }
}
```

### Anti-Patterns to Avoid
- **Multiple render calls per section:** Don't render sections separately - compose one template with all data
- **Inline SQL in handlers:** Use sqlc queries, don't write raw SQL in handlers
- **Missing error handling:** Always handle query errors gracefully - show "N/A" or 0, don't crash

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Card layout | Custom divs with manual styling | `card.Card()`, `card.Header()`, `card.Content()` | Consistent styling, dark mode support |
| Status badges | Custom spans | `badge.Badge()` with variants | Built-in variants match design system |
| Tables for lists | Manual table HTML | `table.Table()` components | Responsive, consistent styling |
| Clickable cards | Card with onclick JS | `<a>` wrapping card or card with `Attributes` for hx-get | Native navigation, proper semantics |

**Key insight:** templUI components already handle dark mode, responsive behavior, and consistent styling. Hand-rolling means duplicating this work and risking inconsistency.

## Common Pitfalls

### Pitfall 1: N+1 Query Problem
**What goes wrong:** Handler makes separate query for each stat, causing many database round trips.
**Why it happens:** Natural to think "get documents, then get count, then get failed..."
**How to avoid:** Create aggregate queries that return multiple stats in one call.
**Warning signs:** Dashboard loads slowly, database connection pool exhaustion.

### Pitfall 2: Blocking on Failed Queries
**What goes wrong:** One failed query (e.g., AI service down) crashes entire dashboard.
**Why it happens:** Error handling returns early on any error.
**How to avoid:** Log errors, use default values, show partial dashboard.
**Warning signs:** Dashboard shows blank page when one service is unhealthy.

### Pitfall 3: Template Signature Bloat
**What goes wrong:** Template takes 15+ individual parameters.
**Why it happens:** Adding one more stat, then another...
**How to avoid:** Create a single `DashboardData` struct with nested sections.
**Warning signs:** Long parameter lists, difficult to add new stats.

### Pitfall 4: Inconsistent Card Sizing
**What goes wrong:** Cards in same row have different heights due to content.
**Why it happens:** Variable content length, missing min-height constraints.
**How to avoid:** Use consistent card structure, truncate long text, grid handles equal height.
**Warning signs:** Visual jumping, cards not aligned.

### Pitfall 5: Stale "Today" Stats
**What goes wrong:** "Documents uploaded today" shows stale counts.
**Why it happens:** No timezone handling, cache issues.
**How to avoid:** Use server's timezone consistently, fresh queries on each load.
**Warning signs:** Counts reset at wrong time, different users see different counts.

## Code Examples

### Dashboard Data Struct
```go
// Source: Pattern derived from existing codebase
type DashboardData struct {
    // Documents section
    Documents struct {
        Total     int64
        Processed int64
        Pending   int64
        Failed    int64
        Today     int64 // Uploaded today
    }
    TagCount         int64
    CorrespondentCount int64

    // Processing section
    Processing struct {
        Pending   int64
        Completed int64
        Failed    int64
        Health    string // "healthy", "warning", "issues"
    }
    PendingSuggestions int64
    RecentJobs         []sqlc.Job
    ActiveProvider     string

    // Sources section
    Inboxes struct {
        Total   int64
        Enabled int64
    }
    NetworkSources struct {
        Total   int64
        Enabled int64
    }
    FilesImportedToday int64
}
```

### Dashboard Aggregation Query
```sql
-- name: GetDashboardDocumentStats :one
-- Source: New query for dashboard
SELECT
    COUNT(*) as total,
    COUNT(*) FILTER (WHERE processing_status = 'completed') as processed,
    COUNT(*) FILTER (WHERE processing_status = 'pending') as pending,
    COUNT(*) FILTER (WHERE processing_status = 'failed') as failed,
    COUNT(*) FILTER (WHERE created_at >= CURRENT_DATE) as today
FROM documents;

-- name: GetDashboardQueueStats :one
SELECT
    COUNT(*) FILTER (WHERE status = 'pending') as pending,
    COUNT(*) FILTER (WHERE status = 'completed') as completed,
    COUNT(*) FILTER (WHERE status = 'failed') as failed
FROM jobs;

-- name: GetDashboardSourceStats :one
SELECT
    (SELECT COUNT(*) FROM inboxes) as inbox_total,
    (SELECT COUNT(*) FROM inboxes WHERE enabled = true) as inbox_enabled,
    (SELECT COUNT(*) FROM network_sources) as network_total,
    (SELECT COUNT(*) FROM network_sources WHERE enabled = true) as network_enabled;

-- name: CountTags :one
SELECT COUNT(*) FROM tags;

-- name: CountCorrespondents :one
SELECT COUNT(*) FROM correspondents;
```

### Health Status Calculation
```go
// Source: Based on CONTEXT.md requirements
func calculateQueueHealth(pending, failed int64) string {
    if failed > 0 {
        return "issues"
    }
    if pending >= 10 {
        return "warning"
    }
    return "healthy"
}
```

### Clickable Stat Card
```templ
// Source: Based on existing StatCard pattern with navigation
templ ClickableStatCard(title, value, href string, icon templ.Component) {
    <a href={ templ.SafeURL(href) } class="block">
        @card.Card(card.Props{Class: "hover:border-primary/50 transition-colors cursor-pointer"}) {
            @card.Header(card.HeaderProps{Class: "flex flex-row items-center justify-between space-y-0 pb-2"}) {
                @card.Title(card.TitleProps{Class: "text-sm font-medium text-muted-foreground"}) {
                    { title }
                }
                @icon
            }
            @card.Content(card.ContentProps{Class: "pt-0"}) {
                <div class="text-2xl font-bold">{ value }</div>
            }
        }
    </a>
}
```

### Quick Action Button in Section
```templ
// Source: Based on existing button patterns, CONTEXT.md requirements
templ QuickActionButton(label, href string, icon templ.Component) {
    @button.Button(button.Props{
        Variant: button.VariantOutline,
        Size:    button.SizeSm,
        Href:    href,
    }) {
        @icon
        <span class="ml-1">{ label }</span>
    }
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Separate queries per stat | Aggregate queries with FILTER | PostgreSQL 9.4+ | Fewer round trips |
| Individual template params | Data struct parameter | Pattern convention | Cleaner signatures |
| Static stat cards | Clickable navigation cards | CONTEXT.md decision | Better UX |

**Deprecated/outdated:**
- The existing placeholder dashboard in `dashboard.templ` will be completely replaced

## Open Questions

1. **Active Provider Display**
   - What we know: `aiSvc.AvailableProviders()` returns list of available providers, `settings.PreferredProvider` is the configured preference
   - What's unclear: Whether to show "preferred" provider or "last used" provider
   - Recommendation: Show `settings.PreferredProvider` if set and available, otherwise show first available, or "None configured"

2. **Files Imported Today Scope**
   - What we know: `network_sources.files_imported` is cumulative count, not per-day
   - What's unclear: Whether to track daily imports separately
   - Recommendation: For MVP, show cumulative. Add daily tracking in future phase if needed.

3. **Recent Activity List Contents**
   - What we know: CONTEXT.md specifies 5 items
   - What's unclear: Mix of jobs and inbox events, or just jobs?
   - Recommendation: Show recent jobs only (already have `GetRecentJobs` query). Simpler, more actionable.

## Sources

### Primary (HIGH confidence)
- Existing codebase files examined:
  - `templates/pages/admin/dashboard.templ` - Current placeholder, patterns to follow
  - `templates/pages/admin/queue_dashboard.templ` - Reference for stat cards, tables, badges
  - `components/card/card.templ` - templUI v1.4.0 card component API
  - `components/badge/badge.templ` - templUI v1.4.0 badge variants
  - `sqlc/queries/jobs.sql` - Existing aggregation query patterns
  - `sqlc/queries/documents.sql` - Document query patterns
  - `internal/handler/ai.go` - Handler data aggregation patterns

### Secondary (MEDIUM confidence)
- CONTEXT.md decisions for structure and requirements

### Tertiary (LOW confidence)
- None - all findings verified against codebase

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - Already in codebase, no new dependencies
- Architecture: HIGH - Patterns extracted from existing working code
- Pitfalls: HIGH - Based on common Go/database patterns and existing codebase structure

**Research date:** 2026-02-03
**Valid until:** 2026-03-03 (stable patterns, no external dependencies)
