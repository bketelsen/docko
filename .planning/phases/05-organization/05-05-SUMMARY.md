---
phase: 05-organization
plan: 05
subsystem: organization
tags: [correspondent, htmx, ui, document-management]
dependency-graph:
  requires: ["05-02", "05-03"]
  provides: ["correspondent-assignment-ui", "document-correspondent-picker"]
  affects: ["06-search"]
tech-stack:
  added: []
  patterns: ["HTMX partial updates", "inline create pattern", "document picker component"]
key-files:
  created:
    - templates/partials/correspondent_picker.templ
  modified:
    - sqlc/queries/correspondents.sql
    - sqlc/queries/documents.sql
    - internal/handler/correspondents.go
    - internal/handler/documents.go
    - internal/handler/handler.go
    - templates/pages/admin/document_detail.templ
    - templates/pages/admin/documents.templ
decisions:
  - id: correspondent-picker-pattern
    choice: "Same pattern as tag picker but for 1:1 relationship"
    reason: "Consistency with existing tag picker UI"
metrics:
  duration: "4 min"
  completed: "2026-02-03"
---

# Phase 05 Plan 05: Correspondent Assignment UI Summary

Correspondent picker with searchable dropdown and inline creation for document-correspondent assignment.

## What Was Built

### Correspondent Picker Component (`templates/partials/correspondent_picker.templ`)
- `CorrespondentPicker` - Full picker with current correspondent display and search
- `CorrespondentDisplay` - Shows assigned correspondent with Change/Remove buttons
- `CorrespondentEmptyState` - Shows "No correspondent assigned" with Assign button
- `CorrespondentSearchInput` - Search input with 300ms debounce
- `CorrespondentSearchResults` - Dropdown with matching correspondents and "Create" option
- `InlineCorrespondentDisplay` - Compact display for document list view

### Assignment Handlers (`internal/handler/correspondents.go`)
- `SearchCorrespondentsForDocument` - Wildcard search with "Create new" option
- `SetDocumentCorrespondent` - Assign/replace with inline creation support
- `RemoveDocumentCorrespondent` - Unassign correspondent from document
- `GetDocumentCorrespondent` - Fetch current assignment

### SQL Queries Added
- `GetDocumentCorrespondent` - Get correspondent for a document
- `SetDocumentCorrespondent` - Upsert document-correspondent relationship
- `RemoveDocumentCorrespondent` - Delete relationship
- `SearchCorrespondentsWithLimit` - Search with ILIKE pattern
- `ListDocumentsWithCorrespondent` - Documents list with correspondent join

### Page Integrations
- Document detail page: Correspondent section added to Overview tab
- Document list page: Correspondent column showing assigned name

## Key Implementation Details

1. **1:1 Relationship Enforcement**: Database constraint ensures one correspondent per document via `ON CONFLICT (document_id) DO UPDATE`

2. **HTMX Partial Updates**: All operations use partial swaps without full page reloads:
   - Search results update dropdown only
   - Assignment updates correspondent display only
   - Remove returns empty state partial

3. **Inline Creation**: When search query has no exact match, "Create" option appears and creates correspondent + assigns in single request

4. **Efficient List Query**: `ListDocumentsWithCorrespondent` uses LEFT JOIN to fetch correspondent info in single query

## Commits

| Commit | Description |
|--------|-------------|
| 665e5a9 | Add document-correspondent SQL queries |
| ed66383 | Add correspondent assignment handlers |
| eb51ca2 | Add correspondent picker component and integrate into pages |

## Deviations from Plan

None - plan executed exactly as written.

## Verification Checklist

- [x] Correspondent picker component is reusable (171 lines)
- [x] Search filters existing correspondents with 300ms debounce
- [x] Can assign existing correspondent to document
- [x] Can change document's correspondent to a different one
- [x] Can remove correspondent from document
- [x] Can create and assign new correspondent inline
- [x] Document detail page shows current correspondent
- [x] Document list shows correspondent on cards
- [x] All operations use HTMX partial updates
- [x] Only one correspondent per document (enforced by database)

## Next Phase Readiness

Phase 05 Organization complete. All plans executed:
- 05-01: Tag CRUD and management
- 05-02: Correspondent CRUD and management
- 05-03: Correspondent merge functionality
- 05-04: Document tag assignment UI
- 05-05: Correspondent assignment UI (this plan)

Ready to proceed to Phase 06 (Search).
