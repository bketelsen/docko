---
created: 2026-02-03T21:17
title: Fix AI queue workers not starting
area: queue
files:
  - internal/queue/queue.go:130-134
  - cmd/server/main.go:85-86
---

## Problem

The `Queue` struct uses a single shared `running` flag for all queues. When `Start()` is called for the default queue, it sets `running = true`. Then when `Start()` is called for the AI queue, it checks `if q.running` and returns immediately without starting any workers.

**Result:** AI analysis jobs are queued but never processed. Jobs sit in `pending` status indefinitely.

**Evidence:**
- Server logs show `queue starting queue=default workers=4` but no `queue starting queue=ai`
- Jobs table shows AI jobs stuck in `pending` status
- Log shows `ai analysis queued` but no subsequent AI processing logs

**Code at queue.go:130-134:**
```go
if q.running {
    q.mu.Unlock()
    return  // Returns immediately if already running
}
q.running = true
```

**Startup at main.go:85-86:**
```go
q.Start(queueCtx, document.QueueDefault)  // Sets q.running = true
go q.Start(queueCtx, processing.QueueAI)  // Sees q.running=true, exits immediately
```

## Solution

Change from global `running` flag to per-queue tracking:

```go
type Queue struct {
    running map[string]bool  // Track running state per queue name
}

func (q *Queue) Start(ctx context.Context, queueName string) {
    q.mu.Lock()
    if q.running[queueName] {
        q.mu.Unlock()
        return
    }
    q.running[queueName] = true
    q.mu.Unlock()
    // ... start workers
}
```

Also need to update `Stop()` to handle per-queue shutdown.

**Detailed bug doc:** .planning/phases/12-queues-detail/BUG-ai-queue-not-starting.md
