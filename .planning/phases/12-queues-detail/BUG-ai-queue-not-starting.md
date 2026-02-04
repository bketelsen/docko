# Bug: AI Queue Workers Never Start

## Summary
AI analysis jobs are queued but never processed because the AI queue workers don't start.

## Symptoms
- Log shows `ai analysis queued doc_id=...` but no subsequent AI processing logs
- Jobs table shows AI jobs stuck in `pending` status indefinitely
- Server startup shows `queue starting queue=default workers=4` but NO `queue starting queue=ai`

## Root Cause
The `Queue` struct in `internal/queue/queue.go` uses a single shared `running` flag for all queues.

**Problem code** (`queue.go:130-134`):
```go
if q.running {
    q.mu.Unlock()
    return  // Returns immediately if already running
}
q.running = true
```

**Startup sequence** (`main.go:85-86`):
```go
q.Start(queueCtx, document.QueueDefault)  // Sets q.running = true
go q.Start(queueCtx, processing.QueueAI)  // Sees q.running=true, exits immediately
```

The second `Start()` call for the AI queue sees `running=true` (set by the default queue) and returns without starting any workers.

## Evidence
```sql
SELECT queue_name, status, COUNT(*) FROM jobs GROUP BY queue_name, status;
-- Shows: ai queue has pending jobs, default queue jobs complete
```

## Fix
Change from global `running` flag to per-queue tracking:

```go
type Queue struct {
    // ...
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

## Affected Files
- `internal/queue/queue.go` - Queue struct and Start method
- `cmd/server/main.go` - Queue initialization (no changes needed after fix)

## Test Plan
1. Start server, verify both queues log startup messages
2. Upload a document with AI auto-process enabled
3. Verify AI job transitions from `pending` → `processing` → `completed`
4. Check `ai_suggestions` table for results
