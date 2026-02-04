---
created: 2026-02-03T21:19
title: Replace magic strings with Go constants
area: refactoring
files:
  - internal/processing/status.go
  - internal/processing/processor.go
  - internal/processing/ai_processor.go
---

## Problem

Codebase uses hardcoded "magic strings" for status values, event types, and other repeated literals. Examples:

- SSE event types: `"ai_complete"`, `"ai_processing"`, `"ai_failed"`, `"processing"`, `"complete"`
- Job statuses: `"pending"`, `"processing"`, `"completed"`, `"failed"`
- Queue names: `"default"`, `"ai"`

This creates risk of typos, makes refactoring harder, and reduces IDE support for find-all-references.

## Solution

Define Go `const` blocks for each category of strings:

```go
// internal/processing/events.go
const (
    EventAIComplete    = "ai_complete"
    EventAIProcessing  = "ai_processing"
    EventAIFailed      = "ai_failed"
    EventProcessing    = "processing"
    EventComplete      = "complete"
)

// internal/queue/queue.go
const (
    StatusPending    = "pending"
    StatusProcessing = "processing"
    StatusCompleted  = "completed"
    StatusFailed     = "failed"
)
```

Then replace all literal usages with constants.
