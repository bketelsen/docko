# Docko

## What This Is

Docko is a PDF document management system for a household/small team managing tens of thousands of documents. It provides automated ingestion from local directories and network shares (SMB/NFS), a processing pipeline for full-text indexing and AI-powered tagging, and a search interface to find any document by content, tags, or correspondent.

## Core Value

Find any document instantly AND automate the tagging/filing that's currently manual.

## Requirements

### Validated

<!-- Shipped and confirmed valuable. -->

- ✓ Admin authentication with session cookies — existing
- ✓ PostgreSQL database with migrations — existing
- ✓ Go + Echo + Templ web framework — existing
- ✓ Tailwind CSS + HTMX + templUI components — existing
- ✓ Structured logging with slog — existing

### Active

<!-- Current scope. Building toward these. -->

**Document Store:**
- [ ] Document store directory structure (inbox/originals/documents/thumbnails)
- [ ] Document metadata database (UUID, original name, file size, page count)
- [ ] Document thumbnail generation

**Document Sources:**
- [ ] Local inbox directory watching
- [ ] SMB network share integration (via Go libraries)
- [ ] NFS network share integration (via Go libraries)
- [ ] Per-source duplicate handling configuration (delete vs rename)

**Queue System:**
- [ ] Queue infrastructure with configurable concurrency
- [ ] Dashboard showing pending/completed counts per queue
- [ ] Per-document audit trail (every processing step logged)
- [ ] Automatic retry with exponential backoff on failures

**Processing Pipeline:**
- [ ] Ingestion queue (UUID assignment, copy to originals)
- [ ] Duplicate detection queue
- [ ] Full-text extraction and PostgreSQL indexing
- [ ] AI tagging queue (flexible provider, configurable page limit)
- [ ] Correspondent detection queue (best-effort deduplication)

**Tags & Correspondents:**
- [ ] Tag management (view, create, edit, delete, merge)
- [ ] Correspondent management (view, create, edit, merge)
- [ ] Manual tag assignment to documents
- [ ] Manual correspondent assignment to documents

**Search & Browse:**
- [ ] Simple search box (searches everything)
- [ ] Faceted filters (tags, correspondents, date ranges)
- [ ] Search results with document previews
- [ ] Document detail view with metadata

**Document Viewing:**
- [ ] In-browser PDF preview
- [ ] PDF download

### Out of Scope

<!-- Explicit boundaries. Includes reasoning to prevent re-adding. -->

- OCR processing — PDFs already have text content; defer OCR for scanned documents to v2
- Mobile app — Web-first, responsive design sufficient for v1
- Multi-tenant — Single household/team deployment only
- Real-time collaboration — Not needed for document archive use case
- Document editing — This is a read-only archive, not an editor
- Cloud storage backends (S3, GCS) — Local and network shares only for v1

## Context

**Technical environment:**
- Existing Go web application with Echo, Templ, HTMX, Tailwind
- PostgreSQL database with pgx/v5 driver and sqlc for type-safe queries
- Admin authentication already implemented with session cookies
- templUI components available for consistent UI
- Air hot-reload development workflow established

**User context:**
- Tens of thousands of PDFs across local drives and network shares
- Small team (household) of trusted users
- Need both search capability AND processing automation
- Network shares must work without requiring OS-level mounts

**Known challenges:**
- SMB/NFS integration in Go requires careful library selection
- Full-text search at scale needs proper PostgreSQL indexing
- AI tagging costs need management (page limits, provider flexibility)
- Correspondent deduplication is inherently fuzzy (best-effort acceptable)

## Constraints

- **Tech stack**: Go + Echo + Templ + HTMX + Tailwind — continue existing patterns
- **Database**: PostgreSQL only — use built-in full-text search, no Elasticsearch
- **Network access**: SMB/NFS via Go libraries — no OS mount requirements
- **AI provider**: Flexible (OpenAI, Claude, etc.) — not locked to one vendor
- **Deployment**: Single-server deployment — no distributed architecture needed

## Key Decisions

<!-- Decisions that constrain future work. Add throughout project lifecycle. -->

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| PostgreSQL full-text search | Avoids additional infrastructure (Elasticsearch), sufficient for tens of thousands of docs | — Pending |
| File-based document store | Simple, backup-friendly, avoids object storage complexity | — Pending |
| SMB/NFS via Go libraries | User shouldn't need to configure OS mounts | — Pending |
| Flexible AI provider | Avoid vendor lock-in, allow cost optimization | — Pending |
| Queue per processing step | Allows independent scaling, better visibility, easier debugging | — Pending |
| UUID-based filenames | Decouples storage from metadata, handles duplicates cleanly | — Pending |

---
*Last updated: 2026-02-02 after initialization*
