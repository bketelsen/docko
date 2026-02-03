# Phase 5: Organization - Research

**Researched:** 2026-02-03
**Domain:** Tag and Correspondent Management with HTMX/Templ/Go
**Confidence:** HIGH

## Summary

This phase implements tag and correspondent management for document organization. The research focused on three main areas: (1) HTMX patterns for searchable dropdowns with inline creation, (2) database patterns for managing many-to-many relationships and merging records, and (3) UI components from templUI that align with the existing codebase patterns.

The existing codebase already has database tables for tags, correspondents, document_tags, and document_correspondents. The UI will leverage templUI's `selectbox` component (with built-in search) and `dialog` component for modal forms. HTMX's active search pattern provides the foundation for dropdown filtering and inline tag creation.

**Primary recommendation:** Use templUI's `selectbox` component for tag/correspondent pickers with server-side search via HTMX. Implement inline creation by detecting "not found" scenarios and offering a quick-create option. Use existing `dialog` component patterns for create/edit modals.

## Standard Stack

The established libraries/tools for this domain:

### Core (Already in Codebase)
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| templUI | v1.4.0 | UI components | Already installed, provides dialog, selectbox, tabs, breadcrumb |
| HTMX | 2.0.4 | Server-driven interactions | Already in use for forms, SSE, modals |
| Templ | latest | Go templating | Project standard |
| sqlc | v2 | Type-safe SQL queries | Project standard |
| pgx/v5 | latest | PostgreSQL driver | Project standard |

### Supporting (Need to Install)
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| selectbox | v1.4.0 | Searchable dropdown | Tag/correspondent picker |
| popover | v1.4.0 | Floating UI | Required by selectbox |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| templUI selectbox | Custom HTMX dropdown | selectbox has search built-in, saves dev time |
| Server-side search | Client-side filter | Server-side scales better with many tags |
| Modal dialogs | Inline forms | Modals match existing inbox pattern |

**Installation:**
```bash
templui add selectbox popover
```

## Architecture Patterns

### Recommended Project Structure
```
internal/
  handler/
    tags.go              # Tag CRUD handlers
    correspondents.go    # Correspondent CRUD handlers
sqlc/
  queries/
    tags.sql             # Tag queries with counts
    correspondents.sql   # Correspondent queries with counts
templates/
  pages/admin/
    tags.templ           # Tag management page
    correspondents.templ # Correspondent management page
  partials/
    tag_picker.templ     # Reusable tag picker component
    correspondent_picker.templ # Reusable correspondent picker
```

### Pattern 1: Searchable Dropdown with HTMX
**What:** Server-side filtered dropdown using active search pattern
**When to use:** Tag/correspondent selection with large lists
**Example:**
```html
<!-- Source: htmx.org/examples/active-search/ -->
<input type="search"
       name="q"
       placeholder="Search tags..."
       hx-get="/api/tags/search"
       hx-trigger="input changed delay:300ms, keyup[key=='Enter']"
       hx-target="#tag-results"
       hx-indicator=".search-indicator">
<div id="tag-results">
  <!-- Server returns filtered options -->
</div>
```

### Pattern 2: Inline Create While Selecting
**What:** Allow creating new tags when typed name doesn't exist
**When to use:** During tag assignment workflow
**Example:**
```html
<!-- Server detects no match and returns create option -->
<div id="tag-results">
  <div class="text-muted-foreground px-3 py-2">
    No tags matching "new-tag"
  </div>
  <button hx-post="/api/tags"
          hx-vals='{"name": "new-tag"}'
          hx-swap="afterbegin"
          hx-target="#tag-results">
    Create "new-tag"
  </button>
</div>
```

### Pattern 3: Modal CRUD with HTMX
**What:** Dialog modals for create/edit operations
**When to use:** Tag/correspondent management UI
**Example:**
```go
// Source: existing inboxes.templ pattern + templUI dialog
@dialog.Dialog(dialog.Props{ID: "create-tag"}) {
    @dialog.Trigger() {
        @button.Button() { Create Tag }
    }
    @dialog.Content() {
        <form hx-post="/tags" hx-target="#tag-list" hx-swap="beforeend">
            // Form fields
            @dialog.Footer() {
                @dialog.Close() { Cancel }
                <button type="submit">Create</button>
            }
        </form>
    }
}
```

### Pattern 4: Correspondent Merge Operation
**What:** Database transaction to merge multiple correspondents into one
**When to use:** Bulk merge workflow
**Example:**
```sql
-- Source: PostgreSQL best practices
-- Transaction: merge correspondents B, C into A
BEGIN;
  -- Update all document references
  UPDATE document_correspondents
  SET correspondent_id = $1
  WHERE correspondent_id = ANY($2);

  -- Append notes from merged correspondents
  UPDATE correspondents
  SET notes = COALESCE(notes, '') || $3
  WHERE id = $1;

  -- Delete merged correspondents
  DELETE FROM correspondents WHERE id = ANY($2);
COMMIT;
```

### Anti-Patterns to Avoid
- **Client-side filtering only:** Won't scale when tag count grows; use server-side search
- **N+1 queries for document counts:** Use joins/subqueries to get counts in single query
- **Deleting tags with documents:** Either reassign or warn user first
- **Inline editing everywhere:** Modal dialogs are cleaner for forms with multiple fields

## Don't Hand-Roll

Problems that look simple but have existing solutions:

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Searchable dropdown | Custom dropdown with JS | templUI selectbox | Has search, keyboard nav, accessibility built-in |
| Color picker | Custom color input | Preset color buttons | User decision: curated palette, not open picker |
| Modal dialogs | Custom modal code | templUI dialog | Already in codebase, consistent UX |
| Debounced search | setTimeout logic | HTMX trigger delay | `hx-trigger="input changed delay:300ms"` handles it |
| Form validation | Manual checks | Browser + server validation | HTML5 required + server-side checks |

**Key insight:** The existing codebase patterns (templUI components, HTMX attributes, sqlc queries) should be reused. The tag/correspondent management is standard CRUD with a few special operations (inline create, bulk merge).

## Common Pitfalls

### Pitfall 1: Orphaned Tag Relationships on Delete
**What goes wrong:** Deleting a tag doesn't clean up document_tags junction table
**Why it happens:** Forgetting that many-to-many requires cascade or manual cleanup
**How to avoid:** Database already has `ON DELETE CASCADE` on document_tags; verify this is honored
**Warning signs:** Foreign key constraint errors on tag deletion

### Pitfall 2: Race Conditions in Concurrent Tag Creation
**What goes wrong:** Two users create same tag name simultaneously
**Why it happens:** UNIQUE constraint violation if not handled
**How to avoid:** Use `INSERT ... ON CONFLICT DO NOTHING RETURNING` or catch unique violation and return existing tag
**Warning signs:** 500 errors when creating tags quickly

### Pitfall 3: N+1 Queries for Document Counts
**What goes wrong:** Slow page load when listing tags with their document counts
**Why it happens:** Separate COUNT query for each tag
**How to avoid:** Use LEFT JOIN with GROUP BY or subquery in single query
**Warning signs:** Page slows down as tag count increases

### Pitfall 4: Missing Transaction for Merge Operation
**What goes wrong:** Partial merge leaves data inconsistent
**Why it happens:** Update references and delete happen separately without transaction
**How to avoid:** Wrap entire merge in database transaction
**Warning signs:** Orphaned document references, missing notes

### Pitfall 5: HTMX Target Confusion with Modals
**What goes wrong:** Form submission replaces wrong element or entire page
**Why it happens:** hx-target not set correctly for modal forms
**How to avoid:** Use specific element IDs for targets; close modal after success
**Warning signs:** Page content duplicated or modal doesn't close

## Code Examples

Verified patterns from official sources:

### SQL: List Tags with Document Counts
```sql
-- name: ListTagsWithCounts :many
SELECT
    t.id, t.name, t.color, t.created_at,
    COUNT(dt.document_id)::int as document_count
FROM tags t
LEFT JOIN document_tags dt ON dt.tag_id = t.id
GROUP BY t.id
ORDER BY t.name;
```

### SQL: Create Tag with Conflict Handling
```sql
-- name: CreateTagOrGet :one
INSERT INTO tags (name, color)
VALUES ($1, $2)
ON CONFLICT (name) DO UPDATE SET name = tags.name
RETURNING *;
```

### SQL: Merge Correspondents
```sql
-- name: MergeCorrespondents :exec
-- First update all document references to target correspondent
UPDATE document_correspondents
SET correspondent_id = $1
WHERE correspondent_id = ANY($2::uuid[]);

-- name: AppendCorrespondentNotes :one
UPDATE correspondents
SET notes = COALESCE(notes, '') || E'\n---\n' || $2
WHERE id = $1
RETURNING *;

-- name: DeleteCorrespondents :exec
DELETE FROM correspondents WHERE id = ANY($1::uuid[]);
```

### Go Handler: Search Tags Endpoint
```go
// Source: HTMX active search pattern
func (h *Handler) SearchTags(c echo.Context) error {
    ctx := c.Request().Context()
    query := c.QueryParam("q")

    tags, err := h.db.Queries.SearchTags(ctx, "%"+query+"%")
    if err != nil {
        return c.String(http.StatusInternalServerError, "Search failed")
    }

    // If no results and query not empty, offer create option
    return partials.TagSearchResults(tags, query).Render(ctx, c.Response().Writer)
}
```

### Templ: Tag Picker with Inline Create
```templ
// Source: templUI selectbox + HTMX patterns
templ TagPicker(documentID string, currentTags []sqlc.Tag) {
    <div class="relative">
        <input type="search"
               name="q"
               placeholder="Search or create tags..."
               hx-get={ fmt.Sprintf("/documents/%s/tags/search", documentID) }
               hx-trigger="input changed delay:300ms, focus"
               hx-target="#tag-search-results"
               class="w-full px-3 py-2 border rounded-md"/>
        <div id="tag-search-results" class="absolute w-full bg-background border rounded-md shadow-lg hidden">
            // Server populates with matching tags + create option
        </div>
    </div>
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| jQuery select2 | Native + HTMX | 2023-2024 | No JS library dependency |
| Client-side filtering | Server-side with HTMX | 2024 | Better scalability |
| Full page reload CRUD | HTMX partial updates | 2023 | Better UX |
| CSS color picker | Preset palette buttons | N/A | Simpler UX (user decision) |

**Deprecated/outdated:**
- jQuery-based tag libraries: Replaced by HTMX + server-side rendering
- Full form posts with page reload: Use HTMX for partial updates

## Curated Color Palette

Based on Tailwind CSS v4 defaults, recommended 12-color palette for tags:

| Name | Tailwind Class | Hex (light mode) | Usage |
|------|----------------|------------------|-------|
| Red | bg-red-500 | #ef4444 | Urgent, important |
| Orange | bg-orange-500 | #f97316 | Warning, attention |
| Amber | bg-amber-500 | #f59e0b | Highlight |
| Yellow | bg-yellow-500 | #eab308 | Note |
| Green | bg-green-500 | #22c55e | Success, done |
| Emerald | bg-emerald-500 | #10b981 | Active |
| Teal | bg-teal-500 | #14b8a6 | Info |
| Blue | bg-blue-500 | #3b82f6 | Default, neutral |
| Indigo | bg-indigo-500 | #6366f1 | Feature |
| Purple | bg-purple-500 | #a855f7 | Special |
| Pink | bg-pink-500 | #ec4899 | Personal |
| Gray | bg-gray-500 | #6b7280 | Archive, inactive |

Store as simple string value (e.g., "red", "blue") in database `tags.color` column, apply Tailwind class in template.

## Open Questions

Things that couldn't be fully resolved:

1. **Delete confirmation UX (Claude's discretion)**
   - What we know: User wants confirmation before delete
   - Options: (A) Show document count in confirm dialog, (B) Block delete if tag has documents
   - Recommendation: Option A - show count, allow delete with cascade (per database design)

2. **Sidebar vs Settings feature parity (Claude's discretion)**
   - What we know: Both locations should have tag/correspondent management
   - Recommendation: Sidebar links go to full management pages; no separate "quick" vs "full" versions

3. **Modal form layout (Claude's discretion)**
   - Recommendation: Single-column layout matching existing inbox create form pattern

## Sources

### Primary (HIGH confidence)
- Existing codebase: `/home/bjk/projects/corpus/docko/` - migrations, handlers, templates
- templUI v1.4.0 documentation - dialog, selectbox components
- HTMX official examples - active search pattern

### Secondary (MEDIUM confidence)
- [HTMX Active Search](https://htmx.org/examples/active-search/) - debounced search pattern
- [templUI Select Box](https://templui.io/docs/components/select-box) - searchable dropdown API
- [templUI Dialog](https://templui.io/docs/components/dialog) - modal forms
- [PostgreSQL MERGE patterns](https://www.baeldung.com/sql/postgresql-upsert-merge-insert) - conflict handling
- [Tailwind CSS Colors](https://tailwindcss.com/docs/customizing-colors) - color palette

### Tertiary (LOW confidence)
- [HTMX Best Practices 2026](https://www.refactor.website/web-development/htmx-no-build-modern-web-apps-2026) - general patterns
- [InclusiveColors](https://www.inclusivecolors.com/) - accessible color selection

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - Components already in codebase, well-documented
- Architecture: HIGH - Patterns verified against existing handlers and templates
- Pitfalls: HIGH - Based on database schema analysis and HTMX documentation
- Color palette: MEDIUM - Based on Tailwind defaults, specific choices are subjective

**Research date:** 2026-02-03
**Valid until:** 2026-03-03 (30 days - stable domain, established patterns)
