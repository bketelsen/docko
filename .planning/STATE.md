# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-02-02)

**Core value:** Find any document instantly AND automate the tagging/filing that's currently manual
**Current focus:** Phase 13 - Environment Configuration Verification

## Current Position

Phase: 13 of 13 (Envrc Verification)
Plan: 0 of ? in current phase
Status: Not started
Last activity: 2026-02-04 - Added Phase 13 (Envrc Verification)

Progress: [################################################] 54/54 plans complete (Phase 13 pending)

## Performance Metrics

**Velocity:**
- Total plans completed: 54
- Average duration: 3.9 min
- Total execution time: 3.8 hours

**By Phase:**

| Phase | Plans | Total | Avg/Plan |
|-------|-------|-------|----------|
| 01-foundation | 3 | 10 min | 3.3 min |
| 02-ingestion | 5 | 39 min | 7.8 min |
| 03-processing | 5 | 28 min | 5.6 min |
| 04-viewing | 3 | 10 min | 3.3 min |
| 05-organization | 5 | 29 min | 5.8 min |
| 06-search | 3 | 11 min | 3.7 min |
| 07-network-sources | 6 | 19 min | 3.2 min |
| 08-ai-integration | 6 | 33 min | 5.5 min |
| 09-minimum-words | 2 | 7 min | 3.5 min |
| 10-templui-refactor | 7 | 39 min | 5.6 min |
| 11-dashboard | 3 | 7 min | 2.3 min |
| 12-queues-detail | 5 | 14 min | 2.8 min |

**Recent Trend:**

- Last 5 plans: 12-01 (3 min), 12-02 (1 min), 12-03 (2 min), 12-04 (3 min), 12-05 (5 min)
- Trend: Phase 12 complete - SSE live updates for queue activity

*Updated after each plan completion*

## Accumulated Context

### Decisions

Decisions are logged in PROJECT.md Key Decisions table.
Recent decisions affecting current work:

- Use gen_random_uuid() over uuid_generate_v4() for UUID generation
- 5-minute visibility timeout for job queue processing
- 2-level UUID sharding (ab/c1/uuid.ext) for storage paths
- One correspondent per document (1:1 relationship)
- Full jitter formula: random(0, min(cap, base * 2^attempt)) for retry backoff
- Default 4 workers per queue with 1-second poll interval
- Copy file first, then check for duplicate (avoids holding file in memory)
- Multiple inboxes in database (not config file) for UI management
- duplicate_action enum per inbox (delete/rename/skip)
- INBOX_PATH env var optional for default inbox
- 500ms debounce delay for file watcher events
- 4 concurrent inbox workers via semaphore
- PDF validation via magic bytes before ingestion
- Inbox watcher runs in background goroutine with cancellable context
- HTMX partial updates for inbox toggle and delete operations
- OCRmyPDF runs as persistent Docker service (like postgres) with inotify watcher
- App communicates with OCR via shared volumes (ocr-input, ocr-output)
- Thumbnails generated in app container, OCR in separate service
- ThumbnailPath returns .webp extension to match generated thumbnails
- 2-minute timeout prevents hanging on corrupt PDFs
- Placeholder fallback for unrenderable PDFs instead of error
- Bind mounts for OCR volumes (storage/ocr-input, storage/ocr-output) instead of Docker named volumes
- 100-char threshold to determine if embedded text is sufficient for search
- 5-minute timeout for OCR processing, 500ms polling interval
- Queue workers start on startup after handler registration
- Quarantine returns nil so job is marked completed (failure handled gracefully)
- SSE sends HTML partials (not JSON) for HTMX sse-swap compatibility
- 30-second heartbeat keeps SSE connections alive
- 100 subscriber limit for StatusBroadcaster
- docSvc.FileExists helper wraps storage.FileExists for handler access
- ServeThumbnail checks ThumbnailGenerated flag before attempting to serve
- Text extraction status shown instead of OCR status (TextContent field available)
- Storage path not displayed (computed dynamically, not stored)
- PDF.js 4.x legacy build for non-module script compatibility
- HTMX beforeend swap to append modal to body
- Canvas-based rendering with devicePixelRatio support for high-DPI
- Notes column nullable TEXT for optional correspondent info
- Modal dialog pattern with JavaScript open/close and HTMX form submission
- Document count badge shows association impact before delete
- Merge uses database transaction for atomicity
- Notes from merged correspondents prefixed with source name
- Merge mode shows only when 2+ correspondents exist
- Target selection prevents merging target into itself
- HX-Target header detection for inline vs full picker response
- Batch fetch tags with GetTagsForDocuments for list view efficiency
- JavaScript onclick toggle for inline dropdown (simpler than Alpine.js)
- Same picker pattern for correspondents as tags (consistency)
- ListDocumentsWithCorrespondent uses LEFT JOIN for efficient list query
- Generated STORED tsvector column for auto-updating search vector
- websearch_to_tsquery for safe user input handling (no syntax errors)
- Boolean flag pattern for optional sqlc filters (has_X + X_value)
- Tag filter uses AND logic (must have ALL selected tags)
- HX-Request header detection for partial vs full page responses
- Date range presets (today, 7d, 30d, 1y) instead of date pickers
- SearchResult wraps sqlc.SearchDocumentsRow directly (no manual mapping)
- Fetch filter options only on full page load (optimization for HTMX partials)
- 500ms debounce on search input for optimal UX
- Reuse duplicate_action enum from inboxes for network sources
- SHA-256 key derivation from SESSION_SECRET for credential encryption
- Network sources start disabled by default until tested
- Connect per operation for SMB (connections go stale after 10-15 min idle)
- 30-second connection timeout for SMB dial
- fs.WalkDir with io/fs interface for SMB directory walking
- NFS uses copy-then-delete for MoveFile (go-nfs-client lacks Rename)
- AUTH_UNIX with uid/gid 0 for NFS authentication (host-based)
- NewSourceFromConfig factory creates protocol-specific NetworkSource
- 5 consecutive failures auto-disables network source
- 5-minute polling interval for continuous sync sources
- Temp file download approach for network files (same as inbox)
- Post-import actions: leave, delete, or move to subfolder
- Follow inbox handler pattern for network sources UI consistency
- Sync now button only shown for enabled sources
- Network service lifecycle mirrors inbox service pattern
- HTMX toast feedback for async network operations
- Singleton pattern with CHECK(id=1) for ai_settings table
- DECIMAL(3,2) for AI confidence scores (0.00-1.00 range)
- Separate suggestion_type enum (tag/correspondent)
- Partial index on status='pending' for efficient pending suggestion queries
- GPT-4o-mini for OpenAI (cost-effective for tagging)
- Claude Haiku 4.5 for Anthropic (fastest/cheapest Claude)
- llama3.2 default for Ollama (configurable via OLLAMA_MODEL)
- Provider interface: Analyze/Name/Available methods
- Structured JSON output via OpenAI schema, prompt instructions for Anthropic/Ollama
- Fallback tries all providers in order (OpenAI -> Anthropic -> Ollama)
- Auto-apply creates tags/correspondents if not found (not just assigns existing)
- AI queue runs as separate queue (ai) from document processing (default)
- Provider status shows available vs not configured based on env vars
- Settings form uses HTMX POST with redirect and toast feedback
- Usage stats wraps sqlc query to handle nullable int64 fields
- Queue stats use GROUP BY aggregation for efficient counting
- HTMX outerHTML swap returns empty string to remove rows
- ApplySuggestionManual uses transaction for atomic tag/correspondent creation
- AI suggestions displayed in Overview tab below correspondent picker
- Re-analyze deletes existing pending suggestions before queuing new job
- AI auto-processing enqueues job after document processing commit
- Default min_word_count 0 = disabled (no minimum word count enforced)
- Word count check AFTER text extraction, BEFORE thumbnail (need text to count, avoid wasted work)
- CollapsibleIcon mode for sidebar collapse to icon-only view (Phase 10)
- templUI icon component for all navigation and button icons (Phase 10)
- Sidebar Trigger component handles both mobile sheet and desktop collapse (Phase 10)
- Use native select with templUI-consistent styling (selectbox requires complex HTMX setup)
- NoTogglePassword: true for network source password field
- Replace hard-coded gray-* colors with theme variables (foreground, muted-foreground, bg-card)
- Keep JavaScript modal open/close logic compatible with templUI dialog data attributes (Phase 10)
- Use HideCloseButton with custom Cancel button in dialog footer for consistency (Phase 10)
- Style native select elements to match templUI input component styling (Phase 10)
- Badge variants mapped to status: pending=Secondary, processing=Default+animate-pulse, completed=green custom, failed=Destructive (Phase 10)
- Table cells use CellProps{Class} for muted-foreground styling (Phase 10)
- SSE swap targets preserved inside table.Cell elements (Phase 10)
- templUI alert with Title+Description for error messages (Phase 10)
- Keep toggle switches as custom buttons (templUI lacks switch component) (Phase 10)
- Use bg-input for disabled toggle state instead of gray (Phase 10)
- Keep StatIcon template as-is (SVG icons for stat cards remain inline) (Phase 10)
- Use card.HeaderProps/ContentProps for custom layout variations (Phase 10)
- Use templUI button Href prop for pagination links (Phase 10)
- Card stat pattern: Header with row flex, Content with pt-0 (Phase 10)
- Table in card: Content with p-0 for edge-to-edge table (Phase 10)
- PostgreSQL FILTER clause for efficient conditional aggregation (Phase 11)
- Cast all COUNT results to int for consistent int32 Go types (Phase 11)
- Nested struct types in DashboardData for clean section organization (Phase 11)
- Graceful error handling with defaults for dashboard queries (Phase 11)
- Queue health status: issues if failed>0, warning if pending>=10 (Phase 11)
- DashboardData struct in template package for cleaner imports (Phase 11)
- clickableStatCard helper with optional value class for colored text (Phase 11)
- healthBadge component with healthy/warning/issues variants (Phase 11)
- statusDot for enabled/disabled visual indicator (Phase 11)
- Collapsible over accordion for multi-section open capability (Phase 12)
- LEFT JOIN LATERAL for safe JSONB payload extraction in job queries (Phase 12)
- dismissed status preserves audit trail while hiding from active lists (Phase 12)
- Queue-specific bulk operations use POST with :name parameter (Phase 12)
- Dismiss handler returns empty string for outerHTML swap removal (Phase 12)
- Lazy loading via hx-trigger='intersect once' for single fetch on expand (Phase 12)
- Chevron rotation via data-tui-collapsible-state attribute (Phase 12)
- SSE queue events use afterbegin swap to prepend new activity rows (Phase 12)
- Collapsible Script() required for templUI collapsible click handling (Phase 12)

### Pending Todos

1. **Add filebrowser links for inbox error directories** (ui) - Show error count and link to errors subdirectory
2. **Add processing progress visibility** (queue) - Show current step, detect stuck jobs
3. **Fix tags page edit button not working** (ui) - Edit button does nothing when clicked

### Quick Tasks Completed

| # | Description | Date | Commit | Directory |
|---|-------------|------|--------|-----------|
| 001 | Fix AI queue workers + magic strings | 2026-02-04 | b222f88 | [001-fix-pending-bugs](./quick/001-fix-pending-bugs/) |

### Blockers/Concerns

None.

## Session Continuity

Last session: 2026-02-04T21:45:00Z
Stopped at: Completed quick task 001 - Fix AI queue workers + magic strings
Resume file: None

### Roadmap Evolution

- Phase 9 added: Minimum number of words import block
- Phase 10 added: Refactor to use more templUI components
- Phase 11 added: Dashboard with stats, counts, and navigation links
- Phase 12 added: Queues route with expanders for failed jobs and recent activity
- Phase 13 added: Verify all settings are reflected in .envrc and .envrc.example

---
*54 plans executed across 12 phases - Phase 13 added*
