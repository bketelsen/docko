---
status: diagnosed
trigger: "Tags Edit Button Not Working - clicking edit icon doesn't do anything"
created: 2026-02-04T00:00:00Z
updated: 2026-02-04T00:00:00Z
symptoms_prefilled: true
goal: find_root_cause_only
---

## Current Focus

hypothesis: CONFIRMED - templ.JSFuncCall returns ComponentScript, but templ.RenderAttributes doesn't handle ComponentScript type
test: Examined rendered HTML and templ source code
expecting: onclick attribute missing from rendered HTML
next_action: Return diagnosis - fix requires using .Call property or different approach

## Symptoms

expected: Clicking edit icon on /tags page should open edit form or trigger edit action
actual: Clicking edit icon does nothing
errors: Unknown - need to infer from code
reproduction: Go to /tags page, click edit icon on any tag
started: After Phase 15-01 implementation (supposed to fix this)

## Eliminated

## Evidence

- timestamp: 2026-02-04T15:30:00Z
  checked: Rendered HTML for /tags page
  found: Edit button has NO onclick attribute - only class, type, title
  implication: The templ.JSFuncCall value is being silently ignored

- timestamp: 2026-02-04T15:31:00Z
  checked: Delete button for comparison
  found: Delete button correctly has all hx-* attributes (hx-delete, hx-confirm, hx-swap, hx-target)
  implication: String attribute values work, but ComponentScript type does not

- timestamp: 2026-02-04T15:32:00Z
  checked: templ v0.3.977 runtime.go RenderAttributes function (lines 459-529)
  found: Switch statement handles string, *string, bool, *bool, numerics, KeyValue types - NO case for ComponentScript
  implication: ComponentScript values in Attributes map are silently dropped

- timestamp: 2026-02-04T15:33:00Z
  checked: templ.JSFuncCall documentation
  found: Returns ComponentScript with .Call field (HTML-escaped) and .CallInline field (for script tags)
  implication: Code should use .Call property directly as string, not pass ComponentScript to Attributes

## Resolution

root_cause: templ.JSFuncCall returns ComponentScript struct, but when passed to button.Props.Attributes map, templ's RenderAttributes function has no case to handle ComponentScript type - it silently ignores the attribute. The .Call property which contains the properly escaped function call string is never accessed.
fix:
verification:
files_changed: []
