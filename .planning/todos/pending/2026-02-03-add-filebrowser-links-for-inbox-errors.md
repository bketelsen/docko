---
created: 2026-02-03T21:30
title: Add filebrowser links for inbox error directories
area: ui
files:
  - internal/inbox/service.go:439
  - internal/config/config.go:29
  - templates/pages/inboxes.templ
---

## Problem

Each inbox has an `errors` subdirectory (configurable via `INBOX_ERROR_SUBDIR`, default: "errors") where failed import files are moved. Currently there's no easy way for users to:

1. See that error files exist for an inbox
2. Navigate to the error directory to inspect/retry failed files

Users need to manually navigate the filesystem to find `{inbox_path}/errors/` directories.

**Relevant code:**
- `internal/inbox/service.go:439` - `getErrorPath()` returns `filepath.Join(inbox.Path, s.cfg.Inbox.ErrorSubdir)`
- `internal/config/config.go:29` - `ErrorSubdir` config with default "errors"
- Files are moved to error path on import failure (service.go:411)

## Solution

In the inboxes list UI:
1. Show error count badge if `{inbox_path}/errors/` contains files
2. Add a "View Errors" link/button that opens filebrowser to that directory
3. Consider adding a "Retry All" action to re-process error files

Could integrate with existing filebrowser if deployed, or add a simple file listing modal.
