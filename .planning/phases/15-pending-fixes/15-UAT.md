---
status: diagnosed
phase: 15-pending-fixes
source: [15-01-SUMMARY.md, 15-02-SUMMARY.md, 15-03-SUMMARY.md]
started: 2026-02-04T15:30:00Z
updated: 2026-02-04T15:45:00Z
---

## Current Test

[testing complete]

## Tests

### 1. Edit Tag Button

expected: On Tags page (/tags), clicking "Edit" on any tag opens the edit modal with the tag's current name pre-filled in the input field.
result: issue
reported: "first, /admin/tags isn't a route. there is /tags, though. And clicking on the edit icon doesn't do anything."
severity: major

### 2. Edit Correspondent Button

expected: On Correspondents page (/correspondents), clicking "Edit" on any correspondent opens the edit modal with the correspondent's current name and notes pre-filled.
result: issue
reported: "same problem with correspondents, clicking on the edit icon doesn't do anything"
severity: major

### 3. Inbox Error Count Badge

expected: On Inboxes page (/inboxes), if an inbox has PDF files in its error directory, a red badge appears next to the inbox name showing the count (e.g., "3 errors").
result: pass

### 4. Inbox Error Path Count

expected: On Inboxes page (/inboxes), the "Error Path" info section shows the error directory path with file count displayed (e.g., "/path/to/errors (3 files)").
result: pass

### 5. Processing Step in Status

expected: When uploading a new PDF via /upload, the processing status shows current step progressing through: "starting" → "extracting_text" → "generating_thumbnail" → "finalizing" → completed.
result: issue
reported: "there is no processing status on the /upload page. It just has a toast and a green bar at the bottom that says done. It MAY be happening, but too quickly to see it?"
severity: minor

## Summary

total: 5
passed: 2
issues: 3
pending: 0
skipped: 0

## Gaps

- truth: "On Tags page (/tags), clicking Edit opens the edit modal with tag's name pre-filled"
  status: failed
  reason: "User reported: first, /admin/tags isn't a route. there is /tags, though. And clicking on the edit icon doesn't do anything."
  severity: major
  test: 1
  root_cause: "templ.JSFuncCall() returns ComponentScript struct, but when passed to templ.Attributes map, templ silently drops it because RenderAttributes has no case for ComponentScript type. Need to use .Call property to get the escaped string."
  artifacts:
    - path: "templates/pages/admin/tags.templ"
      line: 243
      issue: "Uses templ.JSFuncCall(...) directly instead of .Call property"
  missing:
    - "Add .Call to templ.JSFuncCall() to get string value for onclick attribute"
  debug_session: ".planning/debug/tags-edit-button.md"

- truth: "On Correspondents page (/correspondents), clicking Edit opens the edit modal with correspondent's name and notes pre-filled"
  status: failed
  reason: "User reported: same problem with correspondents, clicking on the edit icon doesn't do anything"
  severity: major
  test: 2
  root_cause: "Same issue as tags - templ.JSFuncCall() returns ComponentScript struct which is silently dropped. Need to use .Call property."
  artifacts:
    - path: "templates/pages/admin/correspondents.templ"
      line: 461
      issue: "Uses templ.JSFuncCall(...) directly instead of .Call property"
  missing:
    - "Add .Call to templ.JSFuncCall() to get string value for onclick attribute"
  debug_session: ".planning/debug/correspondents-edit-button.md"

- truth: "Processing status shows current step progressing through steps during upload"
  status: failed
  reason: "User reported: there is no processing status on the /upload page. It just has a toast and a green bar at the bottom that says done. It MAY be happening, but too quickly to see it?"
  severity: minor
  test: 5
  root_cause: "Three interconnected issues: (1) upload.js doesn't subscribe to SSE events for processing status, (2) SSE handler in status.go drops CurrentStep field when rendering partial, (3) DocumentStatus partial has no parameter or UI for step display"
  artifacts:
    - path: "static/js/upload.js"
      issue: "No SSE subscription for processing events - only tracks HTTP upload progress"
    - path: "internal/handler/status.go"
      line: 74-78
      issue: "Drops CurrentStep when calling partials.DocumentStatus()"
    - path: "templates/partials/document_status.templ"
      line: 10
      issue: "Only accepts 3 params (docID, status, errorMsg) - no currentStep"
  missing:
    - "Add currentStep parameter to DocumentStatus partial"
    - "Update SSE handler to pass update.CurrentStep to partial"
    - "Add step display UI in partial when status is processing"
    - "Add SSE subscription to upload.js after upload succeeds"
  debug_session: ".planning/debug/processing-step-not-visible.md"
