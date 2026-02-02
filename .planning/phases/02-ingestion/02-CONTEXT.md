# Phase 2: Ingestion - Context

**Gathered:** 2026-02-02
**Status:** Ready for planning

<domain>
## Phase Boundary

Users can add PDF documents via web upload (drag-and-drop) and automated local inbox watching. Duplicate detection prevents storing the same document twice. Text extraction, viewing, and organization are separate phases.

</domain>

<decisions>
## Implementation Decisions

### Upload area design
- Full-page drop zone — drag anywhere on page, overlay appears
- Progress bar per file for multi-file uploads
- Upload all files simultaneously (parallel uploads)
- Toast notification on completion, upload area resets

### Post-import handling
- Delete inbox files after successful import
- Near-instant detection using OS-level file watcher
- Failed files (corrupt, not PDF) moved to error folder (inbox/errors/)
- Process existing files on service startup

### Duplicate presentation
- Web upload: Block and show link to existing document
- Inbox: Delete silently, log the occurrence
- Duplicate detection by content hash only (SHA-256)
- Show skipped duplicates log in UI for visibility

### Configuration UX
- Inbox path configurable via both env var and UI (env as default, UI override)
- Multiple inbox directories supported
- Individual inboxes can be enabled/disabled via toggle
- Status indicator per inbox (green/red showing accessibility and watcher health)

### Claude's Discretion
- Exact overlay appearance for drop zone
- Progress bar styling and animation
- Error folder naming convention
- Duplicates log page design and filtering

</decisions>

<specifics>
## Specific Ideas

No specific requirements — open to standard approaches

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 02-ingestion*
*Context gathered: 2026-02-02*
