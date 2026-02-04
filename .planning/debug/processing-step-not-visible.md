---
status: investigating
trigger: "Processing Step Not Visible on Upload - user only sees toast and green 'done' bar, no step progression"
created: 2026-02-04T10:00:00Z
updated: 2026-02-04T10:00:00Z
---

## Current Focus

hypothesis: CurrentStep is added to StatusUpdate struct and processor calls updateStep(), but the step info is never passed to the UI - it's dropped in the SSE handler and the DocumentStatus partial doesn't accept or display it
test: Traced data flow from processor.go -> status.go -> handler/status.go -> partials/document_status.templ
expecting: Find where CurrentStep is dropped
next_action: Report root cause

## Symptoms

expected: User should see processing steps (starting -> extracting_text -> generating_thumbnail -> finalizing) during document processing
actual: User only sees toast notification and green "Done" bar, no step progression visible
errors: None
reproduction: Upload a PDF on /upload page, watch progress
started: After phase 15-03 added step tracking

## Eliminated

(none yet - root cause found on first investigation)

## Evidence

- timestamp: 2026-02-04T10:01:00Z
  checked: /home/bjk/projects/corpus/docko/internal/processing/processor.go
  found: updateStep() is called at lines 78, 84, 119, 139 with steps "starting", "extracting_text", "generating_thumbnail", "finalizing"
  implication: Backend is correctly tracking and broadcasting step changes

- timestamp: 2026-02-04T10:02:00Z
  checked: /home/bjk/projects/corpus/docko/internal/processing/status.go
  found: StatusUpdate struct has CurrentStep field (line 38), Broadcast() sends complete update (line 108)
  implication: Broadcasting infrastructure includes step info

- timestamp: 2026-02-04T10:03:00Z
  checked: /home/bjk/projects/corpus/docko/internal/handler/status.go
  found: Lines 74-78 call partials.DocumentStatus() with only 3 args: update.DocumentID.String(), update.Status, update.Error - DOES NOT pass update.CurrentStep
  implication: CurrentStep is DROPPED here - never sent to partial

- timestamp: 2026-02-04T10:04:00Z
  checked: /home/bjk/projects/corpus/docko/templates/partials/document_status.templ
  found: Function signature (line 10) only accepts 3 params: docID, status, errorMsg. No parameter for step. Line 17-19 shows "Processing..." text with no step display.
  implication: Partial has no mechanism to display step info even if it received it

- timestamp: 2026-02-04T10:05:00Z
  checked: /home/bjk/projects/corpus/docko/static/js/upload.js
  found: upload.js handles XHR upload progress (lines 148-152) and completion (lines 156-196), but has NO SSE subscription for processing status events
  implication: Upload page doesn't even subscribe to SSE - only shows upload progress, not processing progress

## Resolution

root_cause: TWO ISSUES:
1. SSE handler drops CurrentStep - handler/status.go line 74-78 passes Status and Error but NOT CurrentStep to DocumentStatus partial
2. Upload page doesn't subscribe to SSE - upload.js only tracks HTTP upload progress, not backend processing progress. The green "Done" bar is the upload completion, not processing completion.
3. DocumentStatus partial lacks step parameter - even if step was passed, the partial can't display it

fix: (to be applied)
verification: (to be verified)
files_changed: []
