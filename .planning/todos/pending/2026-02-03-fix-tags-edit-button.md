---
created: 2026-02-03T21:35
title: Fix tags page edit button not working
area: ui
files:
  - templates/pages/admin/tags.templ:238-246
  - templates/pages/admin/tags.templ:328-330
  - templates/pages/admin/tags.templ:159-195
---

## Problem

The edit button on the tags page (`/tags`) does nothing when clicked. Users cannot edit existing tags.

**Expected:** Clicking edit button opens the tag modal pre-filled with the tag's name and color for editing.

**Actual:** Nothing happens.

## Relevant Code

**Edit button** (lines 238-246):
```templ
@button.Button(button.Props{...
    Attributes: templ.Attributes{
        "onclick": editTagOnClick(tag),
        "title":   "Edit tag",
    },
})
```

**Script** (lines 328-330):
```templ
script editTagOnClick(tag sqlc.ListTagsWithCountsRow) {
    openEditMode(tag.ID, tag.Name, tag.Color || 'blue');
}
```

**openEditMode function** (lines 159-195): Sets form attributes, selects color, and opens dialog via data attributes.

## Likely Causes

1. **templ Script() not loaded** - The `editTagOnClick` script might not be rendered or loaded
2. **Dialog Script() missing** - templUI dialog requires `@dialog.Script()` to handle open/close
3. **JavaScript error** - openEditMode might be throwing an error (check browser console)
4. **UUID serialization** - `tag.ID` is a UUID which might not serialize correctly in the script

## Debug Steps

1. Check browser console for JavaScript errors
2. Verify `editTagOnClick` function exists in page source
3. Check if dialog.Script() is included in the template
4. Test if openEditMode() works when called directly from console
