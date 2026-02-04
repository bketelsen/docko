---
phase: quick
plan: 001
type: execute
wave: 1
depends_on: []
files_modified:
  - internal/queue/queue.go
  - internal/processing/status.go
  - internal/processing/processor.go
  - internal/processing/ai_processor.go
  - templates/pages/admin/inboxes.templ
  - internal/inbox/service.go
autonomous: true
---

<objective>
Fix three pending bugs: AI queue workers not starting due to shared running flag, magic strings in processing package, and missing inbox error directory visibility.

Purpose: Restore AI queue functionality (critical), improve code maintainability, and improve UX for inbox error monitoring.
Output: Working multi-queue system, cleaner constants, visible error counts on inbox cards.
</objective>

<execution_context>
@/home/bjk/.claude/get-shit-done/workflows/execute-plan.md
@/home/bjk/.claude/get-shit-done/templates/summary.md
</execution_context>

<context>
@internal/queue/queue.go
@internal/processing/status.go
@internal/processing/processor.go
@internal/processing/ai_processor.go
@templates/pages/admin/inboxes.templ
@internal/inbox/service.go
@cmd/server/main.go
</context>

<tasks>

<task type="auto">
  <name>Task 1: Fix multi-queue running state</name>
  <files>internal/queue/queue.go</files>
  <action>
The shared `running bool` flag prevents multiple queues from starting. When `q.Start(ctx, "default")` sets `running = true`, the subsequent `q.Start(ctx, "ai")` returns early because it sees `running == true`.

Fix:
1. Change Queue struct field `running bool` to `running map[string]bool`
2. Initialize in New(): `running: make(map[string]bool)`
3. Update Start() to check/set `q.running[queueName]` instead of `q.running`
4. Update Stop() to:
   - Accept optional queueName parameter OR
   - Track all active queues and stop them all
   - Better: Change to stop all running queues (iterate over running map)
5. Worker shutdown: Each worker already receives queueName, so stop channel approach still works. But we need per-queue stop channels.

Recommended approach - use per-queue stop channels:
```go
type Queue struct {
    ...
    running  map[string]bool
    stopChs  map[string]chan struct{}  // per-queue stop channels
}
```

In Start():
```go
q.mu.Lock()
if q.running[queueName] {
    q.mu.Unlock()
    return
}
q.running[queueName] = true
stopCh := make(chan struct{})
q.stopChs[queueName] = stopCh
q.mu.Unlock()
```

Pass stopCh to workers. In Stop(), close all stop channels.

This maintains backward compatibility - Stop() still stops everything.
  </action>
  <verify>
Check build log: `cat ./tmp/air-combined.log | tail -50`
Look for "queue starting" logs for BOTH "default" and "ai" queues in server output.
  </verify>
  <done>Both default and AI queues start workers. Log shows "queue starting" for both queue names.</done>
</task>

<task type="auto">
  <name>Task 2: Replace magic strings with constants</name>
  <files>internal/processing/status.go, internal/processing/processor.go, internal/processing/ai_processor.go</files>
  <action>
Add status constants to status.go (this file already has StatusUpdate struct):

```go
// Processing status constants
const (
    StatusPending      = "pending"
    StatusProcessing   = "processing"
    StatusCompleted    = "completed"
    StatusFailed       = "failed"
    StatusAIProcessing = "ai_processing"
    StatusAIComplete   = "ai_complete"
    StatusAIFailed     = "ai_failed"
)
```

Note: QueueDefault and JobTypeProcess already exist in internal/document/document.go.
QueueAI and JobTypeAI already exist in ai_processor.go - these are fine where they are.

Update processor.go:
- Line 81: `"processing"` -> `StatusProcessing`
- Line 82: `"default"` -> `document.QueueDefault` (add import)
- Line 192-196: `"completed"` -> `StatusCompleted`, `"default"` -> `document.QueueDefault`
- Line 259-264: `"failed"` -> `StatusFailed`, `"default"` -> `document.QueueDefault`

Update ai_processor.go:
- Line 58-59: `"ai_processing"` -> `StatusAIProcessing`
- Line 72-76: `"ai_failed"` -> `StatusAIFailed`
- Line 92-95: `"ai_complete"` -> `StatusAIComplete`
  </action>
  <verify>
Check build log: `cat ./tmp/air-combined.log | tail -50`
Grep for remaining magic strings: `grep -n '"pending"\|"processing"\|"completed"\|"failed"\|"ai_' internal/processing/*.go`
  </verify>
  <done>No magic status strings remain in processing package. All use constants from status.go or document package.</done>
</task>

<task type="auto">
  <name>Task 3: Add inbox error directory visibility</name>
  <files>internal/inbox/service.go, templates/pages/admin/inboxes.templ</files>
  <action>
Add method to inbox service to count error files:

```go
// CountErrorFiles returns the number of files in an inbox's error directory.
func (s *Service) CountErrorFiles(inbox *sqlc.Inbox) (int, error) {
    errorPath := s.getErrorPath(inbox)
    entries, err := os.ReadDir(errorPath)
    if err != nil {
        if os.IsNotExist(err) {
            return 0, nil
        }
        return 0, err
    }

    count := 0
    for _, entry := range entries {
        if !entry.IsDir() && isPDFFilename(entry.Name()) {
            count++
        }
    }
    return count, nil
}
```

Note: Handler needs to pass error counts to template. Check if Inboxes template receives individual inbox objects or needs enriched data.

Looking at the template, it receives `[]sqlc.Inbox` directly. We need to either:
A) Create a wrapper type with inbox + error count, OR
B) Make error count a method call in template (not recommended - side effects in render)

Option A - Create InboxWithStats type in handler or a new view model:

In handler (documents.go or a new inboxes_handler.go):
```go
type InboxWithErrorCount struct {
    sqlc.Inbox
    ErrorCount int
}
```

Modify handler to:
1. Get inboxes from DB
2. For each inbox, call inboxSvc.CountErrorFiles()
3. Build []InboxWithErrorCount
4. Pass to template

Update inboxes.templ:
1. Change `Inboxes(inboxes []sqlc.Inbox)` to `Inboxes(inboxes []InboxWithErrorCount)` (or define type in template file)
2. In InboxCard, show error count badge next to error path if count > 0:

```templ
<div>
    <span class="text-muted-foreground">Error path:</span>
    <span class="font-mono text-xs">
        if inbox.ErrorPath != nil && *inbox.ErrorPath != "" {
            { *inbox.ErrorPath }
        } else {
            { inbox.Path }/errors
        }
        if inbox.ErrorCount > 0 {
            <span class="ml-2 inline-flex items-center rounded-full bg-destructive px-2 py-0.5 text-xs font-medium text-destructive-foreground">
                { strconv.Itoa(inbox.ErrorCount) } files
            </span>
        }
    </span>
</div>
```

Find the inbox handler (likely in handler/admin.go or handler/inboxes.go) to make changes.
  </action>
  <verify>
Check build log: `cat ./tmp/air-combined.log | tail -50`
Visit /admin/inboxes in browser - should show error file count badges.
  </verify>
  <done>Inbox cards display error file count badge when error directory contains files.</done>
</task>

</tasks>

<verification>
1. `cat ./tmp/air-combined.log | tail -100` - no compilation errors
2. Server log shows both "queue starting" messages for "default" and "ai" queues
3. `grep -rn '"pending"\|"processing"\|"completed"\|"failed"' internal/processing/` returns no results (except in comments)
4. Visit /admin/inboxes and verify error count badges appear
</verification>

<success_criteria>
- AI queue workers start alongside default queue workers
- All processing status strings use constants
- Inbox management page shows error file counts for each inbox
- No compilation errors
</success_criteria>

<output>
After completion, create `.planning/quick/001-fix-pending-bugs/001-SUMMARY.md`
</output>
