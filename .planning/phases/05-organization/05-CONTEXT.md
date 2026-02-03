# Phase 5: Organization - Context

**Gathered:** 2026-02-03
**Status:** Ready for planning

<domain>
## Phase Boundary

Tag and correspondent management for documents. Users can create, edit, and delete tags/correspondents, assign them to documents, and merge duplicate correspondents. Search and filtering are Phase 6.

</domain>

<decisions>
## Implementation Decisions

### Tag design
- User-picked colors from preset palette (~12 curated colors)
- Flat list structure (no hierarchy/nesting)
- Free text naming, but names must be unique

### Assignment workflow
- Assign tags/correspondents from both document detail page and document list view
- Dropdown with search for tag/correspondent picker
- Can create new tags inline while assigning (type name, create if not found)
- Same workflow for correspondents

### Correspondent model
- One correspondent per document (1:1 relationship, already in database)
- Correspondent has name + optional notes field
- Bulk merge: select multiple correspondents, merge all into one target
- When merging, append all notes from merged correspondents

### Management UI
- Quick access in sidebar navigation (Tags, Correspondents links)
- Full management also in Settings area
- Modal dialog for create/edit operations
- Show document count next to each tag/correspondent in lists

### Claude's Discretion
- Delete confirmation UX (show count vs block if in use)
- Exact color palette choices
- Sidebar vs settings page feature parity
- Modal styling and form layout

</decisions>

<specifics>
## Specific Ideas

No specific product references mentioned — open to standard patterns.

</specifics>

<deferred>
## Deferred Ideas

- Bulk tag assignment (select multiple documents, apply tags to all) — future enhancement

</deferred>

---

*Phase: 05-organization*
*Context gathered: 2026-02-03*
