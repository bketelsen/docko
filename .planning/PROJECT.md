# Docko

## What This Is

Docko is a PDF document management system for a household/small team managing tens of thousands of documents. It provides automated ingestion from local directories and network shares (SMB/NFS), a processing pipeline for full-text indexing and AI-powered tagging, and a search interface to find any document by content, tags, or correspondent.

## Core Value

Find any document instantly AND automate the tagging/filing that's currently manual.

## Current State

**Version:** v1.0 shipped 2026-02-04
**Codebase:** ~59,000 lines Go/Templ across 335 files
**Tech stack:** Go + Echo + Templ + HTMX + Tailwind + PostgreSQL

v1.0 delivers the complete document management system:
- Multi-source ingestion (web upload, inbox watching, SMB/NFS)
- Full-text search with filtering
- AI-powered auto-tagging (OpenAI, Anthropic, Ollama)
- Production-ready with Docker Compose deployment

## Requirements

### Validated

<!-- Shipped and confirmed valuable in v1.0 -->

**Foundation:**
- Admin authentication with session cookies — v1.0
- PostgreSQL database with migrations — v1.0
- Go + Echo + Templ web framework — v1.0
- Tailwind CSS + HTMX + templUI components — v1.0
- Structured logging with slog — v1.0

**Document Ingestion:**
- Upload PDF files via web UI with drag-and-drop — v1.0
- Bulk upload (multiple files at once) — v1.0
- Local inbox directory watching — v1.0
- SMB network share integration — v1.0
- NFS network share integration — v1.0
- Duplicate detection by content hash (SHA-256) — v1.0
- Per-source duplicate handling configuration — v1.0

**Document Store:**
- UUID-based organized directory structure — v1.0
- Original files preserved unmodified — v1.0
- Document metadata in database — v1.0

**Tags & Correspondents:**
- Tag CRUD and document assignment — v1.0
- Correspondent CRUD, assignment, and merge — v1.0

**Search & Retrieval:**
- Full-text search with PostgreSQL GIN indexes — v1.0
- Filter by tags, correspondent, date range — v1.0

**Document Viewing:**
- In-browser PDF preview — v1.0
- PDF download — v1.0
- Thumbnail previews (WebP) — v1.0

**Processing Pipeline:**
- Queue-based architecture with SKIP LOCKED — v1.0
- Text extraction with OCR fallback — v1.0
- Queue dashboard with retry capability — v1.0
- Full audit trail — v1.0

**AI Features:**
- Multi-provider LLM integration (OpenAI, Anthropic, Ollama) — v1.0
- Auto-tagging with confidence thresholds — v1.0
- Correspondent detection — v1.0
- Cost control (max pages setting) — v1.0

**Admin:**
- Document source configuration — v1.0
- System status and queue health monitoring — v1.0

### Active

<!-- Current scope. Building toward these. -->

*No active requirements — planning next milestone*

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
- Go web application with Echo, Templ, HTMX, Tailwind
- PostgreSQL database with pgx/v5 driver and sqlc for type-safe queries
- Admin authentication with session cookies
- templUI components for consistent UI
- Air hot-reload development workflow
- Docker Compose for development and production

**User context:**
- Tens of thousands of PDFs across local drives and network shares
- Small team (household) of trusted users
- Need both search capability AND processing automation
- Network shares work without requiring OS-level mounts

**Known technical debt:**
- selectbox component installed but not used in 3 pages (visual inconsistency)
- AI queue timing note requires production monitoring

## Constraints

- **Tech stack**: Go + Echo + Templ + HTMX + Tailwind — continue existing patterns
- **Database**: PostgreSQL only — use built-in full-text search, no Elasticsearch
- **Network access**: SMB/NFS via Go libraries — no OS mount requirements
- **AI provider**: Flexible (OpenAI, Claude, Ollama) — not locked to one vendor
- **Deployment**: Single-server deployment — no distributed architecture needed

## Key Decisions

<!-- Decisions that constrain future work. Add throughout project lifecycle. -->

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| PostgreSQL full-text search | Avoids additional infrastructure (Elasticsearch), sufficient for tens of thousands of docs | Good |
| File-based document store | Simple, backup-friendly, avoids object storage complexity | Good |
| SMB/NFS via Go libraries | User shouldn't need to configure OS mounts | Good |
| Flexible AI provider | Avoid vendor lock-in, allow cost optimization | Good |
| Queue per processing step | Allows independent scaling, better visibility, easier debugging | Good |
| UUID-based filenames | Decouples storage from metadata, handles duplicates cleanly | Good |
| 2-level UUID sharding | Prevents too many files in single directory | Good |
| PostgreSQL-backed queue | Simpler than Redis, atomic with document transactions | Good |
| OCRmyPDF as Docker service | Persistent service with inotify, avoids spawning processes | Good |
| templUI components | Consistent UI, dark mode support, faster development | Good |

---
*Last updated: 2026-02-04 after v1.0 milestone*
