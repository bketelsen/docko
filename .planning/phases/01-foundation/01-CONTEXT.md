# Phase 1: Foundation - Context

**Gathered:** 2026-02-02
**Status:** Ready for planning

<domain>
## Phase Boundary

Establish reliable document storage structure and job queue processing infrastructure. This phase creates the foundation that all other phases build on — file organization, database schema for documents, queue system for async processing, and audit logging.

</domain>

<decisions>
## Implementation Decisions

### Storage organization
- Nested by UUID prefix with 2 levels: `ab/c1/abc123.pdf`
- Parallel structure for derived files: `originals/ab/c1/`, `thumbnails/ab/c1/`, `text/ab/c1/`
- Single `STORAGE_PATH` environment variable for root, subdirectories are fixed within it

### Queue behavior
- Database-backed queue using PostgreSQL (no Redis dependency)
- Configurable retry count with default of 3 attempts
- Exponential backoff with jitter between retries
- Failed jobs remain in queue table with 'failed' status for manual retry

### Audit trail design
- Stored in database table (document_events or similar)
- Log all processing steps: ingested, text_extracted, thumbnail_generated, etc.
- Include full details: error stack traces, job parameters, processing metrics
- Viewable in admin dashboard only (not per-document UI in this phase)

### Metadata scope
- Capture content hash (SHA256) for duplicate detection
- Extract PDF metadata when present: title, author, creation date
- Track both `created_at` (when ingested) and `document_date` (from PDF or user-set) separately
- Derive document status from audit events rather than explicit status field
- Create junction tables now for future phases (document_tags, document_correspondents) — empty but ready

### Claude's Discretion
- Specific table and column naming conventions
- Queue polling interval and worker count
- Exact audit event type names
- Error message formatting

</decisions>

<specifics>
## Specific Ideas

No specific requirements — open to standard approaches within the decisions above.

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 01-foundation*
*Context gathered: 2026-02-02*
