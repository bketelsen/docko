---
created: 2026-02-03T21:32
title: Add processing progress visibility
area: queue
files:
  - internal/processing/processor.go:57-185
  - internal/queue/queue.go
  - templates/pages/queues.templ
---

## Problem

"Processing" status is opaque. Users see a document is "processing" but have no visibility into:

1. **Current step** - Is it extracting text? Running OCR? Generating thumbnail?
2. **Progress** - How far along is it? What's left?
3. **Stuck detection** - Has it been stuck for too long? Where did it hang?

**Evidence:** Two jobs stuck in "processing" for 20+ minutes with no indication of what's wrong:

```
job_id: f6dca96f-bc26-465c-b940-5449acdabfd4
doc_id: 6f79ca1d-d4d8-4a1a-87ac-3ece0d79334e
duration: 00:20:16

job_id: 71948493-f7f3-4855-9bc7-e9afaaf47f23
doc_id: c72705cc-01f5-47c3-a655-f9a84dc908e5
duration: 00:20:15
```

**Processing pipeline stages** (from processor.go):
1. `processing document` (line 57)
2. `text extracted` / `extracted embedded text` / `extracted text via OCR` (lines 52-99)
3. `thumbnail generated` (line 131)
4. `document processing complete` (line 185)

Each stage logs but this info isn't surfaced to UI.

## Solution

Options (pick one or combine):

**A. Add `current_step` column to jobs table:**
```sql
ALTER TABLE jobs ADD COLUMN current_step VARCHAR(50);
-- Values: 'starting', 'extracting_text', 'running_ocr', 'generating_thumbnail', 'finalizing'
```
Update step at each stage in processor.go. Display in queues UI.

**B. Add processing substatus to document:**
```sql
ALTER TABLE documents ADD COLUMN processing_step VARCHAR(50);
```

**C. Use job metadata/payload:**
Store current step in job payload JSONB, update as processing progresses.

**D. Stuck detection:**
- Flag jobs with status='processing' and started_at > 5 minutes ago
- Show warning badge in UI
- Add "Retry" or "Cancel" actions for stuck jobs

**E. SSE progress updates:**
Broadcast step changes via existing SSE infrastructure for real-time UI updates.
