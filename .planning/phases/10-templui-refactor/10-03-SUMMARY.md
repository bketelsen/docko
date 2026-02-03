---
phase: 10-templui-refactor
plan: 03
subsystem: ui-forms
tags: [templui, dialog, forms, tags, correspondents, documents]

dependency_graph:
  requires: [10-01, 10-02]
  provides: [templUI forms for Tags/Correspondents/Documents pages]
  affects: []

tech_stack:
  added: []
  patterns:
    - templUI dialog for modal windows
    - templUI button for all action buttons
    - templUI input for form fields
    - templUI label for form labels
    - Consistent select styling matching templUI

key_files:
  created: []
  modified:
    - templates/pages/admin/tags.templ
    - templates/pages/admin/correspondents.templ
    - templates/pages/admin/documents.templ

decisions:
  - Keep JavaScript modal open/close logic compatible with templUI dialog data attributes
  - Use HideCloseButton with custom Cancel button in dialog footer for consistency
  - Style native select elements to match templUI input component styling
  - Preserve HTMX form submission patterns within templUI dialog components

metrics:
  duration: 4 min
  completed: 2026-02-03
---

# Phase 10 Plan 03: Tags, Correspondents, Documents Forms Summary

**One-liner:** Migrated Tags/Correspondents pages to templUI dialog and form components, Documents page to templUI input/button.

## What Changed

### Tags Page (tags.templ)
- Replaced custom modal with templUI dialog component
- Used templUI input and label for tag name field
- Replaced action buttons (Add, Edit, Delete) with templUI button component
- Added dialog.Script() and input.Script() for component functionality
- Preserved create/edit mode switching via JavaScript
- Maintained HTMX form submission and modal close behavior

### Correspondents Page (correspondents.templ)
- Replaced custom modal with templUI dialog component
- Used templUI input and label for correspondent name field
- Replaced all action buttons with templUI button component
- Styled merge target select to match templUI input styling
- Preserved merge mode functionality with templUI button styling
- Maintained HTMX form submission and merge operations

### Documents Page (documents.templ)
- Replaced search input with templUI input component (TypeSearch)
- Replaced Upload link button with templUI button component (Href variant)
- Styled correspondent and date filter dropdowns to match templUI
- Updated tag filter checkboxes with improved border styling
- Added input.Script() for component functionality
- Preserved all HTMX search/filter interactions

## Technical Approach

### Dialog Integration
The templUI dialog uses data attributes for state management:
- `data-tui-dialog-open="true/false"` controls visibility
- `data-tui-dialog-hidden="true"` prevents display during transitions
- JavaScript functions manipulate these attributes for edit mode

### Form Mode Switching
Both Tags and Correspondents pages support create/edit modes:
- JavaScript functions configure form attributes before dialog opens
- HTMX attributes (hx-post, hx-target, hx-swap) are dynamically updated
- htmx.process() re-processes form after attribute changes

### Component Scripts
Each page includes necessary script tags:
- `@dialog.Script()` - dialog open/close behavior
- `@input.Script()` - input field enhancements

## Deviations from Plan

None - plan executed exactly as written.

## Commits

| Hash | Type | Description |
|------|------|-------------|
| cf4af03 | feat | Refactor Tags page with templUI dialog and form components |
| 5d6b92c | feat | Refactor Correspondents page with templUI dialog and form components |
| 3164cf1 | feat | Refactor Documents page with templUI input and button components |

## Next Phase Readiness

All pages in this plan now use templUI components:
- Tags page: dialog, button, input, label
- Correspondents page: dialog, button, input, label
- Documents page: button, input

The codebase is ready for the next plan in the phase (10-04).
