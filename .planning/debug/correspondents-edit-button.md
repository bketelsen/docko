---
status: diagnosed
trigger: "Correspondents Edit Button Not Working - clicking edit icon doesn't do anything"
created: 2026-02-04T00:00:00Z
updated: 2026-02-04T00:00:00Z
---

## Current Focus

hypothesis: CONFIRMED - templ.JSFuncCall returns ComponentScript struct, not string
test: Rendered output test showing onclick attribute is dropped
expecting: onclick attribute missing from rendered HTML
next_action: Return diagnosis to caller

## Symptoms

expected: Clicking edit icon should open edit modal or trigger edit action
actual: Clicking edit icon does nothing
errors: None reported (silent failure)
reproduction: Go to /correspondents page, click edit icon
started: After Phase 15-01 implementation

## Eliminated

## Evidence

- timestamp: 2026-02-04T00:01:00Z
  checked: correspondents.templ line 461
  found: Uses templ.JSFuncCall directly in templ.Attributes
  implication: Need to check if this is correct usage

- timestamp: 2026-02-04T00:02:00Z
  checked: templ.JSFuncCall documentation
  found: Returns ComponentScript struct with .Call field for HTML-escaped JS
  implication: Must use .Call to get string for attributes

- timestamp: 2026-02-04T00:03:00Z
  checked: templ.RenderAttributes behavior with ComponentScript
  found: When ComponentScript passed to Attributes, onclick is SILENTLY DROPPED
  implication: The edit button has NO onclick attribute in rendered HTML

## Resolution

root_cause: templ.JSFuncCall() returns a ComponentScript struct, but when passed directly to templ.Attributes, the templ library silently drops the attribute because it doesn't know how to render the struct. The correct usage is templ.JSFuncCall(...).Call to get the HTML-escaped JavaScript call string.

fix: Change line 461 from:
  "onclick": templ.JSFuncCall("openEditMode", ...)
to:
  "onclick": templ.JSFuncCall("openEditMode", ...).Call

Same fix needed for tags.templ line 243.

verification:
files_changed: []
