# Phase 6: Search - Context

**Gathered:** 2026-02-03
**Status:** Ready for planning

<domain>
## Phase Boundary

Full-text search with filtering by tags, correspondent, and date range. Users can find any document by content or metadata. Search integrates into the existing documents page. Advanced search features (saved searches, search history) are out of scope.

</domain>

<decisions>
## Implementation Decisions

### Search interface
- Search input and filters live on the Documents page (not a separate page)
- Clear button in search input AND visible filter chips showing active filters
- Chips are removable to clear individual filters

### Results display
- Reuse existing document list layout (not a new layout for search)
- Show match reason indicator ("Matched: content" or "Matched: tag 'receipts'")

### Filter experience
- Filters inline with search input (dropdowns/buttons next to search bar)
- Multi-tag filtering with AND logic (documents must have ALL selected tags)
- Date filter uses preset ranges (Today, Last 7 days, Last 30 days, Last year)

### Empty & edge states
- No results: show helpful guidance with active filters listed and suggestion to remove some
- Empty library: show onboarding message prompting to upload or set up inbox

### Claude's Discretion
- Search trigger pattern (instant vs submit)
- URL state for search (shareable URLs vs local state)
- Text snippets in results (whether to show excerpt where term was found)
- Pagination vs load more for many results
- Loading state approach
- Which date field the filter applies to (document date vs upload date)
- How to handle documents still being processed in search results

</decisions>

<specifics>
## Specific Ideas

No specific requirements — open to standard approaches

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 06-search*
*Context gathered: 2026-02-03*
