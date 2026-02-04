---
created: 2026-02-03T21:36
title: Fix correspondents page edit button not working
area: ui
files:
  - templates/pages/admin/correspondents.templ:456-462
  - templates/pages/admin/correspondents.templ:491-492
  - templates/pages/admin/correspondents.templ:186-220
---

## Problem

The edit button on the correspondents page (`/correspondents`) does nothing when clicked. Users cannot edit existing correspondents.

**Expected:** Clicking edit button opens the correspondent modal pre-filled with name and notes for editing.

**Actual:** Nothing happens.

## Relevant Code

Same pattern as tags page (see `2026-02-03-fix-tags-edit-button.md`).

**Edit button** (lines 456-462):
```templ
@button.Button(button.Props{...
    Attributes: templ.Attributes{
        "onclick": editCorrespondentOnClick(correspondent),
        "title":   "Edit correspondent",
    },
})
```

**Script** (lines 491-492):
```templ
script editCorrespondentOnClick(correspondent sqlc.ListCorrespondentsWithCountsRow) {
    openEditMode(correspondent.ID, correspondent.Name, correspondent.Notes || '');
}
```

## Solution

Fix alongside tags edit button - likely same root cause (templ Script() loading or UUID serialization issue).
