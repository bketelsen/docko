# Phase 6: Search - Research

**Researched:** 2026-02-03
**Domain:** PostgreSQL Full-Text Search + HTMX Search UI
**Confidence:** HIGH

## Summary

This phase implements full-text search with filtering on the existing Documents page. The standard approach for this stack is PostgreSQL's built-in full-text search with GIN indexes, combined with HTMX's active search pattern for responsive UI.

PostgreSQL full-text search is well-suited for document management systems of this scale. The codebase already stores extracted text in `documents.text_content`, which can be indexed with a GIN index on a generated tsvector column. The filtering by tags, correspondent, and date integrates naturally with sqlc's conditional query pattern using boolean flags.

For the UI, HTMX's debounced input pattern (`hx-trigger="input changed delay:500ms"`) provides instant search with efficient request management. URL state via `hx-push-url` enables shareable/bookmarkable searches.

**Primary recommendation:** Add a generated `search_vector` column with GIN index to documents table, use `websearch_to_tsquery` for user input, and implement HTMX active search with 500ms debounce and URL state persistence.

## Standard Stack

The established libraries/tools for this domain:

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| PostgreSQL FTS | Built-in (PG 16+) | Full-text search | Native, no external dependencies, good performance for <1M docs |
| GIN Index | Built-in | tsvector indexing | Recommended by PostgreSQL docs for text search |
| HTMX | Already in project | Search UI interactions | Already used, proven patterns for active search |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| templ | Already in project | Search UI components | Filter chips, result partials |
| sqlc | Already in project | Type-safe queries | Search queries with filters |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| PostgreSQL FTS | Elasticsearch/Meilisearch | External service, more complex ops, overkill for this scale |
| GIN Index | GiST Index | GiST better for frequent updates, GIN faster for reads (our use case) |
| Stored tsvector column | Expression index | Stored column faster at query time, clearer code |

**No additional installation required** - all capabilities are built into the existing stack.

## Architecture Patterns

### Recommended Approach

**1. Database Layer: Generated tsvector Column**

Add a migration to create a stored generated column and GIN index:

```sql
-- Add search vector column (generated/stored)
ALTER TABLE documents
    ADD COLUMN search_vector tsvector
    GENERATED ALWAYS AS (
        to_tsvector('english',
            coalesce(original_filename, '') || ' ' ||
            coalesce(text_content, '')
        )
    ) STORED;

-- Create GIN index for fast full-text search
CREATE INDEX idx_documents_search ON documents USING GIN (search_vector);
```

Why stored column over expression index:
- Query doesn't need to recompute tsvector (faster reads)
- Can use `@@` operator directly on column
- Clearer code - no need to repeat expression in queries

**2. Query Pattern: Optional Filters with Boolean Flags**

sqlc doesn't support dynamic WHERE clauses. Use the boolean flag pattern:

```sql
-- name: SearchDocuments :many
SELECT
    d.*,
    c.id as correspondent_id,
    c.name as correspondent_name,
    ts_rank(d.search_vector, websearch_to_tsquery('english', @query)) as rank,
    ts_headline('english', d.text_content, websearch_to_tsquery('english', @query),
        'MaxFragments=1, MaxWords=30, MinWords=15, StartSel=<mark>, StopSel=</mark>') as headline
FROM documents d
LEFT JOIN document_correspondents dc ON dc.document_id = d.id
LEFT JOIN correspondents c ON c.id = dc.correspondent_id
WHERE
    -- Full-text search (optional - empty string matches all)
    (sqlc.narg(query)::text IS NULL OR sqlc.narg(query)::text = ''
        OR d.search_vector @@ websearch_to_tsquery('english', sqlc.narg(query)::text))
    -- Correspondent filter (optional)
    AND (NOT sqlc.arg(has_correspondent)::boolean OR c.id = sqlc.arg(correspondent_id)::uuid)
    -- Date range filter (optional)
    AND (NOT sqlc.arg(has_date_from)::boolean OR d.document_date >= sqlc.arg(date_from)::timestamptz)
    AND (NOT sqlc.arg(has_date_to)::boolean OR d.document_date <= sqlc.arg(date_to)::timestamptz)
ORDER BY
    CASE WHEN sqlc.narg(query)::text IS NOT NULL AND sqlc.narg(query)::text != ''
         THEN ts_rank(d.search_vector, websearch_to_tsquery('english', sqlc.narg(query)::text))
         ELSE 0 END DESC,
    d.document_date DESC
LIMIT sqlc.arg(limit_count) OFFSET sqlc.arg(offset_count);
```

**3. Tag Filtering: Separate Query with AND Logic**

Tags require AND logic (must have ALL selected tags). Handle with a subquery:

```sql
-- When filtering by tags, add this condition:
AND (NOT sqlc.arg(has_tags)::boolean
    OR d.id IN (
        SELECT dt.document_id
        FROM document_tags dt
        WHERE dt.tag_id = ANY(sqlc.arg(tag_ids)::uuid[])
        GROUP BY dt.document_id
        HAVING COUNT(DISTINCT dt.tag_id) = sqlc.arg(tag_count)::int
    ))
```

**4. HTMX Search UI Pattern**

```html
<form hx-get="/documents"
      hx-trigger="input changed delay:500ms from:find input[name='q'], change from:find select"
      hx-target="#document-results"
      hx-push-url="true"
      hx-indicator="#search-indicator">

    <!-- Search input with clear button -->
    <div class="relative">
        <input type="search" name="q" placeholder="Search documents..."
               value="{{ .Query }}"
               class="...">
        <span id="search-indicator" class="htmx-indicator absolute right-2">
            <!-- spinner -->
        </span>
    </div>

    <!-- Filter controls inline -->
    <div class="flex gap-2">
        <select name="correspondent_id">...</select>
        <select name="date_range">
            <option value="">Any time</option>
            <option value="today">Today</option>
            <option value="7d">Last 7 days</option>
            <option value="30d">Last 30 days</option>
            <option value="1y">Last year</option>
        </select>
    </div>

    <!-- Active filter chips -->
    <div id="active-filters" class="flex gap-1">
        {{ range .ActiveFilters }}
        <span class="chip">
            {{ .Label }}
            <button hx-get="/documents?{{ .RemoveParams }}" hx-target="#document-results">x</button>
        </span>
        {{ end }}
    </div>
</form>
```

### Pattern: Server Handles Both HTMX and Full Page Requests

Critical for URL state: server must detect HTMX vs direct requests:

```go
func (h *Handler) DocumentsPage(c echo.Context) error {
    // Parse search params from URL
    params := parseSearchParams(c)

    // Execute search query
    results, err := h.db.Queries.SearchDocuments(ctx, params)

    // Check if HTMX request (partial) or full page
    if c.Request().Header.Get("HX-Request") == "true" {
        // Return just the results partial
        return partials.DocumentResults(results).Render(ctx, c.Response())
    }

    // Return full page with results
    return pages.Documents(results, params).Render(ctx, c.Response())
}
```

### Anti-Patterns to Avoid
- **Don't call ts_headline for every document** - Only call it on the documents being displayed (use LIMIT first, then headline)
- **Don't use to_tsquery for user input** - Use websearch_to_tsquery which never throws syntax errors
- **Don't rebuild tsvector on every query** - Use stored generated column
- **Don't skip NULL handling** - Always coalesce nullable columns in tsvector

## Don't Hand-Roll

Problems that look simple but have existing solutions:

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Text search | LIKE/ILIKE queries | PostgreSQL tsvector/tsquery | LIKE is O(n), FTS uses index |
| Stemming/normalization | Regex patterns | to_tsvector built-in | Handles plurals, tenses automatically |
| Search result highlighting | String manipulation | ts_headline | Handles word boundaries, fragments |
| Debounced search | Custom JS debounce | HTMX delay modifier | Built-in, battle-tested |
| URL state sync | Custom JS pushState | hx-push-url | Automatic history management |
| Query parsing | Custom parser | websearch_to_tsquery | Handles quotes, OR, minus operators |

**Key insight:** PostgreSQL's full-text search is remarkably capable. The main reason to add external search (Elasticsearch, Meilisearch) is for features like typo tolerance, faceted search, or very large scale - none of which apply here.

## Common Pitfalls

### Pitfall 1: Forgetting NULL Handling in tsvector
**What goes wrong:** `to_tsvector(text_content)` returns NULL if text_content is NULL, making the entire concatenation NULL
**Why it happens:** NULL propagation in PostgreSQL
**How to avoid:** Always use `coalesce(column, '')` in tsvector generation
**Warning signs:** Search returns no results for documents with partial metadata

### Pitfall 2: Using to_tsquery Instead of websearch_to_tsquery
**What goes wrong:** User types "hello world" and gets a syntax error
**Why it happens:** to_tsquery requires operators between terms, websearch_to_tsquery doesn't
**How to avoid:** Always use `websearch_to_tsquery` for user input
**Warning signs:** 500 errors when users type normal search phrases

### Pitfall 3: Calling ts_headline on All Documents
**What goes wrong:** Search becomes slow (seconds) with many results
**Why it happens:** ts_headline processes original text, not indexed tsvector
**How to avoid:** Apply LIMIT first, then ts_headline only on displayed results. Or generate headlines in a subquery with LIMIT.
**Warning signs:** Search time scales linearly with matching documents

### Pitfall 4: Not Specifying Text Search Configuration
**What goes wrong:** Different results on different servers or after config changes
**Why it happens:** Defaults to `default_text_search_config` which varies by installation
**How to avoid:** Always specify 'english' (or appropriate config) in to_tsvector and query functions
**Warning signs:** Inconsistent search results between environments

### Pitfall 5: HTMX Request Detection for URL State
**What goes wrong:** Shared URLs show broken partial instead of full page
**Why it happens:** Server returns partial for direct browser request
**How to avoid:** Check `HX-Request` header; return full page for non-HTMX requests
**Warning signs:** Bookmarked/shared search URLs render incorrectly

### Pitfall 6: Not Handling Empty Search State
**What goes wrong:** User clears search and nothing happens, or page breaks
**Why it happens:** Empty string edge case not handled
**How to avoid:** Treat empty/null query as "no filter" - return all documents
**Warning signs:** Empty search input causes errors or shows "no results"

## Code Examples

Verified patterns from official sources:

### PostgreSQL: Generated tsvector Column
```sql
-- Source: https://www.postgresql.org/docs/current/textsearch-tables.html
ALTER TABLE documents
    ADD COLUMN search_vector tsvector
    GENERATED ALWAYS AS (
        to_tsvector('english', coalesce(original_filename, '') || ' ' ||
                               coalesce(text_content, ''))
    ) STORED;

CREATE INDEX idx_documents_search ON documents USING GIN (search_vector);
```

### PostgreSQL: websearch_to_tsquery Examples
```sql
-- Source: https://www.postgresql.org/docs/current/textsearch-controls.html
-- Simple terms (automatic AND)
SELECT websearch_to_tsquery('english', 'fat cat');
-- Result: 'fat' & 'cat'

-- Quoted phrase (word order matters)
SELECT websearch_to_tsquery('english', '"fat cat"');
-- Result: 'fat' <-> 'cat'

-- OR operator
SELECT websearch_to_tsquery('english', 'cat or dog');
-- Result: 'cat' | 'dog'

-- Exclude terms
SELECT websearch_to_tsquery('english', 'cat -dog');
-- Result: 'cat' & !'dog'
```

### PostgreSQL: ts_headline for Snippets
```sql
-- Source: https://www.postgresql.org/docs/current/textsearch-controls.html
SELECT
    ts_headline('english', text_content,
        websearch_to_tsquery('english', 'search terms'),
        'MaxFragments=1, MaxWords=30, MinWords=15, StartSel=<mark>, StopSel=</mark>'
    ) as headline
FROM documents
WHERE search_vector @@ websearch_to_tsquery('english', 'search terms')
LIMIT 20;
```

### HTMX: Active Search with Debounce
```html
<!-- Source: https://htmx.org/examples/active-search/ -->
<input type="search" name="q"
       hx-get="/documents"
       hx-trigger="input changed delay:500ms, keyup[key=='Enter']"
       hx-target="#document-results"
       hx-push-url="true"
       hx-indicator=".search-spinner"
       placeholder="Search documents...">
```

### sqlc: Optional Filter Pattern
```sql
-- Source: https://dizzy.zone/2024/07/03/SQLC-dynamic-queries/
-- Boolean flag pattern for optional WHERE clauses
SELECT * FROM documents
WHERE
    -- Mandatory condition
    processing_status = 'completed'
    -- Optional filters using boolean flags
    AND (NOT @has_date_from::boolean OR document_date >= @date_from)
    AND (NOT @has_correspondent::boolean OR correspondent_id = @correspondent_id);
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Manual tsvector column + triggers | GENERATED ALWAYS AS ... STORED | PG 12 (2019) | Simpler maintenance, automatic updates |
| plainto_tsquery for user input | websearch_to_tsquery | PG 11 (2018) | Handles quotes, OR, negation naturally |
| Expression indexes for FTS | Generated column + index | PG 12 (2019) | Clearer code, same performance |
| LIKE '%term%' for search | Full-text search | Always existed | Orders of magnitude faster with index |

**Deprecated/outdated:**
- Using triggers to maintain tsvector columns (use generated columns instead)
- Using `textsearch` contrib module (fully integrated into core PostgreSQL)

## Recommendations for Discretionary Items

Based on research, here are recommendations for items marked as Claude's Discretion:

### Search Trigger Pattern
**Recommendation:** Instant search with 500ms debounce + Enter key
- `hx-trigger="input changed delay:500ms, keyup[key=='Enter']"`
- Provides responsive feel while preventing excessive requests
- Enter key provides explicit submission option

### URL State for Search
**Recommendation:** Yes, use `hx-push-url="true"`
- Enables shareable/bookmarkable searches
- Browser back/forward works naturally
- Server must handle both HTMX and direct requests (check HX-Request header)

### Text Snippets in Results
**Recommendation:** Yes, show snippets using ts_headline
- Show 30-word excerpt with highlighted match
- Use `<mark>` tags for highlighting (CSS-styleable)
- Only compute for displayed results (apply LIMIT first)

### Pagination vs Load More
**Recommendation:** Traditional pagination with page numbers
- Better for search results (users want to jump to specific pages)
- Load more/infinite scroll better for browsing, not searching
- Use offset-based pagination (simpler, adequate for this scale)

### Loading State Approach
**Recommendation:** Subtle spinner in search input
- Use `hx-indicator` pointing to spinner inside input container
- Don't disable input during search (user can continue typing)
- Results area can show subtle loading state via CSS

### Date Field for Filter
**Recommendation:** Use `document_date` (not `created_at`/upload date)
- Users think in terms of document dates, not upload dates
- `document_date` is the semantic date of the document content
- Filter presets: Today, Last 7 days, Last 30 days, Last year (relative to document_date)

### Processing Documents in Search
**Recommendation:** Include but mark visually
- Include documents with `processing_status != 'completed'` in results
- Show processing indicator badge (reuse existing status partial)
- Text search only matches processed documents (search_vector is NULL for pending)
- Metadata filters (tags, correspondent, date) work regardless of status

## Open Questions

Things that couldn't be fully resolved:

1. **Tag multi-select UI pattern**
   - What we know: Need multi-select for AND logic filtering
   - What's unclear: Best UI for selecting multiple tags inline
   - Recommendation: Use existing tag picker pattern, show selected tags as removable chips

2. **Match reason display complexity**
   - What we know: User wants "Matched: content" vs "Matched: tag 'receipts'"
   - What's unclear: How to efficiently determine match reason when multiple filters active
   - Recommendation: Check if query is non-empty (content match) or show "Filter match" for filter-only

## Sources

### Primary (HIGH confidence)
- [PostgreSQL 18 Documentation: Tables and Indexes](https://www.postgresql.org/docs/current/textsearch-tables.html) - Generated columns, GIN indexes
- [PostgreSQL 18 Documentation: Controlling Text Search](https://www.postgresql.org/docs/current/textsearch-controls.html) - Query functions, ts_headline, ranking
- [HTMX Examples: Active Search](https://htmx.org/examples/active-search/) - Debounced search pattern
- [HTMX Documentation: hx-push-url](https://htmx.org/attributes/hx-push-url/) - URL state management

### Secondary (MEDIUM confidence)
- [sqlc Macros Documentation](https://docs.sqlc.dev/en/stable/reference/macros.html) - sqlc.narg, sqlc.slice
- [SQLC & Dynamic Queries](https://dizzy.zone/2024/07/03/SQLC-dynamic-queries/) - Boolean flag pattern
- [pganalyze: GIN Index Deep Dive](https://pganalyze.com/blog/gin-index) - GIN index behavior

### Tertiary (LOW confidence)
- Various blog posts on PostgreSQL FTS patterns - cross-verified with official docs

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - All built-in PostgreSQL features with official documentation
- Architecture: HIGH - Patterns verified in PostgreSQL and HTMX official docs
- Pitfalls: HIGH - Documented in official PostgreSQL documentation and verified via multiple sources
- Discretionary recommendations: MEDIUM - Based on best practices, not hard requirements

**Research date:** 2026-02-03
**Valid until:** 2026-03-03 (30 days - PostgreSQL FTS is stable, HTMX patterns stable)
